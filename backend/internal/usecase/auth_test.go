package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/usecase"
	"golang.org/x/crypto/bcrypt"
)

// --- mock ---

type mockUserRepo struct {
	users  map[string]*domain.User
	byID   map[uuid.UUID]*domain.User
	create func(*domain.User) error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{
		users: make(map[string]*domain.User),
		byID:  make(map[uuid.UUID]*domain.User),
	}
}

func (m *mockUserRepo) Create(ctx context.Context, u *domain.User) error {
	if m.create != nil {
		return m.create(u)
	}
	if _, exists := m.users[u.Email]; exists {
		return usecase.ErrEmailAlreadyExists
	}
	m.users[u.Email] = u
	m.byID[u.ID] = u
	return nil
}

func (m *mockUserRepo) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	u, ok := m.users[email]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, nil
	}
	return u, nil
}

func (m *mockUserRepo) UpdateLastLogin(ctx context.Context, id uuid.UUID) error {
	return nil
}
func (m *mockUserRepo) Update(_ context.Context, _ *domain.User) error {
	return nil
}

// --- mock Redis for lock ---

type mockLockStore struct {
	failCount map[string]int
	locked    map[string]bool
}

func newMockLockStore() *mockLockStore {
	return &mockLockStore{
		failCount: make(map[string]int),
		locked:    make(map[string]bool),
	}
}

func (m *mockLockStore) GetFailCount(ctx context.Context, email string) (int, error) {
	return m.failCount[email], nil
}

func (m *mockLockStore) IncrFailCount(ctx context.Context, email string, ttl time.Duration) error {
	m.failCount[email]++
	return nil
}

func (m *mockLockStore) ResetFailCount(ctx context.Context, email string) error {
	delete(m.failCount, email)
	return nil
}

func (m *mockLockStore) IsLocked(ctx context.Context, email string) (bool, error) {
	return m.locked[email], nil
}

func (m *mockLockStore) Lock(ctx context.Context, email string, ttl time.Duration) error {
	m.locked[email] = true
	return nil
}

func (m *mockLockStore) BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error {
	return nil
}

func (m *mockLockStore) IsTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	return false, nil
}

// --- helpers ---

func hashPassword(t *testing.T, pw string) string {
	t.Helper()
	h, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	return string(h)
}

func newAuthUsecase(repo *mockUserRepo, lock *mockLockStore) *usecase.AuthUsecase {
	return usecase.NewAuthUsecase(repo, lock, "test-secret", 1*time.Hour, 7*24*time.Hour)
}

// ─── Register ────────────────────────────────────────────────────────────────

func TestAuthUsecase_Register_Success(t *testing.T) {
	repo := newMockUserRepo()
	uc := newAuthUsecase(repo, newMockLockStore())

	result, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:       "user@example.com",
		Password:    "Secure1234",
		DisplayName: "Test User",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected access token")
	}
	if result.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if result.User.Email != "user@example.com" {
		t.Errorf("expected email user@example.com, got %s", result.User.Email)
	}
	if len(repo.users) != 1 {
		t.Error("expected user to be persisted")
	}
}

func TestAuthUsecase_Register_DuplicateEmail(t *testing.T) {
	repo := newMockUserRepo()
	uc := newAuthUsecase(repo, newMockLockStore())

	input := usecase.RegisterInput{
		Email:       "dup@example.com",
		Password:    "Secure1234",
		DisplayName: "User",
	}
	_, _ = uc.Register(context.Background(), input)
	_, err := uc.Register(context.Background(), input)

	if !errors.Is(err, usecase.ErrEmailAlreadyExists) {
		t.Errorf("expected ErrEmailAlreadyExists, got %v", err)
	}
}

func TestAuthUsecase_Register_PasswordIsHashed(t *testing.T) {
	repo := newMockUserRepo()
	uc := newAuthUsecase(repo, newMockLockStore())

	_, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:       "u@example.com",
		Password:    "Secure1234",
		DisplayName: "User",
	})
	if err != nil {
		t.Fatal(err)
	}

	stored := repo.users["u@example.com"]
	if stored.PasswordHash == "Secure1234" {
		t.Error("password must not be stored in plain text")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored.PasswordHash), []byte("Secure1234")); err != nil {
		t.Error("stored hash does not match original password")
	}
}

