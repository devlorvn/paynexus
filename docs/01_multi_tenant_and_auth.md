# 📄 Nghiệp vụ 1: Onboarding Merchant, Cấp Phát API Key & Phân Quyền Đa Thuê Bao (Multi-Tenant & RBAC)

Tài liệu này tả chi tiết toàn bộ quy trình nghiệp vụ, sơ đồ thiết kế Use Case, Activity, DFD, Component, Class/Struct và ERD cho phân hệ **Quản lý Merchant, Cấp phát khóa bảo mật API Key/Secret và Phân quyền truy cập theo cấu trúc tổ chức/chi nhánh (Multi-Tenant Location-Restricted RBAC)**.

---

## 📑 1. Quy Trình Nghiệp Vụ Chi Tiết (Business Workflow)

1. **Merchant Onboarding (Đăng ký Doanh nghiệp)**:
   - Super Admin tiếp nhận thông tin đăng ký của doanh nghiệp (Tên công ty, Email đại diện, Giấy phép kinh doanh, KYC).
   - Hệ thống khởi tạo một **Tenant ID** (Merchant ID) độc nhất và gán cấu hình mặc định (Loại tiền tệ mặc định, Phí giao dịch mặc định).
2. **Cấp phát & Quản lý Khóa API Key/Secret**:
   - Merchant Admin đăng nhập vào Dashboard và yêu cầu sinh cặp khóa API:
     - `API_KEY_PUBLIC`: Dùng cho các tích hợp phía Client Web/Mobile (Public ID).
     - `API_SECRET_PRIVATE`: Dùng cho Server-to-Server HMAC SHA256 signing (Chỉ hiển thị 1 lần duy nhất khi tạo).
   - Merchant cấu hình danh sách **IP Whitelist** (Chỉ cho phép các IP server này gọi API Rút tiền).
3. **Phân quyền Chi nhánh & Nhân sự (Location-Restricted RBAC)**:
   - Merchant tạo các tài khoản nhân viên (Sub-Accounts / Branch Staff) và phân quyền:
     - **Role Merchant Admin**: Toàn quyền quản lý, rút tiền, đổi Secret Key, xem báo cáo tổng.
     - **Role Branch Manager / POS Staff**: Chỉ có quyền tạo Hóa đơn nạp tiền, xem lịch sử giao dịch tại Chi nhánh/Cửa hàng được phân công (`resource_path = /org1/branchA/`). KHÔNG có quyền rút tiền hay đổi Secret Key.
4. **Cơ chế Xác thực Request (API Authentication Interceptor)**:
   - Mọi request gửi lên `merchant-gateway` phải chứa Header:
     - `X-Nexus-API-Key`: Key công khai.
     - `X-Nexus-Signature`: Chữ ký HMAC-SHA256(`body` + `timestamp`, `API_SECRET`).
     - `X-Nexus-Timestamp`: Thời gian tạo request (Chống Replay Attack, chênh lệch tối đa 300s).

---

## 🎯 2. Sơ Đồ Use Case (Use Case Diagram)

```mermaid
graph TD
    subgraph "Super Admin Tasks"
        SA((Super Admin)) --> UC1[Onboard New Merchant]
        SA --> UC2[Set Platform Fee & Risk Threshold]
        SA --> UC3[Suspend / Lock Merchant]
    end

    subgraph "Merchant Admin Tasks"
        MA((Merchant Admin)) --> UC4[Generate API Keys & Secret]
        MA --> UC5[Configure IP Whitelist & Webhook URL]
        MA --> UC6[Create Sub-Accounts & Assign Roles]
        MA --> UC7[View Master Balance & Financial Reports]
    end

    subgraph "Branch Staff Tasks"
        BS((Branch Staff)) --> UC8[Create POS Invoice / QR Code]
        BS --> UC9[View Branch Transaction History]
    end

    subgraph "System Interceptors"
        SYS((Gateway Interceptor)) --> UC10[Validate HMAC Signature & Timestamp]
        SYS --> UC11[Enforce PostgreSQL RLS Policy]
    end
```

---

## 🔄 3. Sơ Đồ Hoạt Động (Activity Diagram)

