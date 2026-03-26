package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/kojikokojiko/signalix/internal/domain"
	"github.com/kojikokojiko/signalix/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

const maxLoginAttempts = 5
const lockDuration = 15 * time.Minute
const failCountTTL = 30 * time.Minute

// LockStore abstracts Redis-backed brute-force protection and token blacklisting.
type LockStore interface {
	GetFailCount(ctx context.Context, email string) (int, error)
	IncrFailCount(ctx context.Context, email string, ttl time.Duration) error
	ResetFailCount(ctx context.Context, email string) error
	IsLocked(ctx context.Context, email string) (bool, error)
	Lock(ctx context.Context, email string, ttl time.Duration) error
	BlacklistToken(ctx context.Context, jti string, ttl time.Duration) error
	IsTokenBlacklisted(ctx context.Context, jti string) (bool, error)
}

type AuthInput struct {
	Email    string
	Password string
}

type RegisterInput struct {
	Email       string
	DisplayName string
	Password    string
}

type LoginInput struct {
	Email    string
	Password string
}

type AuthResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // seconds
	User         *domain.User
}

type RefreshResult struct {
	AccessToken string
	ExpiresIn   int
}

type jwtClaims struct {
	Email   string `json:"email"`
	IsAdmin bool   `json:"is_admin"`
	Type    string `json:"type"` // "access" or "refresh"
	jwt.RegisteredClaims
}

// TokenClaims holds the public fields from a validated JWT.
type TokenClaims struct {
	Subject string
	JTI     string
	Email   string
	IsAdmin bool
	Exp     time.Time
}

type AuthUsecase struct {
	users          repository.UserRepository
	lock           LockStore
	jwtSecret      []byte
	accessTokenTTL time.Duration
	refreshTokenTTL time.Duration
}

func NewAuthUsecase(
	users repository.UserRepository,
	lock LockStore,
	jwtSecret string,
	accessTokenTTL time.Duration,
	refreshTokenTTL time.Duration,
) *AuthUsecase {
	return &AuthUsecase{
		users:           users,
		lock:            lock,
		jwtSecret:       []byte(jwtSecret),
		accessTokenTTL:  accessTokenTTL,
		refreshTokenTTL: refreshTokenTTL,
	}
}

func (uc *AuthUsecase) Register(ctx context.Context, in RegisterInput) (*AuthResult, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	user := &domain.User{
		ID:                uuid.New(),
		Email:             in.Email,
		PasswordHash:      string(hash),
		DisplayName:       in.DisplayName,
		PreferredLanguage: "en",
		IsActive:          true,
		CreatedAt:         time.Now().UTC(),
		UpdatedAt:         time.Now().UTC(),
	}

	if err := uc.users.Create(ctx, user); err != nil {
		return nil, err
	}

	return uc.buildAuthResult(user)
}

func (uc *AuthUsecase) Login(ctx context.Context, in LoginInput) (*AuthResult, error) {
	locked, err := uc.lock.IsLocked(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("check lock: %w", err)
	}
	if locked {
		return nil, ErrAccountLocked
	}

	user, err := uc.users.FindByEmail(ctx, in.Email)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil {
		// constant-time path: still do bcrypt to prevent timing attacks
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$dummy"), []byte(in.Password))
		return nil, ErrInvalidCredentials
	}

	if !user.IsActive {
		return nil, ErrAccountDisabled
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			_ = uc.lock.IncrFailCount(ctx, in.Email, failCountTTL)
			count, _ := uc.lock.GetFailCount(ctx, in.Email)
			if count >= maxLoginAttempts {
				_ = uc.lock.Lock(ctx, in.Email, lockDuration)
			}
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("compare password: %w", err)
	}

	_ = uc.lock.ResetFailCount(ctx, in.Email)
	_ = uc.users.UpdateLastLogin(ctx, user.ID)

	return uc.buildAuthResult(user)
}

func (uc *AuthUsecase) Logout(ctx context.Context, accessJTI string, accessExp time.Time) error {
	ttl := time.Until(accessExp)
	if ttl > 0 {
		if err := uc.lock.BlacklistToken(ctx, accessJTI, ttl); err != nil {
			return fmt.Errorf("blacklist token: %w", err)
		}
	}
	return nil
}

func (uc *AuthUsecase) Refresh(ctx context.Context, refreshToken string) (*RefreshResult, error) {
	claims, err := uc.parseToken(refreshToken)
	if err != nil {
		return nil, err
	}
	if claims.Type != "refresh" {
		return nil, ErrTokenInvalid
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, ErrTokenInvalid
	}

	user, err := uc.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if user == nil || !user.IsActive {
		return nil, ErrTokenInvalid
	}

	accessToken, _, err := uc.issueToken(user, "access", uc.accessTokenTTL)
	if err != nil {
		return nil, err
	}

	return &RefreshResult{
		AccessToken: accessToken,
		ExpiresIn:   int(uc.accessTokenTTL.Seconds()),
	}, nil
}

// ParseAccessToken validates an access token and returns its public claims.
func (uc *AuthUsecase) ParseAccessToken(tokenStr string) (*TokenClaims, error) {
	claims, err := uc.parseToken(tokenStr)
	if err != nil {
		return nil, err
	}
	if claims.Type != "access" {
		return nil, ErrTokenInvalid
	}
	exp := time.Time{}
	if claims.ExpiresAt != nil {
		exp = claims.ExpiresAt.Time
	}
	return &TokenClaims{
		Subject: claims.Subject,
		JTI:     claims.ID,
		Email:   claims.Email,
		IsAdmin: claims.IsAdmin,
		Exp:     exp,
	}, nil
}

func (uc *AuthUsecase) buildAuthResult(user *domain.User) (*AuthResult, error) {
	accessToken, _, err := uc.issueToken(user, "access", uc.accessTokenTTL)
	if err != nil {
		return nil, err
	}
	refreshToken, _, err := uc.issueToken(user, "refresh", uc.refreshTokenTTL)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int(uc.accessTokenTTL.Seconds()),
		User:         user,
	}, nil
}

func (uc *AuthUsecase) issueToken(user *domain.User, tokenType string, ttl time.Duration) (string, string, error) {
	jti := uuid.New().String()
	now := time.Now().UTC()
	claims := &jwtClaims{
		Email:   user.Email,
		IsAdmin: user.IsAdmin,
		Type:    tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:  user.ID.String(),
			ID:       jti,
			IssuedAt: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(uc.jwtSecret)
	if err != nil {
		return "", "", fmt.Errorf("sign token: %w", err)
	}
	return signed, jti, nil
}

func (uc *AuthUsecase) parseToken(tokenStr string) (*jwtClaims, error) {
	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return uc.jwtSecret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
