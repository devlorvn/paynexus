package repository

import (
	"context"
	"errors"

	"paynexus/internal/golibs/database"
	"paynexus/internal/merchant/domain"
)

type MerchantRepoImpl struct {
	db database.QueryExecer
}

func NewMerchantRepoImpl(db database.QueryExecer) domain.MerchantRepository {
	return &MerchantRepoImpl{
		db: db,
	}
}

func (r *MerchantRepoImpl) Create(ctx context.Context, m *domain.Merchant) error {
	query := `
		INSERT INTO public.merchants (code, name, email, status, resource_path, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;
	`
	return r.db.QueryRow(ctx, query, m.Code, m.Name, m.Email, m.Status, m.ResourcePath, m.CreatedAt, m.UpdatedAt).Scan(&m.ID)
}

func (r *MerchantRepoImpl) GetByID(ctx context.Context, id int64) (*domain.Merchant, error) {
	query := `
		SELECT id, code, name, email, status, resource_path, created_at, updated_at
		FROM public.merchants
		WHERE id = $1;
	`
	m := &domain.Merchant{}
	err := r.db.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.Code, &m.Name, &m.Email, &m.Status, &m.ResourcePath, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, database.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return m, nil
}

func (r *MerchantRepoImpl) GetByCode(ctx context.Context, code string) (*domain.Merchant, error) {
	query := `
		SELECT id, code, name, email, status, resource_path, created_at, updated_at
		FROM public.merchants
		WHERE code = $1;
	`
	m := &domain.Merchant{}
	err := r.db.QueryRow(ctx, query, code).Scan(
		&m.ID, &m.Code, &m.Name, &m.Email, &m.Status, &m.ResourcePath, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, database.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	return m, nil
}
