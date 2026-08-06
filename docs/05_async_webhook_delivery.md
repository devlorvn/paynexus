# 📄 Nghiệp vụ 5: Engine Gửi Webhook Bất Đồng Bộ Tin Cậy (Async Reliable Webhook Delivery Engine)

Tài liệu này mô tả chi tiết quy trình nghiệp vụ, sơ đồ Use Case, Activity, DFD, Component, Class/Struct và ERD cho phân hệ **Gửi Webhook thông báo bất đồng bộ tin cậy (At-Least-Once Delivery Webhook Engine), Tự động Retry lũy thừa (Exponential Backoff), Hàng chờ tin nhắn hỏng (Dead Letter Queue - DLQ) và Ký chữ ký bảo mật HMAC-SHA256**.

---

## 📑 1. Quy Trình Nghiệp Vụ Chi Tiết (Business Workflow)

1. **Phát Sự Kiện Cần Gửi Webhook (Event Triggering)**:
   - Khi bất kỳ hóa đơn Nạp (`deposit.settled`) hoặc lệnh Rút (`payout.settled`) được xử lý xong, `payment-engine` phát Event `webhook.dispatch` vào **NATS JetStream Queue**.
   - Event chứa: `MerchantID`, `EventType`, `PayloadData`, `Timestamp`.
2. **Ký Chữ Ký Bảo Mật Webhook (HMAC-SHA256 Payload Signing)**:
   - Worker nhận Event từ NATS Queue.
   - Tải `WebhookURL` và `API_SECRET` của Merchant từ Redis Cache.
   - Sinh chữ ký HMAC-SHA256 bảo mật:
     ```
     Signature = HMAC-SHA256(Timestamp + "." + PayloadJSON, API_SECRET)
     ```
   - Thêm Header bảo mật vào Request HTTP Post:
     - `X-PayNexus-Signature: t=Timestamp,v1=Signature`
     - `Content-Type: application/json`
3. **Thực thi Gửi Webhook & Kiểm tra Phản hồi (Delivery Execution)**:
   - Gửi HTTP POST Request tới `WebhookURL` của Merchant (Timeout limit = 10 giây).
   - **Xác nhận Thành công**: Nếu HTTP Status Code nhận về là `200 OK` hoặc `201 Created` ➔ Đánh dấu Gửi Webhook thành công (`SUCCESS`). Ghi log thời gian phản hồi.
4. **Chiến lược Retry Tự Động Lũy Thừa (Exponential Backoff Retries)**:
   - Nếu HTTP Status Code `!= 2xx` (VD: `500 Server Error`, `503 Service Unavailable`, `Timeout`):
     - Hệ thống đưa tin nhắn vào hàng chờ Retry với lịch trình tăng dần thời gian giãn cách:
       - **Lần 1**: Ngay lập tức.
       - **Lần 2**: Sau 5 giây.
       - **Lần 3**: Sau 30 giây.
       - **Lần 4**: Sau 5 phút.
       - **Lần 5**: Sau 1 giờ.
5. **Chuyển sang Hàng Chờ Tin Nhắn Hỏng (Dead Letter Queue - DLQ Management)**:
   - Sau **5 lần Retry thất bại** (Merchant Server bị sập kéo dài hoặc URL bị lỗi):
     - Chuyển Webhook Message vào trạng thái `DEAD_LETTER` (DLQ).
     - Gửi cảnh báo trên Merchant Dashboard: *"Cảnh báo: Server của bạn không phản hồi Webhook 5 lần liên tiếp"*.
   - **Manual Re-triggering**: Merchant Admin sau khi sửa xong server có thể vào Dashboard nhấn nút **"Re-send Webhook"** để kích hoạt gửi lại thủ công.

---

## 🎯 2. Sơ Đồ Use Case (Use Case Diagram)

```mermaid
graph TD
    subgraph "PayNexus Webhook System"
        SYS((Payment Engine)) --> UC1[Emit Webhook Dispatch Event]
        WWORK((Webhook Worker)) --> UC2[Sign Payload via HMAC SHA256]
        WWORK --> UC3[Send HTTP POST to Merchant URL]
        WWORK --> UC4[Execute Exponential Backoff Retries]
        WWORK --> UC5[Move Failed Webhook to DLQ]
    end

    subgraph "Merchant Application"
        MA((Merchant Server)) --> UC6[Receive & Verify Webhook Signature]
        MADMIN((Merchant Admin)) --> UC7[View DLQ Logs & Manually Re-trigger Webhook]
    end
```

---

## 🔄 3. Sơ Đồ Hoạt Động (Activity Diagram)

