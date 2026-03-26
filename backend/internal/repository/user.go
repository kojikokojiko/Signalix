package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	FindByEmail(ctx context.Context, email string) (*domain.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
	UpdateLastLogin(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, user *domain.User) error
}
