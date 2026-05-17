package services

import (
	"context"
	"testing"

	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	"github.com/stretchr/testify/assert"
)

func TestInteractionService(t *testing.T) {
	dao, _ := relationdbdao.CreateDao("memory://")
	dao.EnsureTable(context.Background())
	svc := NewInteractionService(dao)
	ctx := context.Background()

	t.Run("Log and GetHistory", func(t *testing.T) {
		identifier := "test-user-1"
		
		// Log inbound
		i1 := &descriptors.Interaction{
			Identifier:  identifier,
			Direction:   "inbound",
			ContentType: "text",
			Content:     "Hello Agent",
			Status:      "processed",
		}
		err := svc.Log(ctx, i1)
		assert.NoError(t, err)
		assert.NotEmpty(t, i1.Id)

		// Log outbound
		i2 := &descriptors.Interaction{
			Identifier:  identifier,
			Direction:   "outbound",
			ContentType: "text",
			Content:     "Hello User",
			Status:      "delivered",
		}
		err = svc.Log(ctx, i2)
		assert.NoError(t, err)

		// Get History
		history, err := svc.GetHistory(ctx, identifier, 10)
		assert.NoError(t, err)
		assert.Len(t, history, 2)
		assert.Equal(t, "Hello User", history[0].Content) // Descending order
		assert.Equal(t, "Hello Agent", history[1].Content)
	})

	t.Run("UpdateStatus", func(t *testing.T) {
		i := &descriptors.Interaction{
			Identifier: "test-user-2",
			Direction:  "outbound",
			Status:     "pending",
		}
		svc.Log(ctx, i)

		err := svc.UpdateStatus(ctx, i.Id, "failed", "gateway error")
		assert.NoError(t, err)

		history, _ := svc.GetHistory(ctx, "test-user-2", 1)
		assert.Equal(t, "failed", history[0].Status)
		assert.Equal(t, "gateway error", history[0].Metadata["error"])
	})

	t.Run("GetPendingOutbound", func(t *testing.T) {
		channel := descriptors.ChannelWhatsApp
		i := &descriptors.Interaction{
			ChannelType: channel,
			Identifier:  "12345",
			Direction:   "outbound",
			Status:      "pending",
		}
		svc.Log(ctx, i)

		pending, err := svc.GetPendingOutbound(ctx, channel)
		assert.NoError(t, err)
		assert.Len(t, pending, 1)
		assert.Equal(t, "12345", pending[0].Identifier)
	})
}
