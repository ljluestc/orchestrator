package security

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestNewAuthManager(t *testing.T) {
	tests := []struct {
		name     string
		jwtKey   string
		bcryptCost int
	}{
		{
			name:       "Valid AuthManager",
			jwtKey:     "test-secret-key",
			bcryptCost: bcrypt.DefaultCost,
		},
		{
			name:       "Empty JWT Key",
			jwtKey:     "",
			bcryptCost: bcrypt.DefaultCost,
		},
		{
			name:       "Custom BCrypt Cost",
			jwtKey:     "test-secret-key",
			bcryptCost: 12,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthManager(tt.jwtKey, tt.bcryptCost)

			assert.Equal(t, tt.jwtKey, auth.JWTKey)
			assert.Equal(t, tt.bcryptCost, auth.BCryptCost)
			assert.NotNil(t, auth.Users)
			assert.NotNil(t, auth.Sessions)
		})
	}
}

func TestAuthManager_CreateUser(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	tests := []struct {
		name        string
		username    string
		password    string
		role        string
		expectError bool
	}{
		{
			name:        "Valid user",
			username:    "testuser",
			password:    "password123",
			role:        "user",
			expectError: false,
		},
		{
			name:        "Admin user",
			username:    "admin",
			password:    "admin123",
			role:        "admin",
			expectError: false,
		},
		{
			name:        "Empty username",
			username:    "",
			password:    "password123",
			role:        "user",
			expectError: true,
		},
		{
			name:        "Empty password",
			username:    "testuser",
			password:    "",
			role:        "user",
			expectError: true,
		},
		{
			name:        "Empty role",
			username:    "testuser",
			password:    "password123",
			role:        "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.CreateUser(tt.username, tt.password, tt.role)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify user was created
				user, exists := auth.Users[tt.username]
				assert.True(t, exists)
				assert.Equal(t, tt.username, user.Username)
				assert.Equal(t, tt.role, user.Role)
				assert.NotEmpty(t, user.PasswordHash)
				assert.True(t, time.Since(user.CreatedAt) < time.Minute)
				assert.True(t, time.Since(user.UpdatedAt) < time.Minute)

				// Verify password is hashed
				assert.NotEqual(t, tt.password, user.PasswordHash)
				assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.password)))
			}
		})
	}
}

func TestAuthManager_CreateUserDuplicate(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create user first time
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	// Try to create same user again
	err = auth.CreateUser("testuser", "password123", "user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user already exists")
}

func TestAuthManager_AuthenticateUser(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	tests := []struct {
		name        string
		username    string
		password    string
		expectError bool
	}{
		{
			name:        "Valid credentials",
			username:    "testuser",
			password:    "password123",
			expectError: false,
		},
		{
			name:        "Invalid username",
			username:    "nonexistent",
			password:    "password123",
			expectError: true,
		},
		{
			name:        "Invalid password",
			username:    "testuser",
			password:    "wrongpassword",
			expectError: true,
		},
		{
			name:        "Empty username",
			username:    "",
			password:    "password123",
			expectError: true,
		},
		{
			name:        "Empty password",
			username:    "testuser",
			password:    "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user, err := auth.AuthenticateUser(tt.username, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, user)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, user)
				assert.Equal(t, tt.username, user.Username)
				assert.Equal(t, "user", user.Role)
			}
		})
	}
}

func TestAuthManager_GenerateToken(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	user := auth.Users["testuser"]

	tests := []struct {
		name        string
		user        *User
		expiration time.Duration
		expectError bool
	}{
		{
			name:        "Valid user with default expiration",
			user:        user,
			expiration: 0,
			expectError: false,
		},
		{
			name:        "Valid user with custom expiration",
			user:        user,
			expiration: 2 * time.Hour,
			expectError: false,
		},
		{
			name:        "Nil user",
			user:        nil,
			expiration: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := auth.GenerateToken(tt.user, tt.expiration)

			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, token)

				// Verify token is stored in sessions
				session, exists := auth.Sessions[token]
				assert.True(t, exists)
				assert.Equal(t, tt.user.Username, session.Username)
				assert.Equal(t, tt.user.Role, session.Role)
				assert.True(t, time.Since(session.CreatedAt) < time.Minute)

				// Verify expiration
				if tt.expiration == 0 {
					assert.True(t, time.Until(session.ExpiresAt) < 25*time.Hour) // Default 24h + buffer
				} else {
					assert.True(t, time.Until(session.ExpiresAt) < tt.expiration+time.Minute)
				}
			}
		})
	}
}

