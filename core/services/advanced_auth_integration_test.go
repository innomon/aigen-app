package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

func TestWhatsAppIntegration(t *testing.T) {
	ctx := context.Background()
	dao, _ := relationdbdao.CreateDao("memory://")
	dao.EnsureTable(ctx)
	
	intSvc := NewInteractionService(dao)
	authSvc := NewAuthService(dao, "secret", nil, nil)
	
	// Mock WhatsApp Gateway
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := descriptors.ChannelConfig{
		Enabled:    true,
		GatewayURL: server.URL,
	}
	s, _ := NewWhatsAppService(cfg)
	
	// Need a full ChannelService to support LoginByChannel properly as it is used by AuthService
	// But let's test AuthService.LoginByChannel directly as it calls whatsapp service for TOTP if needed
	chanSvc := NewChannelService(dao, descriptors.ChannelsConfig{}, intSvc, nil)
	authSvc.channelService = chanSvc
	authSvc.whatsappService = s

	t.Run("Login by WhatsApp (New User)", func(t *testing.T) {
		identifier := "+19998887777"
		token, err := authSvc.LoginByChannel(ctx, descriptors.ChannelWhatsApp, identifier, "", "127.0.0.1", "test-agent")
		assert.NoError(t, err)
		assert.NotEmpty(t, token)

		// Verify user created
		userId, _, err := authSvc.ValidateToken(token)
		assert.NoError(t, err)
		user, err := authSvc.Me(ctx, userId)
		assert.NoError(t, err)
		assert.Equal(t, identifier, user.Phone)
	})
}
