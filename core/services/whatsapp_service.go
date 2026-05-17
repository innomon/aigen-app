package services

import (
	"context"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/innomon/aigen-app/core/descriptors"
)

type WhatsAppService struct {
	config     descriptors.ChannelConfig
	privateKey *rsa.PrivateKey
	gatewayPub *rsa.PublicKey
	otps       map[string]otpEntry
	mu         sync.RWMutex
}

type otpEntry struct {
	otp       string
	expiresAt time.Time
}

func NewWhatsAppService(config descriptors.ChannelConfig) (*WhatsAppService, error) {
	s := &WhatsAppService{
		config: config,
		otps:   make(map[string]otpEntry),
	}

	if config.PrivateKey != "" {
		block, _ := pem.Decode([]byte(config.PrivateKey))
		if block == nil {
			return nil, fmt.Errorf("failed to parse private key PEM")
		}
		priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			// Try PKCS8 if PKCS1 fails
			p8, err := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %v", err)
			}
			s.privateKey = p8.(*rsa.PrivateKey)
		} else {
			s.privateKey = priv
		}
	}

	if config.PublicKey != "" {
		block, _ := pem.Decode([]byte(config.PublicKey))
		if block == nil {
			return nil, fmt.Errorf("failed to parse gateway public key PEM")
		}
		pub, err := x509.ParsePKIXPublicKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse gateway public key: %v", err)
		}
		s.gatewayPub = pub.(*rsa.PublicKey)
	}

	return s, nil
}

func (s *WhatsAppService) GenerateReverseOTPJWT(mobile string, challengeID string) (string, error) {
	if s.privateKey == nil {
		return "", fmt.Errorf("private key not configured")
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.MapClaims{
		"mobile":       mobile,
		"app_name":     "AIGenApp",
		"challenge_id": challengeID,
		"iat":          time.Now().Unix(),
		"exp":          time.Now().Add(5 * time.Minute).Unix(),
	})

	return token.SignedString(s.privateKey)
}

func (s *WhatsAppService) VerifyGatewayJWT(tokenString string) (string, string, error) {
	if s.gatewayPub == nil {
		return "", "", fmt.Errorf("gateway public key not configured")
	}

	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.gatewayPub, nil
	})

	if err != nil {
		return "", "", err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		mobile := claims["mobile"].(string)
		challengeID := claims["challenge_id"].(string)
		return mobile, challengeID, nil
	}

	return "", "", fmt.Errorf("invalid token")
}

func (s *WhatsAppService) GenerateOTP(challengeID string) (string, error) {
	otp := fmt.Sprintf("%06d", rand.Intn(1000000))
	
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.otps[challengeID] = otpEntry{
		otp:       otp,
		expiresAt: time.Now().Add(10 * time.Minute),
	}
	
	return otp, nil
}

func (s *WhatsAppService) VerifyOTP(challengeID string, otp string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	entry, ok := s.otps[challengeID]
	if !ok {
		return false, nil
	}
	
	if time.Now().After(entry.expiresAt) {
		delete(s.otps, challengeID)
		return false, nil
	}
	
	if entry.otp == otp {
		delete(s.otps, challengeID)
		return true, nil
	}
	
	return false, nil
}

func (s *WhatsAppService) GenerateTOTPCode(secret []byte, pubKey []byte, userID, appID string) uint32 {
	return s.generateTOTPAtTime(secret, pubKey, userID, appID, uint64(time.Now().Unix()))
}

func (s *WhatsAppService) generateTOTPAtTime(secret []byte, pubKey []byte, userID, appID string, timestamp uint64) uint32 {
	stepSec := uint64(30)
	counter := timestamp / stepSec
	
	var buf [32 + 128 + 128 + 8]byte
	n := copy(buf[:], pubKey)
	n += copy(buf[n:], userID)
	n += copy(buf[n:], appID)
	binary.BigEndian.PutUint64(buf[n:], counter)
	
	h := hmac.New(sha256.New, secret)
	h.Write(buf[:n+8])
	hash := h.Sum(nil)

	offset := int(hash[31] & 0x0f)
	binaryValue := binary.BigEndian.Uint32(hash[offset : offset+4]) & 0x7fffffff
	return binaryValue % 1_000_000
}

func (s *WhatsAppService) VerifyTOTPCode(secret []byte, pubKey []byte, userID, appID string, code uint32) bool {
	now := uint64(time.Now().Unix())
	// Check window of ±1
	for i := -1; i <= 1; i++ {
		ts := now + uint64(i*30)
		if s.generateTOTPAtTime(secret, pubKey, userID, appID, ts) == code {
			return true
		}
	}
	return false
}

func (s *WhatsAppService) EnrollTOTP(ctx context.Context, userId int64, pubKey []byte) ([]byte, error) {
	secret := make([]byte, 32)
	_, err := rand.Read(secret)
	if err != nil {
		return nil, err
	}
	return secret, nil
}
