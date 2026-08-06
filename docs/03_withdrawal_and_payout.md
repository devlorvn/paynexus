# 📄 Nghiệp vụ 3: Yêu Cầu Rút Tiền, Khóa Số Dư & Duyệt Hạn Mức An Toàn (Automatic Withdrawal & Risk Threshold Payout)

Tài liệu này tả chi tiết quy trình nghiệp vụ, sơ đồ Use Case, Activity, DFD, Component, Class/Struct và ERD cho phân hệ **Chi trả / Rút tiền (Payout/Withdrawal Engine), Khóa số dư khả dụng chống Race-Condition và Duyệt hạn mức an toàn tự động/thủ công (Risk Threshold Approval)**.

---

## 📑 1. Quy Trình Nghiệp Vụ Chi Tiết (Business Workflow)

1. **Khởi tạo Yêu cầu Rút tiền (Withdrawal Request)**:
   - Merchant Admin gọi API `/v1.PaymentService/RequestWithdrawal` kèm `Amount`, `Currency`, `DestinationAddress/BankNo` và `Idempotency-Key`.
   - Gateway xác thực HMAC Signature và kiểm tra quyền `Withdraw` của API Key.
2. **Chống Race-Condition & Khóa Số dư Khả dụng (Balance Locking)**:
   - Worker mở **Redis Distributed Lock (`redlock`)** trên `Merchant_ID`: `lock:merchant:{id}:withdraw`.
   - Kiểm tra Số dư khả dụng (`Available Balance` >= `Amount` + `Fee`).
   - Mở PostgreSQL Transaction:
     - `Available Balance` = `Available Balance` - (`Amount` + `Fee`).
     - `Locked Balance` = `Locked Balance` + `Amount` + `Fee`.
     - Khởi tạo Lệnh Rút tiền (`Withdrawal Ticket`) ở trạng thái `PROCESSING`.
   - Mở Redis Distributed Lock.
3. **Phân Luồng Kiểm Soát Rủi Ro (Risk Threshold Approval Evaluation)**:
   - System kiểm tra số tiền rút so với Hạn mức An toàn (`Risk Threshold`, VD: $5,000 USD):
     - **Nếu Amount < $5,000**: Tự động duyệt ➔ Chuyển trạng thái sang `APPROVED_AUTO` và đẩy Event `payout.execute` vào **NATS JetStream Queue**.
     - **Nếu Amount >= $5,000**: Tự động giữ ➔ Chuyển trạng thái sang `PENDING_ADMIN_APPROVAL`. Gửi cảnh báo Telegram/Email cho Super Admin Risk Officer.
4. **Super Admin Duyệt / Từ chối (Manual Review Flow)**:
   - **Trường hợp Approve**: Admin bấm "Approve" trên Admin Portal ➔ Chuyển trạng thái sang `APPROVED_MANUAL` ➔ Đẩy Event `payout.execute` vào NATS.
   - **Trường hợp Reject**: Admin bấm "Reject" kèm lý do ➔ Chuyển trạng thái sang `REJECTED` ➔ **Hoàn tiền số dư**: `Locked Balance` trừ đi, `Available Balance` cộng lại nguyên vẹn.
5. **Thực thi Chuyển tiền & Tối ưu Phí Gas (On-chain / Banking Execution)**:
   - Worker nhận Event `payout.execute` từ NATS.
   - **Crypto Transaction Batching**: Gom nhiều lệnh rút nhỏ cùng mạng (VD: TRON/BSC) thành 1 giao dịch Multi-output để tiết kiệm 60% Phí Gas.
   - Ký giao dịch bằng Private Key từ KMS/Vault và phát sóng (Broadcast) lên Blockchain / Banking API.
6. **Xác nhận Hoàn tất & Ghi Sổ cái (Settlement & Webhook)**:
   - Khi On-chain Tx đạt đủ số Block Confirmation / Ngân hàng báo chuyển thành công:
     - `Locked Balance` trừ đi lượng tiền đã rút.
     - Chuyển trạng thái Ticket sang `SUCCESS`.
     - Ghi dòng Ledger Entry cho khoản phí dịch vụ và khoản tiền đã chi.
     - Gửi Webhook báo về cho Merchant.

