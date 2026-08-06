# 📚 PayNexus Engine - Tài Liệu Đặc Tả Chi Tiết Các Quy Trình Nghiệp Vụ

Chào mừng bạn đến với bộ tài liệu đặc tả thiết kế chi tiết toàn bộ các quy trình nghiệp vụ và lộ trình triển khai cho hệ thống **PayNexus Engine (Multi-Chain Crypto & Fiat Payment Switch Engine)**.

Mỗi file tài liệu dưới đây đại diện cho một phân hệ nghiệp vụ độc lập, bao gồm đầy đủ **Mô tả quy trình, Sơ đồ Use Case, Activity, DFD Level 1, Component Diagram, Class/Struct Diagram và ERD Database**.

---

## 📂 Danh Mục Các File Đặc Tả Nghiệp Vụ

1. 🔐 **[01. Multi-Tenant Onboarding & RBAC API Security](docs/01_multi_tenant_and_auth.md)**
   - Quy trình Onboarding Merchant, Sinh cặp khóa API Key/Secret, Whitelist IP.
   - Phân quyền nhân sự chi nhánh (Location-Restricted RBAC) và cách ly dữ liệu bằng PostgreSQL RLS.
   - Xử lý xác thực chữ ký HMAC-SHA256 & chống Replay Attack ở tầng Gateway.

2. 💳 **[02. Dynamic Deposit & Invoice Gateway Engine](docs/02_deposit_and_invoice.md)**
   - Khởi tạo Hóa đơn Nạp tiền kèm Idempotency-Key chống trùng lặp.
   - Cấp phát VietQR Memo Code & Crypto Address Pool linh hoạt.
   - Lắng nghe sự kiện tiền vào (Blockchain/Bank) bất đồng bộ & Khớp hóa đơn tự động.

3. 💸 **[03. Automatic Withdrawal & Risk Threshold Payout Engine](docs/03_withdrawal_and_payout.md)**
   - Khóa số dư khả dụng bằng Redis Distributed Lock (`redlock`) chống Race-Condition.
   - Phân luồng kiểm soát rủi ro (Risk Threshold Approval): Duyệt tự động vs Duyệt thủ công.
   - Gom lệnh rút (Crypto Tx Batching) tối ưu 60% Phí Gas.

4. 📊 **[04. Double-Entry Ledger & Financial Reconciliation Engine](docs/04_double_entry_ledger_and_reconciliation.md)**
   - Nguyên tắc Ghi sổ đúp (Double-Entry Ledger) Nợ/Có bất biến (Immutable Audit Log).
   - CDC Pipeline (Debezium/Postgres WAL) stream sổ cái sang ClickHouse / BigQuery.
   - Tự động chạy Job đối soát 5 phút/lần & Công tắc ngắt khẩn cấp (Emergency Circuit Breaker).

5. 🔔 **[05. Async Reliable Webhook Delivery Engine](docs/05_async_webhook_delivery.md)**
   - Cam kết At-Least-Once Webhook Delivery.
   - Lịch trình Retry lũy thừa (Exponential Backoff: 5s, 30s, 5m, 1h).
   - Quản lý Hàng chờ tin nhắn hỏng (Dead Letter Queue - DLQ) & Ký chữ ký Webhook HMAC-SHA256.

6. 📅 **[06. Implementation Roadmap & Milestones (Tiến Trình Thực Hiện Dự Án)](docs/06_implementation_roadmap_and_milestones.md)**
   - Sơ đồ Gantt Roadmap 6 Tháng / 12 Sprints chi tiết.
   - Phân công công việc theo 6 Giai đoạn (Phase-by-Phase Execution Plan).
   - Chỉ số KPI bàn giao và Bảng theo dõi tiến độ dự án (Milestone Tracking Matrix).

---

## 📌 Tài Liệu Tổng Quan

- **[PayNexus Engine Core Product Specification](specification.md)**: Đặc tả sản phẩm tổng thể, bối cảnh thị trường, các tác nhân và yêu cầu phi nghiệp vụ (SLA / TPS).
