package services

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

type mockAssetService struct{}
func (m *mockAssetService) Save(ctx context.Context, asset *descriptors.Asset) (*descriptors.Asset, error) { return asset, nil }
func (m *mockAssetService) Upload(ctx context.Context, path string, reader io.Reader) error { return nil }
func (m *mockAssetService) UpdateAssetsLinks(ctx context.Context, oldAssetIds []int64, newAssetPaths []string, entityName string, recordId int64) error { return nil }
func (m *mockAssetService) GetAssetByPath(ctx context.Context, path string) (*descriptors.Asset, error) { return nil, nil }

func TestInteractionIntegration(t *testing.T) {
	ctx := context.Background()
	dao, _ := relationdbdao.CreateDao("memory://")
	dao.EnsureTable(ctx)
	
	intSvc := NewInteractionService(dao)
	assetSvc := &mockAssetService{}
	
	// Mock Gateway
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := descriptors.ChannelsConfig{
		WhatsApp: descriptors.ChannelConfig{
			Enabled:    true,
			GatewayURL: server.URL,
		},
	}
	
	chanSvc := NewChannelService(dao, cfg, intSvc, assetSvc)

	t.Run("Inbound message logging", func(t *testing.T) {
		identifier := "wa-12345"
		payload := map[string]interface{}{
			"message": "Hello Agent",
		}
		
		err := chanSvc.HandleInbound(ctx, descriptors.ChannelWhatsApp, identifier, payload)
		assert.NoError(t, err)

		history, err := intSvc.GetHistory(ctx, identifier, 1)
		assert.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, "Hello Agent", history[0].Content)
		assert.Equal(t, "inbound", history[0].Direction)
	})

	t.Run("Outbound message logging", func(t *testing.T) {
		identifier := "wa-67890"
		userId := int64(100)
		
		// Setup UserChannel to avoid SendNotification skipping
		chanSvc.RegisterChannel(ctx, userId, descriptors.ChannelWhatsApp, identifier, nil)
		// Mark as authenticated
		chanSvc.VerifyChannel(ctx, userId, descriptors.ChannelWhatsApp, "token")

		err := chanSvc.SendNotification(ctx, userId, "Hello User", []descriptors.ChannelType{descriptors.ChannelWhatsApp})
		assert.NoError(t, err)

		history, err := intSvc.GetHistory(ctx, identifier, 1)
		assert.NoError(t, err)
		assert.Len(t, history, 1)
		assert.Equal(t, "Hello User", history[0].Content)
		assert.Equal(t, "outbound", history[0].Direction)
		assert.Equal(t, "delivered", history[0].Status)
	})
}
