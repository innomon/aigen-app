package plugins

import (
	"context"
	"fmt"

	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/utils/datamodels"
)

// AIGenHostAPI defines the methods exposed to sandboxed scripts.
type AIGenHostAPI struct {
	EntityService services.IEntityService
	A2UIService   services.IA2UIService
}

func (api *AIGenHostAPI) GetEntity(ctx context.Context, entity string, id any) (map[string]any, error) {
	return api.EntityService.Single(ctx, entity, id)
}

func (api *AIGenHostAPI) ListEntities(ctx context.Context, entity string, limit string) ([]any, error) {
	records, _, err := api.EntityService.List(ctx, entity, datamodels.Pagination{Limit: &limit}, nil, nil)
	if err != nil {
		return nil, err
	}
	res := make([]any, len(records))
	for i, r := range records {
		res[i] = r
	}
	return res, nil
}

func (api *AIGenHostAPI) UpdateUI(ctx context.Context, id string, typ string, attributes map[string]any) error {
	comp := services.A2UIComponent{
		ID:         id,
		Type:       typ,
		Attributes: attributes,
	}
	api.A2UIService.UpdateComponent(ctx, comp)
	return nil
}

// Log allows plugins to write to the host's audit log.
func (api *AIGenHostAPI) Log(ctx context.Context, level string, message string) {
	// TODO: Integrate with AuditService
	fmt.Printf("[%s] Plugin Log: %s\n", level, message)
}