// ─── Login ───────────────────────────────────────────────────────────────────

func TestAuthUsecase_Login_Success(t *testing.T) {
	repo := newMockUserRepo()
	repo.users["u@example.com"] = &domain.User{
		ID:           uuid.New(),
		Email:        "u@example.com",
		PasswordHash: hashPassword(t, "Secure1234"),
		DisplayName:  "User",
		IsActive:     true,
	}

	uc := newAuthUsecase(repo, newMockLockStore())
	result, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "u@example.com",
		Password: "Secure1234",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestAuthUsecase_Login_WrongPassword(t *testing.T) {
	repo := newMockUserRepo()
	repo.users["u@example.com"] = &domain.User{
		ID:           uuid.New(),
		Email:        "u@example.com",
		PasswordHash: hashPassword(t, "Secure1234"),
		DisplayName:  "User",
		IsActive:     true,
	}

	uc := newAuthUsecase(repo, newMockLockStore())
	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "u@example.com",
		Password: "WrongPassword1",
	})

	if !errors.Is(err, usecase.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthUsecase_Login_UnknownEmail(t *testing.T) {
	repo := newMockUserRepo()
	uc := newAuthUsecase(repo, newMockLockStore())

	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "nobody@example.com",
		Password: "Secure1234",
	})

	if !errors.Is(err, usecase.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestAuthUsecase_Login_LockedAccount(t *testing.T) {
	repo := newMockUserRepo()
	repo.users["u@example.com"] = &domain.User{
		ID:           uuid.New(),
		Email:        "u@example.com",
		PasswordHash: hashPassword(t, "Secure1234"),
		IsActive:     true,
	}
	lock := newMockLockStore()
	lock.locked["u@example.com"] = true

	uc := newAuthUsecase(repo, lock)
	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "u@example.com",
		Password: "Secure1234",
	})

	if !errors.Is(err, usecase.ErrAccountLocked) {
		t.Errorf("expected ErrAccountLocked, got %v", err)
	}
}

func TestAuthUsecase_Login_DisabledAccount(t *testing.T) {
	repo := newMockUserRepo()
	repo.users["u@example.com"] = &domain.User{
		ID:           uuid.New(),
		Email:        "u@example.com",
		PasswordHash: hashPassword(t, "Secure1234"),
		IsActive:     false,
	}

	uc := newAuthUsecase(repo, newMockLockStore())
	_, err := uc.Login(context.Background(), usecase.LoginInput{
		Email:    "u@example.com",
		Password: "Secure1234",
	})

	if !errors.Is(err, usecase.ErrAccountDisabled) {
		t.Errorf("expected ErrAccountDisabled, got %v", err)
	}
}

func TestAuthUsecase_Login_LocksAfterFiveFailures(t *testing.T) {
	repo := newMockUserRepo()
	repo.users["u@example.com"] = &domain.User{
		ID:           uuid.New(),
		Email:        "u@example.com",
		PasswordHash: hashPassword(t, "Secure1234"),
		IsActive:     true,
	}
	lock := newMockLockStore()
	uc := newAuthUsecase(repo, lock)

	for i := 0; i < 5; i++ {
		_, _ = uc.Login(context.Background(), usecase.LoginInput{
			Email:    "u@example.com",
			Password: "WrongPassword1",
		})
	}

	if !lock.locked["u@example.com"] {
		t.Error("account should be locked after 5 consecutive failures")
	}
}

// ─── Refresh ─────────────────────────────────────────────────────────────────

func TestAuthUsecase_Refresh_Success(t *testing.T) {
	repo := newMockUserRepo()
	uc := newAuthUsecase(repo, newMockLockStore())

	reg, err := uc.Register(context.Background(), usecase.RegisterInput{
		Email:       "u@example.com",
		Password:    "Secure1234",
		DisplayName: "User",
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := uc.Refresh(context.Background(), reg.RefreshToken)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.AccessToken == "" {
		t.Error("expected new access token")
	}
}

func TestAuthUsecase_Refresh_InvalidToken(t *testing.T) {
	uc := newAuthUsecase(newMockUserRepo(), newMockLockStore())
	_, err := uc.Refresh(context.Background(), "not-a-valid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}