```mermaid
stateDiagram-v2
    [*] --> EventConsumed: Worker nhận Event `webhook.dispatch` từ NATS
    EventConsumed --> FetchSecret: Tải Merchant Webhook URL & API Secret từ Redis
    FetchSecret --> SignHMAC: Sinh chữ ký HMAC-SHA256 Header (X-PayNexus-Signature)
    
    SignHMAC --> SendHTTPPost: Gửi HTTP POST Request (Timeout = 10s)
    SendHTTPPost --> CheckHTTPStatus: Nhận HTTP Status Code từ Merchant Server
    
    CheckHTTPStatus --> DeliverySuccess: Status Code 2xx (200 OK / 201)
    DeliverySuccess --> MarkSuccess: Cập nhật Trạng thái Webhook Log = 'SUCCESS'
    MarkSuccess --> [*]
    
    CheckHTTPStatus --> CheckRetryCount: Status Code != 2xx / Timeout / Conn Refused
    CheckRetryCount --> ScheduleRetry: Retry Count < 5 Lần
    ScheduleRetry --> WaitBackoff: Calculate Exponential Backoff Delay (5s, 30s, 5m, 1h)
    WaitBackoff --> SendHTTPPost
    
    CheckRetryCount --> MoveToDLQ: Retry Count >= 5 Lần
    MoveToDLQ --> MarkDLQ: Cập nhật Trạng thái Webhook Log = 'DEAD_LETTER' (DLQ)
    MarkDLQ --> NotifyDashboard: Đẩy Alert thông báo lên Merchant Dashboard
    NotifyDashboard --> [*]
```

---

## 🌊 4. Sơ Đồ Luồng Dữ Liệu (Data Flow Diagram - DFD Level 1)

```mermaid
graph LR
    PaymentEngine[Payment Engine] -->|"1. Publish Webhook Event"| NATSQueue[NATS JetStream Queue]
    NATSQueue -->|"2. Consume Event"| WebhookWorker[Webhook Dispatch Worker]
    WebhookWorker -->|"3. Get Webhook URL & Secret"| Redis[(Redis Cache)]
    
    WebhookWorker -->|"4. Sign & Send HTTP POST"| MerchantServer[Merchant External Server]
    MerchantServer -->|"5. Return 200 OK / Error"| WebhookWorker
    
    WebhookWorker -->|"6a. If Failed: Re-queue with Delay"| NATSQueue
    WebhookWorker -->|"6b. If Max Retries: Write to DLQ Table"| Postgres[(PostgreSQL Webhook Logs)]
    
    MerchantAdmin[Merchant Admin] -->|"7. Manual Re-send Request"| Dashboard[Merchant Dashboard]
    Dashboard -->|"8. Push DLQ Message Back to Queue"| NATSQueue
```

---

## 🧩 5. Sơ Đồ Thành Phần (Component Diagram)

```mermaid
flowchart TD
    subgraph "Message Queue Layer"
        NATSQueue["NATS JetStream (webhook.events)"]
        DLQQueue["NATS Dead Letter Queue (webhook.dlq)"]
    end

    subgraph "Webhook Delivery Worker Layer"
        HMACSigner["HMAC Payload Signer"]
        HTTPDisp["HTTP Client Dispatcher"]
        BackoffSched["Exponential Backoff Scheduler"]
        DLQMgr["DLQ Manager"]
    end

    subgraph "Storage & Logging Layer"
        Postgres[("PostgreSQL (Webhook Delivery Logs)")]
        Redis[("Redis (Merchant Secret & Retry Counter)")]
    end

    NATSQueue --> HMACSigner
    HMACSigner --> Redis
    HMACSigner --> HTTPDisp
    HTTPDisp --> BackoffSched
    BackoffSched --> NATSQueue
    BackoffSched --> DLQMgr
    DLQMgr --> DLQQueue
    DLQMgr --> Postgres
```

---

## 📐 6. Sơ Đồ Lớp / Struct Go (Class / Struct Diagram)

```mermaid
classDiagram
    class WebhookLog {
        +int64 ID
        +int64 MerchantID
        +string EventType
        +string TargetURL
        +string PayloadJSON
        +string Signature
        +int AttemptCount
        +int LastHTTPStatus
        +string Status
        +time.Time NextRetryAt
        +time.Time CreatedAt
    }

    class WebhookSigner {
        +GenerateHMAC(payload []byte, timestamp int64, secret string) string
        +VerifyHMAC(payload []byte, timestamp int64, secret string, signature string) bool
    }

    class WebhookDispatcherService {
        -httpClient HTTPClientAdapter
        -natsClient NatsAdapter
        -webhookRepo WebhookRepository
        +DispatchWebhook(ctx context.Context, log *WebhookLog) error
        +RetryWebhook(ctx context.Context, logID int64) error
        +MoveToDLQ(ctx context.Context, logID int64) error
    }

    WebhookDispatcherService ..> WebhookLog : dispatches
    WebhookDispatcherService ..> WebhookSigner : uses
```

---

## 🗄️ 7. Sơ Đồ Thực Thể Liên Kết (ERD - Entity Relationship Diagram)

```mermaid
erDiagram
    MERCHANTS ||--o{ WEBHOOK_LOGS : "receives"
    WEBHOOK_LOGS ||--o{ WEBHOOK_DELIVERY_ATTEMPTS : "has_history"

    WEBHOOK_LOGS {
        bigint id PK
        bigint merchant_id FK
        string event_type
        string target_url
        jsonb payload_json
        string signature
        integer attempt_count
        integer last_http_status
        string status
        timestamp next_retry_at
        timestamp created_at
    }

    WEBHOOK_DELIVERY_ATTEMPTS {
        bigint id PK
        bigint webhook_log_id FK
        integer attempt_number
        integer response_http_status
        text response_body
        integer execution_time_ms
        timestamp attempted_at
    }
```
