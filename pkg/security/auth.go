// Package security provides authentication and authorization (Task 21)
package security

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrTokenExpired      = errors.New("token expired")
)

// User represents a system user
type User struct {
	Username     string
	PasswordHash string
	Roles        []string
	Permissions  []string
	CreatedAt    time.Time
	LastLogin    time.Time
}

// Token represents an authentication token
type Token struct {
	Value     string
	Username  string
	ExpiresAt time.Time
	IssuedAt  time.Time
}

// AuthManager manages authentication and authorization
type AuthManager struct {
	users  map[string]*User
	tokens map[string]*Token
	mu     sync.RWMutex
}

// NewAuthManager creates a new authentication manager
func NewAuthManager() *AuthManager {
	return &AuthManager{
		users:  make(map[string]*User),
		tokens: make(map[string]*Token),
	}
}

// CreateUser creates a new user with hashed password
func (am *AuthManager) CreateUser(username, password string, roles []string) error {
	am.mu.Lock()
	defer am.mu.Unlock()

	if _, exists := am.users[username]; exists {
		return fmt.Errorf("user %s already exists", username)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	am.users[username] = &User{
		Username:     username,
		PasswordHash: string(hash),
		Roles:        roles,
		Permissions:  getRolePermissions(roles),
		CreatedAt:    time.Now(),
	}

	return nil
}

// Authenticate verifies username and password
func (am *AuthManager) Authenticate(username, password string) (*Token, error) {
	am.mu.RLock()
	user, exists := am.users[username]
	am.mu.RUnlock()

	if !exists {
		return nil, ErrInvalidCredentials
	}

	err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	// Generate token
	token := am.generateToken(username)

	am.mu.Lock()
	user.LastLogin = time.Now()
	am.tokens[token.Value] = token
	am.mu.Unlock()

	return token, nil
}

// ValidateToken validates an authentication token
func (am *AuthManager) ValidateToken(tokenValue string) (*User, error) {
	am.mu.RLock()
	defer am.mu.RUnlock()

	token, exists := am.tokens[tokenValue]
	if !exists {
		return nil, ErrUnauthorized
	}

	if time.Now().After(token.ExpiresAt) {
		return nil, ErrTokenExpired
	}

	user, exists := am.users[token.Username]
	if !exists {
		return nil, ErrUnauthorized
	}

	return user, nil
}

// Authorize checks if user has required permission
func (am *AuthManager) Authorize(ctx context.Context, tokenValue, permission string) error {
	user, err := am.ValidateToken(tokenValue)
	if err != nil {
		return err
	}

	if !hasPermission(user, permission) {
		return ErrUnauthorized
	}

	return nil
}

// generateToken generates a new authentication token
func (am *AuthManager) generateToken(username string) *Token {
	now := time.Now()
	data := fmt.Sprintf("%s:%d", username, now.UnixNano())
	hash := sha256.Sum256([]byte(data))
	
	return &Token{
		Value:     hex.EncodeToString(hash[:]),
		Username:  username,
		IssuedAt:  now,
		ExpiresAt: now.Add(24 * time.Hour), // 24 hour expiration
	}
}

// getRolePermissions returns permissions for given roles
func getRolePermissions(roles []string) []string {
	permMap := make(map[string]bool)
	
	for _, role := range roles {
		switch role {
		case "admin":
			permMap["read"] = true
			permMap["write"] = true
			permMap["delete"] = true
			permMap["manage_users"] = true
			permMap["deploy"] = true
		case "operator":
			permMap["read"] = true
			permMap["write"] = true
			permMap["deploy"] = true
		case "viewer":
			permMap["read"] = true
		}
	}

	perms := make([]string, 0, len(permMap))
	for perm := range permMap {
		perms = append(perms, perm)
	}
	return perms
}

// hasPermission checks if user has specific permission
func hasPermission(user *User, permission string) bool {
	for _, perm := range user.Permissions {
		if perm == permission {
			return true
		}
	}
	return false
}

// RevokeToken revokes an authentication token
func (am *AuthManager) RevokeToken(tokenValue string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	delete(am.tokens, tokenValue)
}

// CleanupExpiredTokens removes expired tokens
func (am *AuthManager) CleanupExpiredTokens() {
	am.mu.Lock()
	defer am.mu.Unlock()

	now := time.Now()
	for tokenValue, token := range am.tokens {
		if now.After(token.ExpiresAt) {
			delete(am.tokens, tokenValue)
		}
	}
}

// StartTokenCleanup starts periodic token cleanup
func (am *AuthManager) StartTokenCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.CleanupExpiredTokens()
		}
	}
}