---

## 🎯 2. Sơ Đồ Use Case (Use Case Diagram)

```mermaid
graph TD
    subgraph "Merchant Admin"
        MA((Merchant Admin)) --> UC1[Submit Withdrawal Request]
        MA --> UC2[View Withdrawal Status]
    end

    subgraph "Risk Officer / Super Admin"
        RA((Risk Officer)) --> UC3[Review Pending High-Value Payouts]
        RA --> UC4[Approve Payout Ticket]
        RA --> UC5[Reject Payout Ticket & Refund Balance]
    end

    subgraph "Payout Engine Worker"
        SYS((Payout Worker)) --> UC6[Acquire Redis Redlock & Deduct Balance]
        SYS --> UC7[Evaluate Risk Threshold Limit]
        SYS --> UC8[Batch Crypto Tx & Sign via KMS]
        SYS --> UC9[Settle Payout Ticket & Release Lock Balance]
    end
```

---

## 🔄 3. Sơ Đồ Hoạt Động (Activity Diagram)

```mermaid
stateDiagram-v2
    [*] --> RequestSubmitted: Merchant gửi Yêu cầu Rút tiền
    RequestSubmitted --> AcquireRedlock: Lấy Redis Distributed Lock (`redlock`)
    
    AcquireRedlock --> CheckAvailableBalance: Kiểm tra Available Balance >= Amount + Fee
    CheckAvailableBalance --> LockFailed: Số dư không đủ
    LockFailed --> ReleaseRedlock: Giải phóng Lock
    ReleaseRedlock --> RejectRequest: Trả về Lỗi Insufficient Funds
    RejectRequest --> [*]
    
    CheckAvailableBalance --> LockBalanceDB: Số dư Hợp lệ
    LockBalanceDB --> CreatePayoutTicket: Deduct Available Balance, Add Locked Balance
    CreatePayoutTicket --> ReleaseRedlock2: Set Ticket Status = 'PROCESSING'
    ReleaseRedlock2 --> EvaluateThreshold: Giải phóng Lock
    
    EvaluateThreshold --> AutoApprove: Amount < $5,000 USD
    EvaluateThreshold --> ManualApproval: Amount >= $5,000 USD
    
    ManualApproval --> WaitAdmin: Gửi Alert Telegram/Email cho Admin
    WaitAdmin --> RejectAdmin: Admin Bấm Reject
    RejectAdmin --> RefundBalance: Restore Available Balance from Locked Balance
    RefundBalance --> MarkRejected: Set Ticket Status = 'REJECTED'
    MarkRejected --> [*]
    
    WaitAdmin --> ApproveAdmin: Admin Bấm Approve
    ApproveAdmin --> AutoApprove
    
    AutoApprove --> BatchExecution: Set Ticket Status = 'APPROVED'
    BatchExecution --> SignAndBroadcast: Gom lệnh (Tx Batching) & Ký qua KMS
    SignAndBroadcast --> WaitConfirmation: Broadcast lên Blockchain / Bank API
    WaitConfirmation --> SettleSuccess: Confirmation Success
    SettleSuccess --> ReleaseLockedBalance: Locked Balance -= Amount + Fee
    ReleaseLockedBalance --> MarkSuccess: Set Ticket Status = 'SUCCESS' & Write Ledger
    MarkSuccess --> [*]
```

---

## 🌊 4. Sơ Đồ Luồng Dữ Liệu (Data Flow Diagram - DFD Level 1)