func TestAuthManager_ValidateToken(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	user := auth.Users["testuser"]

	// Generate token
	token, err := auth.GenerateToken(user, 0)
	assert.NoError(t, err)

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Valid token",
			token:       token,
			expectError: false,
		},
		{
			name:        "Invalid token",
			token:       "invalid-token",
			expectError: true,
		},
		{
			name:        "Empty token",
			token:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			session, err := auth.ValidateToken(tt.token)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, session)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, session)
				assert.Equal(t, user.Username, session.Username)
				assert.Equal(t, user.Role, session.Role)
			}
		})
	}
}

func TestAuthManager_ValidateTokenExpired(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	user := auth.Users["testuser"]

	// Generate token with short expiration
	token, err := auth.GenerateToken(user, 1*time.Millisecond)
	assert.NoError(t, err)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validate expired token
	session, err := auth.ValidateToken(token)
	assert.Error(t, err)
	assert.Nil(t, session)
	assert.Contains(t, err.Error(), "token expired")
}

func TestAuthManager_RevokeToken(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	user := auth.Users["testuser"]

	// Generate token
	token, err := auth.GenerateToken(user, 0)
	assert.NoError(t, err)

	// Verify token exists
	_, exists := auth.Sessions[token]
	assert.True(t, exists)

	// Revoke token
	err = auth.RevokeToken(token)
	assert.NoError(t, err)

	// Verify token is removed
	_, exists = auth.Sessions[token]
	assert.False(t, exists)

	// Verify token is invalid
	_, err = auth.ValidateToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token not found")
}

func TestAuthManager_RevokeTokenNotFound(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	err := auth.RevokeToken("nonexistent-token")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token not found")
}

func TestAuthManager_RevokeUserTokens(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	user := auth.Users["testuser"]

	// Generate multiple tokens
	token1, err := auth.GenerateToken(user, 0)
	assert.NoError(t, err)
	token2, err := auth.GenerateToken(user, 0)
	assert.NoError(t, err)
	token3, err := auth.GenerateToken(user, 0)
	assert.NoError(t, err)

	// Verify tokens exist
	assert.Len(t, auth.Sessions, 3)

	// Revoke all user tokens
	err = auth.RevokeUserTokens("testuser")
	assert.NoError(t, err)

	// Verify all tokens are removed
	assert.Len(t, auth.Sessions, 0)

	// Verify tokens are invalid
	_, err = auth.ValidateToken(token1)
	assert.Error(t, err)
	_, err = auth.ValidateToken(token2)
	assert.Error(t, err)
	_, err = auth.ValidateToken(token3)
	assert.Error(t, err)
}

