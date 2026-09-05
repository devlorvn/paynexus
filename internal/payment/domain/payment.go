package domain

import (
	"context"
	"errors"
	"time"
)

var (
	ErrInvoiceNotFound  = errors.New("invoice not found")
	ErrDuplicateInvoice = errors.New("duplicate order reference")
)

// Invoice Entity
type Invoice struct {
	ID             int64     `db:"id"`
	MerchantID     int64     `db:"merchant_id"`
	InvoiceCode    string    `db:"invoice_code"`
	OrderReference string    `db:"order_reference"`
	Currency       string    `db:"currency"`
	Amount         float64   `db:"amount"`
	DepositAddress string    `db:"deposit_address"`
	MemoCode       string    `db:"memo_code"`
	Status         string    `db:"status"`
	ResourcePath   string    `db:"resource_path"`
	ExpiredAt      time.Time `db:"expired_at"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}

// Balance Entity
type Balance struct {
	ID               int64     `db:"id"`
	MerchantID       int64     `db:"merchant_id"`
	Currency         string    `db:"currency"`
	AvailableBalance float64   `db:"available_balance"`
	LockedBalance    float64   `db:"locked_balance"`
	ResourcePath     string    `db:"resource_path"`
	UpdatedAt        time.Time `db:"updated_at"`
}

// PaymentRepository Interface thuần Clean Architecture
type PaymentRepository interface {
	CreateInvoice(ctx context.Context, invoice *Invoice) error
	GetInvoiceByCode(ctx context.Context, invoiceCode string) (*Invoice, error)
	UpdateInvoiceStatus(ctx context.Context, invoiceCode string, status string) error

	GetBalance(ctx context.Context, merchantID int64, currency string) (*Balance, error)
	// ExecuteSettlementTx thực thi ACID Transaction: Cộng số dư + Ghi dòng Nợ/Có vào Ledger
	ExecuteSettlementTx(ctx context.Context, invoice *Invoice, txID string) error
}

type EventPublisher interface {
	PublishEvent(ctx context.Context, subject string, payload []byte) error
}

type IdempotencyStorage interface {
	GetResponse(ctx context.Context, key string) ([]byte, bool, error)
	SaveResponse(ctx context.Context, key string, responseBytes []byte, ttl time.Duration) error
}