```mermaid
graph LR
    Merchant[Merchant Admin] -->|"1. Request Withdrawal"| Gateway[Merchant Gateway]
    Gateway -->|"2. Lock Account Lock"| RedisLock[(Redis Redlock Engine)]
    Gateway -->|"3. Lock Balance & Create Ticket"| BalancesDB[(Postgres Balances & Payouts)]
    
    Gateway -->|"4. Risk Threshold Check"| RiskEngine[Risk Assessment Engine]
    RiskEngine -->|"5a. High Value Alert"| AdminDashboard[Super Admin Portal]
    AdminDashboard -->|"5b. Manual Approve/Reject"| RiskEngine
    
    RiskEngine -->|"6. Publish Approved Payout Event"| NATSQueue[NATS JetStream Queue]
    NATSQueue -->|"7. Consume Payout Event"| PayoutBroadcaster[Payout Broadcaster Worker]
    PayoutBroadcaster -->|"8. Fetch Private Key & Sign"| KMS[GCP KMS / Vault Security]
    PayoutBroadcaster -->|"9. Broadcast Tx"| ExtNode[Blockchain Node / Bank Gateway]
    ExtNode -->|"10. Confirm Success"| PayoutBroadcaster
    PayoutBroadcaster -->|"11. Final Settle & Release Locked Balance"| BalancesDB
```

---

## 🧩 5. Sơ Đồ Thành Phần (Component Diagram)

```mermaid
flowchart TD
    subgraph "API & Risk Evaluation Layer"
        WithdrawCtrl["Withdrawal Controller"]
        RiskEval["Risk Threshold Evaluator"]
        AdminAppr["Admin Approval Manager"]
    end

    subgraph "Concurrency & Balance Manager Layer"
        RedisRedlock[("Redis Distributed Redlock")]
        BalLockMgr["Balance Lock Manager"]
    end

    subgraph "Execution Engine Layer"
        BatchEngine["Tx Batching Engine"]
        KMSSigner["KMS Signature Signer"]
        Broadcaster["Blockchain/Bank Broadcaster"]
    end

    subgraph "Data Layer"
        Postgres[("PostgreSQL (Payout Tickets & Ledgers)")]
        Redis[("Redis (Redlock & Nonce Tracker)")]
        NATSQueue["NATS JetStream (payout.events)"]
    end

    WithdrawCtrl --> RedisRedlock
    RedisRedlock --> BalLockMgr
    BalLockMgr --> Postgres
    WithdrawCtrl --> RiskEval
    RiskEval --> NATSQueue
    NATSQueue --> BatchEngine
    BatchEngine --> KMSSigner
    KMSSigner --> Broadcaster
    Broadcaster --> Postgres
```

---

## 📐 6. Sơ Đồ Lớp / Struct Go (Class / Struct Diagram)

```mermaid
classDiagram
    class PayoutTicket {
        +int64 ID
        +int64 MerchantID
        +string TicketCode
        +string Currency
        +decimal Amount
        +decimal NetworkFee
        +string DestinationAddress
        +string Status
        +bool RequiresManualApproval
        +time.Time CreatedAt
    }

    class RedlockManager {
        -redisClient RedisAdapter
        +AcquireLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
        +ReleaseLock(ctx context.Context, key string) error
    }

    class PayoutExecutionService {
        -kmsClient KMSSigner
        -payoutRepo PayoutRepository
        -natsClient NatsAdapter
        +ExecuteBatchPayout(ctx context.Context, tickets []*PayoutTicket) (string, error)
        +RefundRejectedPayout(ctx context.Context, ticketID int64) error
    }

    PayoutExecutionService ..> PayoutTicket : executes
    PayoutExecutionService ..> RedlockManager : uses
```

---

## 🗄️ 7. Sơ Đồ Thực Thể Liên Kết (ERD - Entity Relationship Diagram)

```mermaid
erDiagram
    MERCHANTS ||--o{ PAYOUT_TICKETS : "submits"
    PAYOUT_TICKETS ||--o| BATCH_PAYOUT_LOGS : "grouped_in"
    MERCHANTS ||--o{ BALANCES : "owns"

    PAYOUT_TICKETS {
        bigint id PK
        bigint merchant_id FK
        string ticket_code UK
        string currency
        decimal amount
        decimal network_fee
        string destination_address
        string status
        boolean requires_manual_approval
        bigint approved_by_user_id
        timestamp created_at
    }

    BATCH_PAYOUT_LOGS {
        bigint id PK
        string batch_tx_hash UK
        string network
        integer total_tickets_count
        decimal total_gas_spent
        timestamp broadcast_at
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
