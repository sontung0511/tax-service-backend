package repository

import (
	"context"

	"tax-client/backend/internal/domain"
)

type Repository interface {
	Snapshot(context.Context) (domain.Database, error)
	Update(context.Context, func(*domain.Database) error) error
}