```mermaid
stateDiagram-v2
    [*] --> RequestReceived: Request tới Gateway kèm API Key & Signature
    RequestReceived --> CheckTimestamp: Kiểm tra X-Nexus-Timestamp
    
    CheckTimestamp --> RejectReplay: Chênh lệch > 300 giây
    RejectReplay --> [*]
    
    CheckTimestamp --> CheckIPWhitelist: Timestamp hợp lệ
    CheckIPWhitelist --> RejectIP: IP không nằm trong Whitelist
    RejectIP --> [*]
    
    CheckIPWhitelist --> FetchAPISecret: IP Hợp lệ
    FetchAPISecret --> VerifyHMAC: Tải API Secret từ Redis/Postgres
    VerifyHMAC --> RejectAuth: Signature không khớp
    RejectAuth --> [*]
    
    VerifyHMAC --> ExtractTenantContext: Signature Khớp
    ExtractTenantContext --> InjectRLS: Lấy TenantID & ResourcePath
    InjectRLS --> ExecuteBusinessLogic: Inject vào Go context.Context
    ExecuteBusinessLogic --> [*]: Thực thi RPC Service
```

---

## 🌊 4. Sơ Đồ Luồng Dữ Liệu (Data Flow Diagram - DFD Level 1)

```mermaid
graph LR
    MerchantApp[Merchant App / Client] -->|"1. Submit Request + Signature"| Gateway[Merchant Gateway Interceptor]
    Gateway -->|"2. Query Secret & IP Whitelist"| Redis[(Redis Cache)]
    Redis -->|"3. Return Secret"| Gateway
    Gateway -->|"4. Validate Signature Success"| AuthContext[Context Generator]
    AuthContext -->|"5. Inject Tenant ID & RLS Context"| MerchantService[Merchant Core Service]
    MerchantService -->|"6. Query Data with RLS"| PostgresDB[(PostgreSQL Database)]
    PostgresDB -->|"7. Return Isolated Tenant Data"| MerchantService
    MerchantService -->|"8. gRPC Response"| MerchantApp
```

---

## 🧩 5. Sơ Đồ Thành Phần (Component Diagram)

```mermaid
flowchart TD
    subgraph "Merchant Gateway Layer"
        AuthInt["Auth Interceptor"]
        HMACVal["HMAC Validator"]
        RateLimit["Rate Limiter & IP Filter"]
    end

    subgraph "Merchant Core Service"
        MerchMgr["Merchant Manager"]
        APIKeySvc["API Key Service"]
        RBACCheck["RBAC Permission Checker"]
    end

    subgraph "Data Layer"
        Postgres[("PostgreSQL (RLS Enabled)")]
        RedisCache[("Redis (Session & Secret Cache)")]
    end

    AuthInt --> HMACVal
    HMACVal --> RateLimit
    RateLimit --> MerchMgr
    MerchMgr --> APIKeySvc
    APIKeySvc --> RedisCache
    MerchMgr --> Postgres
```

---

## 📐 6. Sơ Đồ Lớp / Struct Go (Class / Struct Diagram)

```mermaid
classDiagram
    class Merchant {
        +int64 ID
        +string Code
        +string Name
        +string Email
        +string Status
        +string ResourcePath
        +time.Time CreatedAt
    }

    class APIKey {
        +int64 ID
        +int64 MerchantID
        +string PublicKey
        +string SecretKeyHash
        +string[] IPWhitelist
        +string Status
        +time.Time ExpiresAt
    }

    class UserAccount {
        +int64 ID
        +int64 MerchantID
        +string Username
        +string PasswordHash
        +string Role
        +string LocationPath
    }

    class AuthInterceptor {
        -redisClient RedisAdapter
        -merchantRepo MerchantRepository
        +UnaryServerInterceptor() grpc.UnaryServerInterceptor
        +ValidateHMAC(ctx context.Context) error
    }

    Merchant "1" -- "N" APIKey : owns
    Merchant "1" -- "N" UserAccount : employs
    AuthInterceptor ..> APIKey : validates
```

---

## 🗄️ 7. Sơ Đồ Thực Thể Liên Kết (ERD - Entity Relationship Diagram)

```mermaid
erDiagram
    MERCHANTS ||--o{ API_KEYS : "has"
    MERCHANTS ||--o{ USER_ACCOUNTS : "employs"
    MERCHANTS ||--o{ MERCHANT_CONFIGS : "configured_by"
    USER_ACCOUNTS }|--|| ROLES : "assigned"

    MERCHANTS {
        bigint id PK
        string code UK
        string name
        string email
        string status
        string resource_path
        timestamp created_at
    }

    API_KEYS {
        bigint id PK
        bigint merchant_id FK
        string public_key UK
        string secret_key_hash
        string ip_whitelist
        string status
        timestamp expires_at
    }

    USER_ACCOUNTS {
        bigint id PK
        bigint merchant_id FK
        string username UK
        string password_hash
        string role_code FK
        string location_path
    }

    ROLES {
        string code PK
        string name
        jsonb permissions
    }

    MERCHANT_CONFIGS {
        bigint merchant_id PK
        string default_currency
        decimal platform_fee_rate
        string webhook_url
    }
```
