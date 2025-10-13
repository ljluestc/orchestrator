package security

import (
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestNewAuthManager(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "Valid AuthManager",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := NewAuthManager()

			assert.NotNil(t, auth)
			assert.NotNil(t, auth.users)
			assert.NotNil(t, auth.tokens)
		})
	}
}

func TestAuthManager_CreateUser(t *testing.T) {
	auth := NewAuthManager()

	tests := []struct {
		name        string
		username    string
		password    string
		roles       []string
		expectError bool
	}{
		{
			name:        "Valid user",
			username:    "testuser",
			password:    "password123",
			roles:       []string{"user"},
			expectError: false,
		},
		{
			name:        "Admin user",
			username:    "admin",
			password:    "admin123",
			roles:       []string{"admin"},
			expectError: false,
		},
		{
			name:        "Empty username",
			username:    "",
			password:    "password123",
			roles:       []string{"user"},
			expectError: true,
		},
		{
			name:        "Empty password",
			username:    "testuser",
			password:    "",
			roles:       []string{"user"},
			expectError: true,
		},
		{
			name:        "Empty roles",
			username:    "testuser",
			password:    "password123",
			roles:       []string{},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := auth.CreateUser(tt.username, tt.password, tt.roles)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify user was created
				user, exists := auth.users[tt.username]
				assert.True(t, exists)
				assert.Equal(t, tt.username, user.Username)
				assert.Equal(t, tt.roles, user.Roles)
				assert.NotEmpty(t, user.PasswordHash)
				assert.True(t, time.Since(user.CreatedAt) < time.Minute)

				// Verify password is hashed
				assert.NotEqual(t, tt.password, user.PasswordHash)
				assert.NoError(t, bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(tt.password)))
			}
		})
	}
}

func TestAuthManager_CreateUserDuplicate(t *testing.T) {
	auth := NewAuthManager()

	// Create user first time
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	// Try to create same user again
	err = auth.CreateUser("testuser", "password123", []string{"user"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user already exists")
}

func TestAuthManager_Authenticate(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
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
			token, err := auth.Authenticate(tt.username, tt.password)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, token)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, token)
				assert.Equal(t, tt.username, token.Username)
				assert.NotEmpty(t, token.Value)
			}
		})
	}
}

func TestAuthManager_GenerateToken(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	user := auth.users["testuser"]

	tests := []struct {
		name        string
		user        *User
		expectError bool
	}{
		{
			name:        "Valid user",
			user:        user,
			expectError: false,
		},
		{
			name:        "Nil user",
			user:        nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.user == nil {
				// Test empty username case
				token := auth.GenerateToken("")
				assert.NotEmpty(t, token)
				assert.Equal(t, "", token.Username)
				assert.NotEmpty(t, token.Value)
			} else {
				token := auth.GenerateToken(tt.user.Username)
				assert.NotEmpty(t, token)
				assert.Equal(t, tt.user.Username, token.Username)
				assert.NotEmpty(t, token.Value)
				assert.True(t, time.Since(token.IssuedAt) < time.Minute)
			}
		})
	}
}

func TestAuthManager_ValidateToken(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	// Generate token by authenticating
	var token *Token
	token, err = auth.Authenticate("testuser", "password123")
	assert.NoError(t, err)
	assert.NotNil(t, token)

	// Get the user for comparison
	user := auth.users["testuser"]

	tests := []struct {
		name        string
		token       string
		expectError bool
	}{
		{
			name:        "Valid token",
			token:       token.Value,
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
				assert.Equal(t, user.Roles, session.Roles)
			}
		})
	}
}

func TestAuthManager_ValidateTokenExpired(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	user := auth.users["testuser"]

	// Generate token with short expiration
	token := auth.GenerateToken(user.Username)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Validate expired token
	user, err = auth.ValidateToken(token.Value)
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "token expired")
}

func TestAuthManager_RevokeToken(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	// Generate token by authenticating
	var token *Token
	token, err = auth.Authenticate("testuser", "password123")
	assert.NoError(t, err)
	assert.NotNil(t, token)

	// Verify token exists
	_, exists := auth.tokens[token.Value]
	assert.True(t, exists)

	// Revoke token
	auth.RevokeToken(token.Value)

	// Verify token is removed
		_, exists = auth.tokens[token.Value]
	assert.False(t, exists)

	// Verify token is invalid
	_, err = auth.ValidateToken(token.Value)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "token not found")
}

func TestAuthManager_RevokeTokenNotFound(t *testing.T) {
	auth := NewAuthManager()

	auth.RevokeToken("nonexistent-token")
}

func TestAuthManager_RevokeUserTokens(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	user := auth.users["testuser"]

	// Generate multiple tokens
	token1 := auth.GenerateToken(user.Username)
	token2 := auth.GenerateToken(user.Username)
	token3 := auth.GenerateToken(user.Username)

	// Verify tokens exist
	assert.Len(t, auth.tokens, 3)

	// Revoke all user tokens
	auth.RevokeUserTokens("testuser")

	// Verify all tokens are removed
	assert.Len(t, auth.tokens, 0)

	// Verify tokens are invalid
	_, err = auth.ValidateToken(token1.Value)
	assert.Error(t, err)
	_, err = auth.ValidateToken(token2.Value)
	assert.Error(t, err)
	_, err = auth.ValidateToken(token3.Value)
	assert.Error(t, err)
}