func TestAuthManager_RevokeUserTokensNotFound(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	err := auth.RevokeUserTokens("nonexistent-user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAuthManager_UpdateUser(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	originalUser := auth.Users["testuser"]
	originalUpdatedAt := originalUser.UpdatedAt

	// Wait a bit to ensure UpdatedAt changes
	time.Sleep(10 * time.Millisecond)

	// Update user
	err = auth.UpdateUser("testuser", "newpassword123", "admin")
	assert.NoError(t, err)

	// Verify user was updated
	updatedUser := auth.Users["testuser"]
	assert.Equal(t, "testuser", updatedUser.Username)
	assert.Equal(t, "admin", updatedUser.Role)
	assert.NotEqual(t, originalUser.PasswordHash, updatedUser.PasswordHash)
	assert.True(t, updatedUser.UpdatedAt.After(originalUpdatedAt))

	// Verify new password works
	_, err = auth.AuthenticateUser("testuser", "newpassword123")
	assert.NoError(t, err)

	// Verify old password doesn't work
	_, err = auth.AuthenticateUser("testuser", "password123")
	assert.Error(t, err)
}

func TestAuthManager_UpdateUserNotFound(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	err := auth.UpdateUser("nonexistent", "newpassword123", "admin")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAuthManager_DeleteUser(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	// Generate token
	user := auth.Users["testuser"]
	token, err := auth.GenerateToken(user, 0)
	assert.NoError(t, err)

	// Verify user exists
	_, exists := auth.Users["testuser"]
	assert.True(t, exists)

	// Verify token exists
	_, exists = auth.Sessions[token]
	assert.True(t, exists)

	// Delete user
	err = auth.DeleteUser("testuser")
	assert.NoError(t, err)

	// Verify user is removed
	_, exists = auth.Users["testuser"]
	assert.False(t, exists)

	// Verify token is removed
	_, exists = auth.Sessions[token]
	assert.False(t, exists)
}

func TestAuthManager_DeleteUserNotFound(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	err := auth.DeleteUser("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAuthManager_ListUsers(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test users
	err := auth.CreateUser("user1", "password123", "user")
	assert.NoError(t, err)
	err = auth.CreateUser("user2", "password123", "admin")
	assert.NoError(t, err)
	err = auth.CreateUser("user3", "password123", "user")
	assert.NoError(t, err)

	// List users
	users := auth.ListUsers()

	assert.Len(t, users, 3)

	// Verify users are returned
	usernames := make(map[string]bool)
	for _, user := range users {
		usernames[user.Username] = true
	}

	assert.True(t, usernames["user1"])
	assert.True(t, usernames["user2"])
	assert.True(t, usernames["user3"])
}

func TestAuthManager_ListUsersEmpty(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	users := auth.ListUsers()
	assert.Len(t, users, 0)
}

func TestAuthManager_GetUser(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	// Get user
	user, err := auth.GetUser("testuser")
	assert.NoError(t, err)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "user", user.Role)
	assert.NotEmpty(t, user.PasswordHash)
}

func TestAuthManager_GetUserNotFound(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	user, err := auth.GetUser("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAuthManager_CleanupExpiredTokens(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	user := auth.Users["testuser"]

	// Generate token with short expiration
	token, err := auth.GenerateToken(user, 1*time.Millisecond)
	assert.NoError(t, err)

	// Verify token exists
	_, exists := auth.Sessions[token]
	assert.True(t, exists)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Cleanup expired tokens
	auth.CleanupExpiredTokens()

	// Verify expired token is removed
	_, exists = auth.Sessions[token]
	assert.False(t, exists)
}

func TestAuthManager_CleanupExpiredTokensWithValidTokens(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	user := auth.Users["testuser"]

	// Generate token with long expiration
	token, err := auth.GenerateToken(user, 24*time.Hour)
	assert.NoError(t, err)

	// Verify token exists
	_, exists := auth.Sessions[token]
	assert.True(t, exists)

	// Cleanup expired tokens
	auth.CleanupExpiredTokens()

	// Verify valid token is not removed
	_, exists = auth.Sessions[token]
	assert.True(t, exists)
}

func TestAuthManager_StartCleanupRoutine(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Start cleanup routine
	auth.StartCleanupRoutine(100 * time.Millisecond)

	// Create test user
	err := auth.CreateUser("testuser", "password123", "user")
	assert.NoError(t, err)

	user := auth.Users["testuser"]

	// Generate token with short expiration
	token, err := auth.GenerateToken(user, 1*time.Millisecond)
	assert.NoError(t, err)

	// Verify token exists
	_, exists := auth.Sessions[token]
	assert.True(t, exists)

	// Wait for cleanup routine to run
	time.Sleep(200 * time.Millisecond)

	// Verify expired token is removed
	_, exists = auth.Sessions[token]
	assert.False(t, exists)
}

func TestAuthManager_UserStructures(t *testing.T) {
	now := time.Now()
	user := &User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Role:         "admin",
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "hashedpassword", user.PasswordHash)
	assert.Equal(t, "admin", user.Role)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.UpdatedAt)
}

func TestAuthManager_SessionStructures(t *testing.T) {
	now := time.Now()
	session := &Session{
		Username:  "testuser",
		Role:      "admin",
		CreatedAt: now,
		ExpiresAt: now.Add(24 * time.Hour),
	}

	assert.Equal(t, "testuser", session.Username)
	assert.Equal(t, "admin", session.Role)
	assert.Equal(t, now, session.CreatedAt)
	assert.Equal(t, now.Add(24*time.Hour), session.ExpiresAt)
}

func TestAuthManager_HashPassword(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	password := "testpassword123"
	hash, err := auth.HashPassword(password)

	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Verify hash can be compared
	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	assert.NoError(t, err)
}

func TestAuthManager_VerifyPassword(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	password := "testpassword123"
	hash, err := auth.HashPassword(password)
	assert.NoError(t, err)

	// Test valid password
	err = auth.VerifyPassword(hash, password)
	assert.NoError(t, err)

	// Test invalid password
	err = auth.VerifyPassword(hash, "wrongpassword")
	assert.Error(t, err)
}

func TestAuthManager_GenerateSessionID(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	sessionID := auth.GenerateSessionID()

	assert.NotEmpty(t, sessionID)
	assert.Len(t, sessionID, 64) // SHA256 hex string length

	// Verify it's a valid hex string
	_, err := hex.DecodeString(sessionID)
	assert.NoError(t, err)
}

func TestAuthManager_GenerateSessionIDUnique(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	sessionIDs := make(map[string]bool)

	// Generate many session IDs
	for i := 0; i < 1000; i++ {
		sessionID := auth.GenerateSessionID()
		assert.False(t, sessionIDs[sessionID], "Session ID should be unique")
		sessionIDs[sessionID] = true
	}
}

func TestAuthManager_ConcurrentAccess(t *testing.T) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	const numGoroutines = 10
	const numOperations = 100

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			for j := 0; j < numOperations; j++ {
				username := fmt.Sprintf("user-%d", j)
				password := fmt.Sprintf("password-%d", j)
				role := "user"

				// Create user
				auth.CreateUser(username, password, role)

				// Authenticate user
				user, err := auth.AuthenticateUser(username, password)
				if err == nil {
					// Generate token
					token, err := auth.GenerateToken(user, 0)
					if err == nil {
						// Validate token
						auth.ValidateToken(token)

						// Revoke token
						auth.RevokeToken(token)
					}
				}

				// Delete user
				auth.DeleteUser(username)
			}
			done <- true
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}
}

