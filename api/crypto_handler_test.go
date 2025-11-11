package api

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"nofx/crypto"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupTestServer configures a test server and CryptoHandler
func setupTestServer() (*gin.Engine, *crypto.CryptoService, error) {
	gin.SetMode(gin.TestMode)
	router := gin.Default()

	// Generate a temporary RSA key for testing
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	// Create a temporary CryptoService
	cryptoService := &crypto.CryptoService{}
	cryptoService.SetPrivateKey(privateKey)

	// Set a dummy data encryption key for tests that require it
	os.Setenv("DATA_ENCRYPTION_KEY", "test-data-encryption-key")

	handler := NewCryptoHandler(cryptoService)
	router.GET("/publicKey", handler.HandleGetPublicKey)
	router.POST("/decrypt", handler.HandleDecryptSensitiveData)

	return router, cryptoService, nil
}

func TestHandleGetPublicKey(t *testing.T) {
	router, _, err := setupTestServer()
	assert.NoError(t, err)

	req, _ := http.NewRequest(http.MethodGet, "/publicKey", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Contains(t, response, "public_key")
	assert.Contains(t, response, "algorithm")
	assert.Equal(t, "RSA-OAEP-2048", response["algorithm"])
}

func TestHandleDecryptSensitiveData(t *testing.T) {
	router, cryptoService, err := setupTestServer()
	assert.NoError(t, err)

	// 1. Success case
	t.Run("Success", func(t *testing.T) {
		plaintext := "this is a secret message"

		// Create a valid encrypted payload
		payload, err := createTestPayload(cryptoService, plaintext)
		assert.NoError(t, err)

		payloadBytes, err := json.Marshal(payload)
		assert.NoError(t, err)

		req, _ := http.NewRequest(http.MethodPost, "/decrypt", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var response map[string]string
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, plaintext, response["plaintext"])
	})

	// 2. Failure case: Invalid JSON
	t.Run("InvalidJSON", func(t *testing.T) {
		req, _ := http.NewRequest(http.MethodPost, "/decrypt", bytes.NewBufferString("{invalid json"))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	// 3. Failure case: Decryption failed
	t.Run("DecryptionFailed", func(t *testing.T) {
		// Create a payload with a tampered ciphertext
		payload, err := createTestPayload(cryptoService, "another message")
		assert.NoError(t, err)
		payload.Ciphertext = "tampered"

		payloadBytes, err := json.Marshal(payload)
		assert.NoError(t, err)

		req, _ := http.NewRequest(http.MethodPost, "/decrypt", bytes.NewBuffer(payloadBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})
}

// createTestPayload is a helper function to encrypt data for testing
func createTestPayload(cs *crypto.CryptoService, plaintext string) (*crypto.EncryptedPayload, error) {
	// This is a simplified version of the client-side encryption for testing purposes
	aesKey := make([]byte, 32) // 256-bit AES key
	_, err := rand.Read(aesKey)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	_, err = rand.Read(nonce)
	if err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)

	wrappedKey, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, cs.GetPublicKey(), aesKey, nil)
	if err != nil {
		return nil, err
	}

	return &crypto.EncryptedPayload{
		WrappedKey: base64.RawURLEncoding.EncodeToString(wrappedKey),
		IV:         base64.RawURLEncoding.EncodeToString(nonce),
		Ciphertext: base64.RawURLEncoding.EncodeToString(ciphertext),
	}, nil
}