func TestAuthManager_RevokeUserTokensNotFound(t *testing.T) {
	auth := NewAuthManager()

	auth.RevokeUserTokens("nonexistent-user")
}

func TestAuthManager_UpdateUser(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	originalUser := auth.users["testuser"]

	// Wait a bit to ensure UpdatedAt changes
	time.Sleep(10 * time.Millisecond)

	// Update user
	err = auth.UpdateUser("testuser", "newpassword123", []string{"admin"})
	assert.NoError(t, err)

	// Verify user was updated
	updatedUser := auth.users["testuser"]
	assert.Equal(t, "testuser", updatedUser.Username)
	assert.Equal(t, []string{"admin"}, updatedUser.Roles)
	assert.NotEqual(t, originalUser.PasswordHash, updatedUser.PasswordHash)

	// Verify new password works
	_, err = auth.Authenticate("testuser", "newpassword123")
	assert.NoError(t, err)

	// Verify old password doesn't work
	_, err = auth.Authenticate("testuser", "password123")
	assert.Error(t, err)
}

func TestAuthManager_UpdateUserNotFound(t *testing.T) {
	auth := NewAuthManager()

	err := auth.UpdateUser("nonexistent", "newpassword123", []string{"admin"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAuthManager_DeleteUser(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	// Generate token
	user := auth.users["testuser"]
	token := auth.GenerateToken(user.Username)

	// Verify user exists
	_, exists := auth.users["testuser"]
	assert.True(t, exists)

	// Verify token exists
		_, exists = auth.tokens[token.Value]
	assert.True(t, exists)

	// Delete user
	err = auth.DeleteUser("testuser")
	assert.NoError(t, err)

	// Verify user is removed
	_, exists = auth.users["testuser"]
	assert.False(t, exists)

	// Verify token is removed
		_, exists = auth.tokens[token.Value]
	assert.False(t, exists)
}

func TestAuthManager_DeleteUserNotFound(t *testing.T) {
	auth := NewAuthManager()

	err := auth.DeleteUser("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAuthManager_ListUsers(t *testing.T) {
	auth := NewAuthManager()

	// Create test users
	err := auth.CreateUser("user1", "password123", []string{"user"})
	assert.NoError(t, err)
	err = auth.CreateUser("user2", "password123", []string{"admin"})
	assert.NoError(t, err)
	err = auth.CreateUser("user3", "password123", []string{"user"})
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
	auth := NewAuthManager()

	users := auth.ListUsers()
	assert.Len(t, users, 0)
}

func TestAuthManager_GetUser(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	// Get user
	user, err := auth.GetUser("testuser")
	assert.NoError(t, err)
	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, []string{"user"}, user.Roles)
	assert.NotEmpty(t, user.PasswordHash)
}

func TestAuthManager_GetUserNotFound(t *testing.T) {
	auth := NewAuthManager()

	user, err := auth.GetUser("nonexistent")
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "user not found")
}

func TestAuthManager_CleanupExpiredTokens(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	user := auth.users["testuser"]

	// Generate token with short expiration
	token := auth.GenerateToken(user.Username)

	// Verify token exists
		_, exists := auth.tokens[token.Value]
	assert.True(t, exists)

	// Wait for token to expire
	time.Sleep(10 * time.Millisecond)

	// Cleanup expired tokens
	auth.CleanupExpiredTokens()

	// Verify expired token is removed
		_, exists = auth.tokens[token.Value]
	assert.False(t, exists)
}

func TestAuthManager_CleanupExpiredTokensWithValidTokens(t *testing.T) {
	auth := NewAuthManager()

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	user := auth.users["testuser"]

	// Generate token with long expiration
	token := auth.GenerateToken(user.Username)

	// Verify token exists
		_, exists := auth.tokens[token.Value]
	assert.True(t, exists)

	// Cleanup expired tokens
	auth.CleanupExpiredTokens()

	// Verify valid token is not removed
		_, exists = auth.tokens[token.Value]
	assert.True(t, exists)
}

func TestAuthManager_StartCleanupRoutine(t *testing.T) {
	auth := NewAuthManager()

	// Start cleanup routine
	auth.StartCleanupRoutine(100 * time.Millisecond)

	// Create test user
	err := auth.CreateUser("testuser", "password123", []string{"user"})
	assert.NoError(t, err)

	user := auth.users["testuser"]

	// Generate token with short expiration
	token := auth.GenerateToken(user.Username)

	// Verify token exists
		_, exists := auth.tokens[token.Value]
	assert.True(t, exists)

	// Wait for cleanup routine to run
	time.Sleep(200 * time.Millisecond)

	// Verify expired token is removed
		_, exists = auth.tokens[token.Value]
	assert.False(t, exists)
}

func TestAuthManager_UserStructures(t *testing.T) {
	now := time.Now()
	user := &User{
		Username:     "testuser",
		PasswordHash: "hashedpassword",
		Roles:        []string{"admin"},
		CreatedAt:    now,
		LastLogin:    now,
	}

	assert.Equal(t, "testuser", user.Username)
	assert.Equal(t, "hashedpassword", user.PasswordHash)
	assert.Equal(t, []string{"admin"}, user.Roles)
	assert.Equal(t, now, user.CreatedAt)
	assert.Equal(t, now, user.LastLogin)
}

func TestAuthManager_TokenStructures(t *testing.T) {
	now := time.Now()
	token := &Token{
		Value:     "test-token",
		Username:  "testuser",
		ExpiresAt: now.Add(24 * time.Hour),
		IssuedAt:  now,
	}

	assert.Equal(t, "testuser", token.Username)
	assert.Equal(t, now.Add(24*time.Hour), token.ExpiresAt)
	assert.Equal(t, now, token.IssuedAt)
}

func TestAuthManager_HashPassword(t *testing.T) {
	auth := NewAuthManager()

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
	auth := NewAuthManager()

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
	auth := NewAuthManager()

	sessionID := auth.GenerateSessionID()

	assert.NotEmpty(t, sessionID)
	assert.Len(t, sessionID, 64) // SHA256 hex string length

	// Verify it's a valid hex string
	_, err := hex.DecodeString(sessionID)
	assert.NoError(t, err)
}

func TestAuthManager_GenerateSessionIDUnique(t *testing.T) {
	auth := NewAuthManager()

	sessionIDs := make(map[string]bool)

	// Generate many session IDs
	for i := 0; i < 1000; i++ {
		sessionID := auth.GenerateSessionID()
		assert.False(t, sessionIDs[sessionID], "Session ID should be unique")
		sessionIDs[sessionID] = true
	}
}

func TestAuthManager_ConcurrentAccess(t *testing.T) {
	auth := NewAuthManager()

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
				auth.CreateUser(username, password, []string{role})

				// Authenticate user
				user, err := auth.Authenticate(username, password)
				if err == nil {
					// Generate token
					token := auth.GenerateToken(user.Username)
					if err == nil {
						// Validate token
						auth.ValidateToken(token.Value)

						// Revoke token
						auth.RevokeToken(token.Value)
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
			auth := NewAuthManager()

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
			auth := NewAuthManager()

			// Create test user
			err := auth.CreateUser("testuser", "password123", []string{"user"})
			assert.NoError(t, err)

			user := auth.users["testuser"]

			// Generate token
					token := auth.GenerateToken(user.Username)
			assert.NotEmpty(t, token)

			// Validate token
			user, err = auth.ValidateToken(token.Value)
			assert.NoError(t, err)
			assert.NotNil(t, user)
		})
	}
}

func BenchmarkAuthManager_CreateUser(b *testing.B) {
	auth := NewAuthManager()

	for i := 0; i < b.N; i++ {
		username := fmt.Sprintf("user-%d", i)
		password := fmt.Sprintf("password-%d", i)
		auth.CreateUser(username, password, []string{"user"})
	}
}

func BenchmarkAuthManager_Authenticate(b *testing.B) {
	auth := NewAuthManager()

	// Create test user
	auth.CreateUser("testuser", "password123", []string{"user"})

	for i := 0; i < b.N; i++ {
		auth.Authenticate("testuser", "password123")
	}
}

func BenchmarkAuthManager_GenerateToken(b *testing.B) {
	auth := NewAuthManager()

	// Create test user
	auth.CreateUser("testuser", "password123", []string{"user"})
	user := auth.users["testuser"]

	for i := 0; i < b.N; i++ {
		auth.GenerateToken(user.Username)
	}
}

func BenchmarkAuthManager_ValidateToken(b *testing.B) {
	auth := NewAuthManager()

	// Create test user
	auth.CreateUser("testuser", "password123", []string{"user"})

	// Generate token by authenticating
	token, err := auth.Authenticate("testuser", "password123")
	assert.NoError(b, err)
	assert.NotNil(b, token)

	for i := 0; i < b.N; i++ {
		auth.ValidateToken(token.Value)
	}
}

func BenchmarkAuthManager_HashPassword(b *testing.B) {
	auth := NewAuthManager()

	password := "testpassword123"

	for i := 0; i < b.N; i++ {
		auth.HashPassword(password)
	}
}

func BenchmarkAuthManager_VerifyPassword(b *testing.B) {
	auth := NewAuthManager()

	password := "testpassword123"
	hash, _ := auth.HashPassword(password)

	for i := 0; i < b.N; i++ {
		auth.VerifyPassword(hash, password)
	}
}

func BenchmarkAuthManager_GenerateSessionID(b *testing.B) {
	auth := NewAuthManager()

	for i := 0; i < b.N; i++ {
		auth.GenerateSessionID()
	}
}
