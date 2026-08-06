# 📄 Nghiệp vụ 4: Sổ Cái Ghi Đúp & Tự Động Đối Soát Tài Chính (Double-Entry Ledger & Financial Reconciliation Engine)

Tài liệu này mô tả chi tiết quy trình nghiệp vụ, sơ đồ Use Case, Activity, DFD, Component, Class/Struct và ERD cho phân hệ **Ghi sổ đúp (Double-Entry Ledger), Lưu vết biến động tài chính bất biến và Tự động đối soát tài chính (Automated Financial Reconciliation)**.

---

## 📑 1. Quy Trình Nghiệp Vụ Chi Tiết (Business Workflow)

1. **Nguyên tắc Ghi Sổ Đúp Bất Biến (Immutable Accounting Double-Entry)**:
   - Hệ thống **TUYỆT ĐỐI KHÔNG** sử dụng câu lệnh `UPDATE balance SET amount = ...` trực tiếp mà không kèm theo dòng ghi log sổ cái.
   - Mọi biến động tiền tệ (Nạp tiền, Rút tiền, Phí dịch vụ, Thưởng, Hoàn tiền) phải được ghi nhận dưới dạng một cặp giao dịch **Nợ (Debit)** và **Có (Credit)**:
     $$\sum \text{Debit Amount} \quad \equiv \quad \sum \text{Credit Amount}$$
2. **Luồng Ghi nhận Sổ cái (Ledger Writing Workflow)**:
   - Khi có bất kỳ giao dịch Nạp/Rút thành công, `payment-engine` tự động sinh ra một `Ledger Transaction Entry` bao gồm 2 hoặc nhiều dòng `ledger_entries`:
     - *Ví dụ Luồng Nạp 100 USDT (Phí 1%)*:
       - `Debit` Tài khoản Tiền Nguồn Hệ thống (System Clearing Account): +100 USDT.
       - `Credit` Tài khoản Ví Khả Dụng Merchant (Merchant Available Account): +99 USDT.
       - `Credit` Tài khoản Doanh Thu Phí Hệ thống (Platform Revenue Account): +1 USDT.
3. **Đồng bộ Dữ liệu Kiểm toán quy mô lớn (CDC Pipeline to ClickHouse)**:
   - Toàn bộ các dòng `ledger_entries` mới ghi vào PostgreSQL sẽ được **Debezium CDC (Change Data Capture)** tự động bắt từ Postgres Write-Ahead Log (WAL) và stream sang **ClickHouse / BigQuery (OLAP Database)**.
   - Giúp việc lưu trữ hàng trăm triệu dòng sổ cái không làm chậm DB giao dịch chính (OLTP Postgres).
4. **Quy trình Tự động Đối soát Tài chính (Automated Reconciliation Engine)**:
   - Microservice `audit-ledger` tự động kích hoạt Cronjob chạy **5 phút/lần**:
     - **Bước 1**: Tính tổng số tiền thực tế đang lưu trữ trên tất cả các Ví On-chain & Tài khoản Ngân hàng tổng ($V_{\text{real}}$).
     - **Bước 2**: Tính tổng số dư của tất cả Merchant + Số dư Phí hệ thống ghi nhận trong Database ($V_{\text{system}}$).
     - **Bước 3**: So sánh sai lệch $\Delta = |V_{\text{real}} - V_{\text{system}}|$.
   - **Xử lý Sai lệch (Discrepancy Handling)**:
     - Nếu $\Delta = 0$: Hệ thống an toàn (Status `RECONCILED_MATCH`).
     - Nếu $\Delta > 0$ (Sai lệch dù chỉ $0.01):
       - Chuyển trạng thái hệ thống sang `RECONCILED_MISMATCH` (Alert Cấp độ Đỏ).
       - Tự động kích hoạt công tắc khẩn cấp (Emergency Circuit Breaker): **Tạm dừng tính năng Rút tiền tự động**.
       - Gửi Cảnh báo khẩn cấp Telegram/Email cho Chủ doanh nghiệp & Trưởng phòng Kế toán.

---

## 🎯 2. Sơ Đồ Use Case (Use Case Diagram)

```mermaid
graph TD
    subgraph "System Engine"
        SYS((Payment Engine)) --> UC1[Write Immutable Double-Entry Ledger]
    end

    subgraph "Audit & Reconciliation Engine"
        AUD((Audit Worker)) --> UC2[Capture CDC Log & Stream to ClickHouse]
        AUD --> UC3[Run Periodic 5-Min Balance Reconciliation]
        AUD --> UC4[Trigger Emergency Circuit Breaker on Discrepancy]
    end

    subgraph "Finance Manager & Auditor"
        FM((Finance Manager)) --> UC5[View Financial Balance Sheets & Audit Logs]
        FM --> UC6[Manual Reconciliation & Adjust Discrepancy]
        FM --> UC7[Release Emergency Circuit Breaker]
    end
```

---

## 🔄 3. Sơ Đồ Hoạt Động (Activity Diagram)

