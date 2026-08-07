package domain

import (
	"context"
	"errors"
	"time"
)

var ErrNotFound = errors.New("merchant not found")

type Merchant struct {
	ID           int64     `db:"id"`
	Code         string    `db:"code"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	Status       string    `db:"status"`
	ResourcePath string    `db:"resource_path"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

type MerchantRepository interface {
	Create(ctx context.Context, merchant *Merchant) error
	GetByID(ctx context.Context, id int64) (*Merchant, error)
	GetByCode(ctx context.Context, code string) (*Merchant, error)
}
