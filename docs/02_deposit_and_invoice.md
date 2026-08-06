# 📄 Nghiệp vụ 2: Cổng Nạp Tiền & Xử Lý Hóa Đơn Tự Động (Deposit & Dynamic Invoice Gateway)

Tài liệu này mô tả chi tiết quy trình nghiệp vụ, sơ đồ Use Case, Activity, DFD, Component, Class/Struct và ERD cho phân hệ **Tạo Hóa đơn (Invoice Generation), Cấp phát địa chỉ ví/QR code linh hoạt và Xử lý Tiền vào (Deposit Ingestion) bất đồng bộ chống nạp trùng**.

---

## 📑 1. Quy Trình Nghiệp Vụ Chi Tiết (Business Workflow)

1. **Khởi tạo Hóa đơn Nạp tiền (Create Invoice Request)**:
   - Merchant App gọi gRPC/REST API `CreateInvoice(Amount, Currency, OrderReference)` kèm `Idempotency-Key` trên Header.
   - `payment-engine` kiểm tra `Idempotency-Key` trong Redis. Nếu key đã tồn tại ➔ Trả về kết quả Hóa đơn đã sinh trước đó (Ngăn chặn tạo hóa đơn trùng).
   - Nếu key chưa tồn tại ➔ Khởi tạo Hóa đơn ở trạng thái `PENDING` kèm thời gian hết hạn (`ExpiredAt` = 15-30 phút).
2. **Gán Phương thức Thanh toán (Address Pool / Dynamic QR Allocation)**:
   - **Với Fiat (VND)**: Hệ thống sinh mã VietQR động chứa mã `Memo Code` độc nhất (VD: `NEXUS-98123`).
   - **Với Crypto (USDT/USDC)**: 
     - *Phương án A*: Cấp phát 1 địa chỉ Ví tạm thời từ `Address Pool` sẵn có.
     - *Phương án B*: Dùng 1 địa chỉ Ví duy nhất kèm `Memo/Tag` định danh cho từng hóa đơn.
3. **Lắng nghe Sự kiện Tiền vào (On-chain / Banking Event Ingestion)**:
   - **Blockchain Listener Pod**: Lắng nghe sự kiện Transfer token trên BSC/Tron/Solana ➔ Publish Event `deposit.crypto.detected` vào **NATS JetStream**.
   - **Banking Webhook Pod**: Lắng nghe IPN từ cổng Ngân hàng ➔ Publish Event `deposit.fiat.detected` vào **NATS JetStream**.
4. **Xử lý Khớp Hóa đơn & Cộng Số dư (Idempotent Matching & Settlement)**:
   - Worker nhận Event từ NATS Queue.
   - **Idempotency Verification**: Kiểm tra `TxHash` (Crypto) hoặc `BankTransactionID` (Fiat) trong Redis. Nếu đã xử lý ➔ Drop Event.
   - **Matching Logic**: Tìm Hóa đơn `PENDING` có đúng `Memo Code` / `Deposit Address` và `Amount` thỏa mãn chênh lệch cho phép.
   - **Atomic Transaction Settlement**:
     - Mở PostgreSQL ACID Transaction (`pgx.Tx`).
     - Chuyển trạng thái Hóa đơn sang `SUCCESS`.
     - Cộng số dư khả dụng (`Available Balance`) cho Merchant trong bảng `balances`.
     - Thêm dòng Log Sổ cái vào bảng `ledger_entries`.
     - Commit Transaction.
5. **Kích hoạt Webhook Notification**:
   - Publish Event `invoice.settled` vào NATS để `webhook-engine` gửi thông báo về Server của Merchant.

---

## 🎯 2. Sơ Đồ Use Case (Use Case Diagram)

```mermaid
graph TD
    subgraph "Merchant Application"
        M((Merchant App)) --> UC1[Create Dynamic Invoice]
        M --> UC2[Query Invoice Status]
    end

    subgraph "End Customer"
        C((End Customer)) --> UC3[Scan VietQR / Send Crypto to Address]
    end

    subgraph "External Networks"
        BC((Blockchain Node)) --> UC4[Emit On-chain Transfer Event]
        BANK((Bank Gateway)) --> UC5[Emit Bank IPN Event]
    end

    subgraph "PayNexus Engine System"
        UC4 --> UC6[Ingest Deposit Event to NATS Queue]
        UC5 --> UC6
        UC6 --> UC7[Idempotent Matching & Settlement]
        UC7 --> UC8[Update Available Balance & Write Ledger]
        UC8 --> UC9[Trigger Async Webhook Delivery]
    end
```

---

## 🔄 3. Sơ Đồ Hoạt Động (Activity Diagram)