func TestAuthManager_BCryptCostVariations(t *testing.T) {
	tests := []struct {
		name string
		cost int
	}{
		{"MinCost", bcrypt.MinCost},
		{"DefaultCost", bcrypt.DefaultCost},
		{"MaxCost", bcrypt.MaxCost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthManager("test-secret-key", tt.cost)

			password := "testpassword123"
			hash, err := auth.HashPassword(password)

			assert.NoError(t, err)
			assert.NotEmpty(t, hash)

			// Verify hash can be compared
			err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
			assert.NoError(t, err)
		})
	}
}

func TestAuthManager_JWTKeyVariations(t *testing.T) {
	tests := []struct {
		name   string
		jwtKey string
	}{
		{"Short Key", "short"},
		{"Long Key", "this-is-a-very-long-jwt-secret-key-that-should-work-fine"},
		{"Empty Key", ""},
		{"Special Chars", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthManager(tt.jwtKey, bcrypt.DefaultCost)

			// Create test user
			err := auth.CreateUser("testuser", "password123", "user")
			assert.NoError(t, err)

			user := auth.Users["testuser"]

			// Generate token
			token, err := auth.GenerateToken(user, 0)
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Validate token
			session, err := auth.ValidateToken(token)
			assert.NoError(t, err)
			assert.NotNil(t, session)
		})
	}
}

func BenchmarkAuthManager_CreateUser(b *testing.B) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	for i := 0; i < b.N; i++ {
		username := fmt.Sprintf("user-%d", i)
		password := fmt.Sprintf("password-%d", i)
		auth.CreateUser(username, password, "user")
	}
}

func BenchmarkAuthManager_AuthenticateUser(b *testing.B) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	auth.CreateUser("testuser", "password123", "user")

	for i := 0; i < b.N; i++ {
		auth.AuthenticateUser("testuser", "password123")
	}
}

func BenchmarkAuthManager_GenerateToken(b *testing.B) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	auth.CreateUser("testuser", "password123", "user")
	user := auth.Users["testuser"]

	for i := 0; i < b.N; i++ {
		auth.GenerateToken(user, 0)
	}
}

func BenchmarkAuthManager_ValidateToken(b *testing.B) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	// Create test user
	auth.CreateUser("testuser", "password123", "user")
	user := auth.Users["testuser"]

	// Generate token
	token, _ := auth.GenerateToken(user, 0)

	for i := 0; i < b.N; i++ {
		auth.ValidateToken(token)
	}
}

func BenchmarkAuthManager_HashPassword(b *testing.B) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	password := "testpassword123"

	for i := 0; i < b.N; i++ {
		auth.HashPassword(password)
	}
}

func BenchmarkAuthManager_VerifyPassword(b *testing.B) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	password := "testpassword123"
	hash, _ := auth.HashPassword(password)

	for i := 0; i < b.N; i++ {
		auth.VerifyPassword(hash, password)
	}
}

func BenchmarkAuthManager_GenerateSessionID(b *testing.B) {
	auth := NewAuthManager("test-secret-key", bcrypt.DefaultCost)

	for i := 0; i < b.N; i++ {
		auth.GenerateSessionID()
	}
}
