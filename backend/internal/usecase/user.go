package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
)

var ErrUserNotFound = errors.New("user not found")
var ErrTagNotFound = errors.New("tag not found")
var ErrAlreadySubscribed = errors.New("already subscribed")
var ErrNotSubscribed = errors.New("not subscribed")
var ErrSourceNotFoundUser = errors.New("source not found")

// ─── Input / Output types ─────────────────────────────────────────────────────

type UpdateProfileInput struct {
	UserID            uuid.UUID
	DisplayName       *string
	PreferredLanguage *string
}

type SetInterestInput struct {
	TagName string
	Weight  float64
}

// ─── Usecase ──────────────────────────────────────────────────────────────────

type UserUsecase struct {
	users       repository.UserRepository
	interests   repository.InterestRepository
	tags        repository.TagRepository
	userSources repository.UserSourceRepository
	sources     repository.SourceRepository
}

func NewUserUsecase(
	users repository.UserRepository,
	interests repository.InterestRepository,
	tags repository.TagRepository,
	userSources repository.UserSourceRepository,
	sources repository.SourceRepository,
) *UserUsecase {
	return &UserUsecase{users: users, interests: interests, tags: tags, userSources: userSources, sources: sources}
}

func (uc *UserUsecase) GetProfile(ctx context.Context, userID uuid.UUID) (*domain.User, error) {
	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	return user, nil
}

func (uc *UserUsecase) UpdateProfile(ctx context.Context, in UpdateProfileInput) (*domain.User, error) {
	user, err := uc.users.FindByID(ctx, in.UserID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		return nil, ErrUserNotFound
	}

	if in.DisplayName != nil {
		user.DisplayName = *in.DisplayName
	}
	if in.PreferredLanguage != nil {
		user.PreferredLanguage = *in.PreferredLanguage
	}

	if err := uc.users.Update(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}
	return user, nil
}

func (uc *UserUsecase) GetInterests(ctx context.Context, userID uuid.UUID) ([]domain.InterestItem, error) {
	items, err := uc.interests.ListWithTags(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list interests: %w", err)
	}
	return items, nil
}

func (uc *UserUsecase) ListSources(ctx context.Context, userID uuid.UUID) ([]*domain.Source, error) {
	return uc.userSources.List(ctx, userID)
}

func (uc *UserUsecase) SubscribeSource(ctx context.Context, userID uuid.UUID, sourceID string) (*domain.Source, error) {
	source, err := uc.sources.FindByID(ctx, sourceID)
	if err != nil {
		return nil, fmt.Errorf("find source: %w", err)
	}
	if source == nil {
		return nil, ErrSourceNotFoundUser
	}

	already, err := uc.userSources.IsSubscribed(ctx, userID, sourceID)
	if err != nil {
		return nil, fmt.Errorf("check subscription: %w", err)
	}
	if already {
		return nil, ErrAlreadySubscribed
	}

	if err := uc.userSources.Subscribe(ctx, userID, sourceID); err != nil {
		return nil, fmt.Errorf("subscribe: %w", err)
	}
	return source, nil
}

func (uc *UserUsecase) UnsubscribeSource(ctx context.Context, userID uuid.UUID, sourceID string) error {
	subscribed, err := uc.userSources.IsSubscribed(ctx, userID, sourceID)
	if err != nil {
		return fmt.Errorf("check subscription: %w", err)
	}
	if !subscribed {
		return ErrNotSubscribed
	}
	return uc.userSources.Unsubscribe(ctx, userID, sourceID)
}

func (uc *UserUsecase) SetInterests(ctx context.Context, userID uuid.UUID, inputs []SetInterestInput) ([]domain.InterestItem, error) {
	if len(inputs) > 20 {
		return nil, fmt.Errorf("too many interests: max 20")
	}

	entries := make([]repository.InterestEntry, len(inputs))
	for i, inp := range inputs {
		tag, err := uc.tags.FindByName(ctx, inp.TagName)
		if err != nil {
			return nil, fmt.Errorf("find tag %q: %w", inp.TagName, err)
		}
		if tag == nil {
			return nil, fmt.Errorf("%w: %s", ErrTagNotFound, inp.TagName)
		}
		entries[i] = repository.InterestEntry{
			TagID:  tag.ID,
			Weight: inp.Weight,
		}
	}

	if err := uc.interests.ReplaceAll(ctx, userID, entries); err != nil {
		return nil, fmt.Errorf("replace interests: %w", err)
	}

	return uc.GetInterests(ctx, userID)
}