```mermaid
stateDiagram-v2
    [*] --> EventReceived: NATS Worker nhận Event Deposit (TxHash/BankID)
    EventReceived --> CheckRedisTx: Kiểm tra TxHash trong Redis
    
    CheckRedisTx --> DropEvent: TxHash đã tồn tại (Duplicate Event)
    DropEvent --> [*]
    
    CheckRedisTx --> FindMatchingInvoice: TxHash Chưa tồn tại (Event Mới)
    FindMatchingInvoice --> ValidateInvoice: Tìm Invoice PENDING theo Memo/Address
    
    ValidateInvoice --> MarkExpired: Invoice Hết Hạn / Không Tìm Thấy
    MarkExpired --> LogUnmatched: Ghi log Deposit Unmatched (Cần xử lý tay)
    LogUnmatched --> [*]
    
    ValidateInvoice --> BeginDBTx: Tìm Thấy Invoice PENDING Hợp Lệ
    BeginDBTx --> LockBalanceRow: Start Postgres ACID Tx (`pgx.Tx`)
    LockBalanceRow --> UpdateInvoiceStatus: Lock row `balances` theo MerchantID
    UpdateInvoiceStatus --> CreditBalance: Set Invoice Status = 'SUCCESS'
    CreditBalance --> InsertLedger: Available Balance += Invoice Amount
    InsertLedger --> CommitDBTx: Insert dòng Ledger Entry (Debit/Credit)
    
    CommitDBTx --> SaveRedisTx: Commit DB Transaction Thành Công
    SaveRedisTx --> PublishSettledEvent: Lưu TxHash vào Redis TTL 7 ngày
    PublishSettledEvent --> [*]: Publish Event `invoice.settled` vào NATS
```

---

## 🌊 4. Sơ Đồ Luồng Dữ Liệu (Data Flow Diagram - DFD Level 1)

```mermaid
graph LR
    Customer[End Customer] -->|"1. Transfer Crypto/Fiat"| ExtNetwork[Blockchain / Bank]
    ExtNetwork -->|"2. Event Notification"| Listener[Deposit Listener Service]
    Listener -->|"3. Publish Raw Event"| NATSQueue[NATS JetStream Queue]
    
    NATSQueue -->|"4. Consume Event"| DepositWorker[Deposit Settlement Worker]
    DepositWorker -->|"5. Check Duplicate Tx"| Redis[(Redis Cache)]
    DepositWorker -->|"6. Query Pending Invoice"| InvoiceDB[(Postgres Invoices Table)]
    
    DepositWorker -->|"7. Update Balance & Write Ledger"| LedgerDB[(Postgres Balances & Ledgers)]
    DepositWorker -->|"8. Publish Settled Event"| NATSQueue
    NATSQueue -->|"9. Dispatch Webhook"| WebhookEngine[Webhook Engine]
```

---

## 🧩 5. Sơ Đồ Thành Phần (Component Diagram)

```mermaid
flowchart TD
    subgraph "Ingress Layer"
        InvCtrl["Invoice Controller"]
        BCList["Blockchain Listener Pod"]
        BankIPN["Bank IPN Listener Pod"]
    end

    subgraph "Queue & Caching Layer"
        NATSQueue["NATS JetStream (deposit.events)"]
        RedisCache[("Redis (TxHash & Idempotency Lock)")]
    end

    subgraph "Settlement Engine Layer"
        DepMatch["Deposit Matching Processor"]
        BalMgr["Balance Manager"]
        LedgWriter["Ledger Writer"]
    end

    subgraph "Database Layer"
        Postgres[("PostgreSQL (Invoices & Balances)")]
    end

    InvCtrl --> Postgres
    BCList --> NATSQueue
    BankIPN --> NATSQueue
    NATSQueue --> DepMatch
    DepMatch --> RedisCache
    DepMatch --> BalMgr
    BalMgr --> LedgWriter
    LedgWriter --> Postgres
```

---

## 📐 6. Sơ Đồ Lớp / Struct Go (Class / Struct Diagram)

```mermaid
classDiagram
    class Invoice {
        +int64 ID
        +int64 MerchantID
        +string Code
        +string Currency
        +decimal Amount
        +string DepositAddress
        +string MemoCode
        +string Status
        +time.Time ExpiredAt
        +time.Time CreatedAt
    }

    class DepositEvent {
        +string TxHash
        +string Network
        +string DestinationAddress
        +string Memo
        +decimal Amount
        +time.Time BlockTimestamp
    }

    class DepositProcessor {
        -redisClient RedisAdapter
        -invoiceRepo InvoiceRepository
        -balanceRepo BalanceRepository
        -ledgerRepo LedgerRepository
        +ProcessDeposit(ctx context.Context, event DepositEvent) error
        +MatchInvoice(ctx context.Context, memo string) (*Invoice, error)
    }

    DepositProcessor ..> Invoice : updates
    DepositProcessor ..> DepositEvent : consumes
```

---

## 🗄️ 7. Sơ Đồ Thực Thể Liên Kết (ERD - Entity Relationship Diagram)

```mermaid
erDiagram
    MERCHANTS ||--o{ INVOICES : "creates"
    INVOICES ||--o| DEPOSIT_TRANSACTIONS : "settled_by"
    MERCHANTS ||--o{ BALANCES : "holds"

    INVOICES {
        bigint id PK
        bigint merchant_id FK
        string code UK
        string currency
        decimal amount
        string deposit_address
        string memo_code UK
        string status
        timestamp expired_at
        timestamp created_at
    }

    DEPOSIT_TRANSACTIONS {
        bigint id PK
        bigint invoice_id FK
        string tx_hash UK
        string network_type
        decimal actual_amount
        bigint block_number
        timestamp confirmed_at
    }

    BALANCES {
        bigint id PK
        bigint merchant_id FK
        string currency
        decimal available_balance
        decimal locked_balance
        timestamp updated_at
    }
```
