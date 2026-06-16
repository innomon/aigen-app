package services

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/stretchr/testify/assert"
)

func generateKeyPair() (string, string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", err
	}

	privBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}
	privPEM := pem.EncodeToMemory(privBlock)

	pubBytes, err := x509.MarshalPKIXPublicKey(&key.PublicKey)
	if err != nil {
		return "", "", err
	}
	pubBlock := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	}
	pubPEM := pem.EncodeToMemory(pubBlock)

	return string(privPEM), string(pubPEM), nil
}

func TestWhatsAppService_ReverseOTP(t *testing.T) {
	priv, pub, _ := generateKeyPair()
	cfg := descriptors.ChannelConfig{
		PrivateKey: priv,
		PublicKey:  pub, // Using same key for gateway mock simplicity
	}

	s, err := NewWhatsAppService(cfg)
	assert.NoError(t, err)

	mobile := "+919876543210"
	challengeID := "test-challenge"

	token, err := s.GenerateReverseOTPJWT(mobile, challengeID)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	m, c, err := s.VerifyGatewayJWT(token)
	assert.NoError(t, err)
	assert.Equal(t, mobile, m)
	assert.Equal(t, challengeID, c)

	otp, err := s.GenerateOTP(challengeID)
	assert.NoError(t, err)
	assert.Len(t, otp, 6)

	valid, err := s.VerifyOTP(challengeID, otp)
	assert.NoError(t, err)
	assert.True(t, valid)

	valid, err = s.VerifyOTP(challengeID, "wrong")
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestWhatsAppService_TOTP(t *testing.T) {
	s, _ := NewWhatsAppService(descriptors.ChannelConfig{})

	secret := make([]byte, 32)
	rand.Read(secret)
	pubKey := make([]byte, 32)
	rand.Read(pubKey)
	userID := "123456"
	appID := "AIGenApp"

	code := s.GenerateTOTPCode(secret, pubKey, userID, appID)
	assert.True(t, s.VerifyTOTPCode(secret, pubKey, userID, appID, code))

	// Verify it fails with wrong code
	assert.False(t, s.VerifyTOTPCode(secret, pubKey, userID, appID, code+1))

	// Verify window ±1 (simulated by checking at different times if we exposed the internal method, 
	// but we can just trust the internal logic or mock time if needed)
}

func TestWhatsAppService_VerifyADKJWT(t *testing.T) {
	privPEM, pubPEM, err := generateKeyPair()
	assert.NoError(t, err)

	block, _ := pem.Decode([]byte(privPEM))
	privKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	assert.NoError(t, err)

	cfg := descriptors.ChannelConfig{
		PublicKey: pubPEM,
	}
	s, err := NewWhatsAppService(cfg)
	assert.NoError(t, err)

	now := time.Now()
	t.Run("Valid Token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, &ADKClaims{
			UserID:  "919876543210",
			Channel: "whatsapp",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "whatsadk-gateway",
				Audience:  jwt.ClaimStrings{"adk-agent"},
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			},
		})
		tokenString, err := token.SignedString(privKey)
		assert.NoError(t, err)

		claims, err := s.VerifyADKJWT(tokenString)
		assert.NoError(t, err)
		assert.Equal(t, "919876543210", claims.UserID)
		assert.Equal(t, "whatsapp", claims.Channel)
	})

	t.Run("Expired Token", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, &ADKClaims{
			UserID:  "919876543210",
			Channel: "whatsapp",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "whatsadk-gateway",
				Audience:  jwt.ClaimStrings{"adk-agent"},
				IssuedAt:  jwt.NewNumericDate(now.Add(-10 * time.Minute)),
				ExpiresAt: jwt.NewNumericDate(now.Add(-5 * time.Minute)),
			},
		})
		tokenString, err := token.SignedString(privKey)
		assert.NoError(t, err)

		_, err = s.VerifyADKJWT(tokenString)
		assert.Error(t, err)
	})

	t.Run("Missing User ID", func(t *testing.T) {
		token := jwt.NewWithClaims(jwt.SigningMethodRS256, &ADKClaims{
			UserID:  "",
			Channel: "whatsapp",
			RegisteredClaims: jwt.RegisteredClaims{
				Issuer:    "whatsadk-gateway",
				Audience:  jwt.ClaimStrings{"adk-agent"},
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(5 * time.Minute)),
			},
		})
		tokenString, err := token.SignedString(privKey)
		assert.NoError(t, err)

		_, err = s.VerifyADKJWT(tokenString)
		assert.Error(t, err)
	})
}

