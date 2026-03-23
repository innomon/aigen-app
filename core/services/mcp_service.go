package services

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/utils/datamodels"
)

type MCPService struct {
	server        *mcp.Server
	schemaService ISchemaService
	entityService IEntityService
	authService   IAuthService
	config        descriptors.MCPConfig
}

func NewMCPService(schemaService ISchemaService, entityService IEntityService, authService IAuthService, config descriptors.MCPConfig) *MCPService {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "aigen-cms-mcp",
		Version: "1.0.0",
	}, nil)

	s := &MCPService{
		server:        server,
		schemaService: schemaService,
		entityService: entityService,
		authService:   authService,
		config:        config,
	}

	s.registerTools()
	return s
}

func (s *MCPService) GetServer() *mcp.Server {
	return s.server
}

func (s *MCPService) registerTools() {
	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "list_entities",
		Description: "List all available CMS entities",
	}, s.listEntitiesHandler)

	mcp.AddTool(s.server, &mcp.Tool{
		Name:        "get_entity_records",
		Description: "Get records for a specific entity",
	}, s.getEntityRecordsHandler)
}

func (s *MCPService) listEntitiesHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	schemas, err := s.schemaService.All(ctx, nil, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	var entities []string
	for _, sc := range schemas {
		if sc.Type == descriptors.EntitySchema {
			entities = append(entities, sc.Name)
		}
	}

	resJson, _ := json.Marshal(entities)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resJson)},
		},
	}, nil, nil
}

func (s *MCPService) getEntityRecordsHandler(ctx context.Context, req *mcp.CallToolRequest, _ any) (*mcp.CallToolResult, any, error) {
	// Extract arguments from req.Arguments
	var args struct {
		Name string `json:"name"`
	}
	if err := json.Unmarshal(req.Params.Arguments, &args); err != nil {
		return nil, nil, err
	}

	records, _, err := s.entityService.List(ctx, args.Name, datamodels.Pagination{}, nil, nil)
	if err != nil {
		return nil, nil, err
	}

	resJson, _ := json.Marshal(records)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: string(resJson)},
		},
	}, nil, nil
}

func (s *MCPService) Authenticate(ctx context.Context, apiKey string) (int64, []string, error) {
	for _, ak := range s.config.APIKeys {
		if ak.Key == apiKey {
			// Find user to get roles
			user, err := s.authService.Me(ctx, ak.UserId)
			if err != nil {
				return 0, nil, err
			}
			
			// Verify MCP role
			hasMcpRole := false
			for _, r := range user.Roles {
				if r == "MCP" {
					hasMcpRole = true
					break
				}
			}
			
			if !hasMcpRole {
				return 0, nil, fmt.Errorf("user does not have MCP role")
			}

			return user.Id, user.Roles, nil
		}
	}
	return 0, nil, fmt.Errorf("invalid MCP API Key")
}
