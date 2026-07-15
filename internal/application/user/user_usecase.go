package user

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gonszalito/go-ddd-architecture/internal/domain/user"
	"github.com/gonszalito/go-ddd-architecture/internal/shared"
	"golang.org/x/crypto/bcrypt"
)

type UserService struct {
	repo             UserRepository
	jwtSecret        []byte
	jwtExpiryMinutes int
}

type AuthResult struct {
	Token     string
	ExpiresAt time.Time
	UserID    string
	Email     string
	Role      string
}

func NewUserService(repo UserRepository, jwtSecret string, jwtExpiryMinutes int) *UserService {
	return &UserService{
		repo:             repo,
		jwtSecret:        []byte(jwtSecret),
		jwtExpiryMinutes: jwtExpiryMinutes,
	}
}

func (s *UserService) Register(ctx context.Context, name, email, password, role string) (*AuthResult, error) {
	if strings.TrimSpace(email) == "" || strings.TrimSpace(password) == "" {
		return nil, shared.NewAuthError("email and password are required")
	}
	normalizedRole := strings.ToUpper(strings.TrimSpace(role))
	if normalizedRole == "" {
		normalizedRole = user.RoleMerchant
	}
	if !isAllowedRole(normalizedRole) {
		return nil, shared.NewAuthError("role must be one of ADMIN, USER, MERCHANT")
	}

	_, err := s.repo.FindByEmail(email)
	if err == nil {
		return nil, shared.NewConflictError("merchant already exists with this email")
	}
	if !isUserNotFoundError(err) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := &user.User{
		ID:           uuid.NewString(),
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Role:         normalizedRole,
		CreatedAt:    time.Now().UTC(),
	}

	if err := s.repo.CreateWithMerchant(newUser); err != nil {
		return nil, err
	}

	return s.issueToken(newUser)
}

func isAllowedRole(role string) bool {
	switch role {
	case user.RoleAdmin, user.RoleUser, user.RoleMerchant:
		return true
	default:
		return false
	}
}

func isUserNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, sql.ErrNoRows) {
		return true
	}
	switch err.Error() {
	case "user not found", "not_found":
		return true
	default:
		return false
	}
}

func (s *UserService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	u, err := s.repo.FindByEmail(email)
	if err != nil {
		logLoginAudit(ctx, "", email, false)
		return nil, shared.NewAuthError("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		logLoginAudit(ctx, u.ID, email, false)
		return nil, shared.NewAuthError("invalid credentials")
	}

	logLoginAudit(ctx, u.ID, email, true)
	return s.issueToken(u)
}

func (s *UserService) Refresh(ctx context.Context, userID string) (*AuthResult, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, shared.NewAuthError("invalid user context")
	}
	u, err := s.repo.FindByID(userID)
	if err != nil {
		return nil, shared.NewAuthError("invalid user context")
	}
	return s.issueToken(u)
}

func (s *UserService) issueToken(u *user.User) (*AuthResult, error) {
	expiresAt := time.Now().UTC().Add(time.Duration(s.jwtExpiryMinutes) * time.Minute)
	claims := jwt.MapClaims{
		"sub":   u.ID,
		"email": u.Email,
		"role":  u.Role,
		"exp":   expiresAt.Unix(),
		"iat":   time.Now().UTC().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token:     tokenString,
		ExpiresAt: expiresAt,
		UserID:    u.ID,
		Email:     u.Email,
		Role:      u.Role,
	}, nil
}

func logLoginAudit(ctx context.Context, userID, email string, success bool) {
	entry := map[string]any{
		"event":      "login_attempt",
		"user_id":    userID,
		"email":      email,
		"success":    success,
		"occurredAt": time.Now().UTC().Format(time.RFC3339),
	}
	if b, err := json.Marshal(entry); err == nil {
		log.Println(string(b))
	}
}