```mermaid
stateDiagram-v2
    [*] --> CronTrigger: Cronjob Kích hoạt (Mỗi 5 Phút)
    CronTrigger --> FetchOnchainBalance: Đọc Số dư Thực tế từ Blockchain Nodes & Bank API (V_real)
    FetchOnchainBalance --> CalculateSystemBalance: Tính Tổng Số dư Merchant + Fee trong DB (V_system)
    
    CalculateSystemBalance --> CompareDelta: So sánh Delta = |V_real - V_system|
    
    CompareDelta --> ReconciledOK: Delta == 0 (Khớp Số Liệu)
    ReconciledOK --> SaveReportOK: Lưu Báo Cáo Đối Soát SUCCESS
    SaveReportOK --> [*]
    
    CompareDelta --> DiscrepancyDetected: Delta > 0 (Phát Hiện Sai Lệch)
    DiscrepancyDetected --> TriggerCircuitBreaker: BẬT Emergency Circuit Breaker (Tạm khóa Payout)
    TriggerCircuitBreaker --> SendEmergencyAlert: Gửi Thông Báo Khẩn (Telegram / Email / SMS)
    SendEmergencyAlert --> RequireHumanIntervention: Chờ Kế Toán & Risk Officer Kiểm Tra Thủ Công
    RequireHumanIntervention --> [*]
```

---

## 🌊 4. Sơ Đồ Luồng Dữ Liệu (Data Flow Diagram - DFD Level 1)

```mermaid
graph LR
    PaymentEngine[Payment Engine] -->|"1. Write Double-Entry Record"| PostgresDB[(PostgreSQL OLTP)]
    PostgresDB -->|"2. WAL Log Event"| Debezium[Debezium CDC Engine]
    Debezium -->|"3. Stream Ledger Entries"| NATSQueue[NATS JetStream Queue]
    NATSQueue -->|"4. Bulk Ingest"| ClickHouse[(ClickHouse OLAP Analytics)]
    
    AuditWorker[Reconciliation Worker] -->|"5a. Query Real Wallet Balance"| ExtNode[Blockchain Nodes & Bank API]
    AuditWorker -->|"5b. Query System Balance Sum"| PostgresDB
    AuditWorker -->|"6. Compare Balances"| ReconciliationEngine[Reconciliation Evaluator]
    
    ReconciliationEngine -->|"7. If Mismatch: Freeze Payouts"| RedisLock[(Redis Circuit Breaker Key)]
    ReconciliationEngine -->|"8. Generate Audit Report"| ClickHouse
```

---

## 🧩 5. Sơ Đồ Thành Phần (Component Diagram)

```mermaid
flowchart TD
    subgraph "Core Transaction Layer"
        LedgGen["Double-Entry Ledger Generator"]
        PgCoord["Postgres Transaction Coordinator"]
    end

    subgraph "CDC Pipeline Layer"
        Debezium["Debezium CDC Connector"]
        NatsIngest["NATS Ingestion Pipeline"]
    end

    subgraph "Analytics & Reconciliation Layer"
        BalScanner["Periodic Balance Scanner"]
        DiscrepEval["Discrepancy Evaluator"]
        CircuitBreaker["Emergency Circuit Breaker Switch"]
    end

    subgraph "Storage Layer"
        Postgres[("PostgreSQL (OLTP Ledgers)")]
        ClickHouse[("ClickHouse (OLAP Historical Audit)")]
        Redis[("Redis (Circuit Breaker Flag)")]
    end

    LedgGen --> PgCoord
    PgCoord --> Postgres
    Postgres --> Debezium
    Debezium --> NatsIngest
    NatsIngest --> ClickHouse
    BalScanner --> DiscrepEval
    DiscrepEval --> CircuitBreaker
    CircuitBreaker --> Redis
```

---

## 📐 6. Sơ Đồ Lớp / Struct Go (Class / Struct Diagram)

```mermaid
classDiagram
    class LedgerEntry {
        +int64 ID
        +string TransactionRefCode
        +int64 AccountID
        +string AccountType
        +string EntryType
        +decimal Amount
        +string Currency
        +time.Time CreatedAt
    }

    class ReconciliationReport {
        +int64 ID
        +time.Time CheckedAt
        +decimal TotalRealBalance
        +decimal TotalSystemBalance
        +decimal DiscrepancyDelta
        +string Status
        +string DiscrepancyDetails
    }

    class ReconciliationService {
        -postgresRepo BalanceRepository
        -clickhouseRepo AnalyticsRepository
        -blockchainClient NodeAdapter
        -redisClient RedisAdapter
        +RunReconciliationJob(ctx context.Context) (*ReconciliationReport, error)
        +TriggerEmergencyFreeze(ctx context.Context, reason string) error
    }

    ReconciliationService ..> LedgerEntry : reads
    ReconciliationService ..> ReconciliationReport : generates
```

---

## 🗄️ 7. Sơ Đồ Thực Thể Liên Kết (ERD - Entity Relationship Diagram)

```mermaid
erDiagram
    ACCOUNTS ||--o{ LEDGER_ENTRIES : "records"
    LEDGER_TRANSACTIONS ||--o{ LEDGER_ENTRIES : "contains"
    RECONCILIATION_REPORTS ||--o{ RECONCILIATION_LOGS : "logs"

    ACCOUNTS {
        bigint id PK
        bigint merchant_id FK
        string account_type
        string currency
        decimal current_balance
    }

    LEDGER_TRANSACTIONS {
        bigint id PK
        string ref_code UK
        string transaction_type
        timestamp created_at
    }

    LEDGER_ENTRIES {
        bigint id PK
        bigint ledger_transaction_id FK
        bigint account_id FK
        string entry_type
        decimal amount
        timestamp created_at
    }

    RECONCILIATION_REPORTS {
        bigint id PK
        timestamp checked_at
        decimal total_real_balance
        decimal total_system_balance
        decimal discrepancy_delta
        string status
    }
```
