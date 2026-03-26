package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
)

type UserSourceRepository interface {
	List(ctx context.Context, userID uuid.UUID) ([]*domain.Source, error)
	Subscribe(ctx context.Context, userID uuid.UUID, sourceID string) error
	Unsubscribe(ctx context.Context, userID uuid.UUID, sourceID string) error
	IsSubscribed(ctx context.Context, userID uuid.UUID, sourceID string) (bool, error)
}
