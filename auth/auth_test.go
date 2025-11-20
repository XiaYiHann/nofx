package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPasswordHashing(t *testing.T) {
	password := "mysecretpassword"
	
	// Test HashPassword
	hash, err := HashPassword(password)
	assert.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Test CheckPassword with correct password
	match := CheckPassword(password, hash)
	assert.True(t, match, "Password should match hash")

	// Test CheckPassword with incorrect password
	match = CheckPassword("wrongpassword", hash)
	assert.False(t, match, "Wrong password should not match hash")
}

func TestTokenBlacklist(t *testing.T) {
	token := "test-token-123"
	
	// Initially not blacklisted
	assert.False(t, IsTokenBlacklisted(token))

	// Blacklist with future expiration
	futureExp := time.Now().Add(1 * time.Hour)
	BlacklistToken(token, futureExp)
	assert.True(t, IsTokenBlacklisted(token), "Token should be blacklisted")

	// Blacklist with past expiration (should be cleaned up on check)
	expiredToken := "expired-token-456"
	pastExp := time.Now().Add(-1 * time.Hour)
	BlacklistToken(expiredToken, pastExp)
	
	// Should return false because it's expired (and be removed)
	assert.False(t, IsTokenBlacklisted(expiredToken), "Expired token should not be considered blacklisted")
}

func TestGenerateOTPSecret(t *testing.T) {
	secret, err := GenerateOTPSecret()
	assert.NoError(t, err)
	assert.NotEmpty(t, secret)
	// TOTP secrets are usually base32 encoded strings, check length or content if needed
	// For now just ensuring it's not empty is a good start
}

func TestJWTSecret(t *testing.T) {
	secret := "super-secret-key"
	SetJWTSecret(secret)
	assert.Equal(t, []byte(secret), JWTSecret)
}
