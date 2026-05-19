package services

import (
	"context"
	"io"

	"github.com/a2aproject/a2a-go/v2/a2a"
	"github.com/a2aproject/a2a-go/v2/a2asrv"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/utils/datamodels"
)

type ISchemaService interface {
	All(ctx context.Context, schemaType *descriptors.SchemaType, names []string, status *descriptors.PublicationStatus) ([]*descriptors.Schema, error)
	ById(ctx context.Context, id int64) (*descriptors.Schema, error)
	BySchemaId(ctx context.Context, schemaId string) (*descriptors.Schema, error)
	ByNameOrDefault(ctx context.Context, name string, schemaType descriptors.SchemaType, status *descriptors.PublicationStatus) (*descriptors.Schema, error)
	ByStartsOrDefault(ctx context.Context, name string, schemaType descriptors.SchemaType, status *descriptors.PublicationStatus) (*descriptors.Schema, error)
	LoadEntity(ctx context.Context, name string) (*descriptors.Entity, error)
	LoadLoadedEntity(ctx context.Context, name string) (*descriptors.LoadedEntity, error)
	Save(ctx context.Context, schema *descriptors.Schema, asPublished bool) (*descriptors.Schema, error)
	Delete(ctx context.Context, schemaId string) error
}

type IEntityService interface {
	List(ctx context.Context, name string, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error)
	Single(ctx context.Context, name string, id interface{}) (datamodels.Record, error)
	Insert(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error)
	Update(ctx context.Context, name string, data datamodels.Record) (datamodels.Record, error)
	Delete(ctx context.Context, name string, id interface{}) error

	CollectionList(ctx context.Context, name, id, attr string, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error)
	CollectionInsert(ctx context.Context, name, id, attr string, data datamodels.Record) (datamodels.Record, error)

	JunctionList(ctx context.Context, name, id, attr string, exclude bool, pagination datamodels.Pagination, filters []datamodels.Filter, sorts []datamodels.Sort) ([]datamodels.Record, int64, error)
	JunctionSave(ctx context.Context, name, id, attr string, targetIds []interface{}) error
	JunctionDelete(ctx context.Context, name, id, attr string, targetIds []interface{}) error
}

type IEvolutionService interface {
	RegisterManifest(bizdefName string, manifest descriptors.EvolutionManifest)
	EvolveRecord(entityName string, rec map[string]interface{}, meta *datamodels.MetaData) (map[string]interface{}, bool, error)
	ScrubEntity(ctx context.Context, entityName string, batchSize int) (int, int, error)
}

type IGraphQLService interface {
	Query(ctx context.Context, query string, variables map[string]interface{}) (interface{}, error)
	ExecuteStoredQuery(ctx context.Context, name string, variables map[string]interface{}) (interface{}, error)
}

type IAssetService interface {
	Save(ctx context.Context, asset *descriptors.Asset) (*descriptors.Asset, error)
	Upload(ctx context.Context, path string, reader io.Reader) error
	UpdateAssetsLinks(ctx context.Context, oldAssetIds []int64, newAssetPaths []string, entityName string, recordId int64) error
	GetAssetByPath(ctx context.Context, path string) (*descriptors.Asset, error)
}

type IEngagementService interface {
	Track(ctx context.Context, status *descriptors.EngagementStatus) error
}

type ICommentService interface {
	List(ctx context.Context, entityName string, recordId int64, pagination datamodels.Pagination) ([]*descriptors.Comment, error)
	Single(ctx context.Context, id string) (*descriptors.Comment, error)
	Save(ctx context.Context, comment *descriptors.Comment) (*descriptors.Comment, error)
	Delete(ctx context.Context, userId, id string) error
}

type IAuthService interface {
	Register(ctx context.Context, email, password string) (*descriptors.User, error)
	Login(ctx context.Context, email, password string) (string, error)
	LoginByChannel(ctx context.Context, channelType descriptors.ChannelType, identifier string, token string, ip, ua string) (string, error)
	LinkChannel(ctx context.Context, userId int64, channelType descriptors.ChannelType, identifier string) error
	UpdateUser(ctx context.Context, user *descriptors.User) error
	Me(ctx context.Context, userId int64) (*descriptors.User, error)
	ValidateToken(token string) (int64, []string, error)
	GetRoleByName(ctx context.Context, name string) (*descriptors.Role, error)
}

type INotificationService interface {
	List(ctx context.Context, userId string, pagination datamodels.Pagination) ([]*descriptors.Notification, error)
	Send(ctx context.Context, notification *descriptors.Notification) error
	MarkAsRead(ctx context.Context, userId string, id string) error
	MarkAllAsRead(ctx context.Context, userId string) error
}

type IAuditService interface {
	List(ctx context.Context, pagination datamodels.Pagination) ([]*descriptors.AuditLog, error)
	ById(ctx context.Context, id string) (*descriptors.AuditLog, error)
	Log(ctx context.Context, log *descriptors.AuditLog) error
}

type IPageService interface {
	Render(ctx context.Context, path string, strArgs datamodels.StrArgs) (string, error)
}

type IInteractionService interface {
	Log(ctx context.Context, interaction *descriptors.Interaction) error
	GetHistory(ctx context.Context, identifier string, limit int) ([]*descriptors.Interaction, error)
	UpdateStatus(ctx context.Context, id string, status string, errStr string) error
	GetPendingOutbound(ctx context.Context, channel descriptors.ChannelType) ([]*descriptors.Interaction, error)
}

type IPermissionService interface {
	HasAccess(ctx context.Context, userId int64, roles []string, entityName, action string) (bool, error)
	GetRowFilters(ctx context.Context, userId int64, entityName string) ([]datamodels.Filter, error)
	GetFieldPermissions(ctx context.Context, entityName string, roles []string) (map[string]map[string]bool, error)
}

type IChannelService interface {
	RegisterChannel(ctx context.Context, userId int64, channelType descriptors.ChannelType, identifier string, metadata map[string]interface{}) (*descriptors.UserChannel, error)
	VerifyChannel(ctx context.Context, userId int64, channelType descriptors.ChannelType, token string) (bool, error)
	GetChannelsByUserId(ctx context.Context, userId int64) ([]*descriptors.UserChannel, error)
	GetChannelByIdentifier(ctx context.Context, channelType descriptors.ChannelType, identifier string) (*descriptors.UserChannel, error)
	LogAuthAttempt(ctx context.Context, log *descriptors.AuthLog) error
	GetAuthLogs(ctx context.Context, userId int64, pagination datamodels.Pagination) ([]*descriptors.AuthLog, int64, error)
	SendNotification(ctx context.Context, userId int64, message string, preferredChannels []descriptors.ChannelType) error
	HandleInbound(ctx context.Context, channelType descriptors.ChannelType, identifier string, payload map[string]interface{}) error
}

type IWhatsAppService interface {
	GenerateReverseOTPJWT(mobile string, challengeID string) (string, error)
	VerifyGatewayJWT(tokenString string) (mobile string, challengeID string, err error)
	GenerateOTP(challengeID string) (string, error)
	VerifyOTP(challengeID string, otp string) (bool, error)
	
	GenerateTOTPCode(secret []byte, pubKey []byte, userID, appID string) uint32
	VerifyTOTPCode(secret []byte, pubKey []byte, userID, appID string, code uint32) bool
	EnrollTOTP(ctx context.Context, userId int64, pubKey []byte) (secret []byte, err error)
}

type IA2AService interface {
	GetHandler() a2asrv.RequestHandler
	GetAgentCard() *a2a.AgentCard
}

type IMCPService interface {
	GetServer() *mcp.Server
}

type ITempAccessService interface {
	IsExpired(ctx context.Context, path string, filename string) (bool, error)
	CleanupExpired(ctx context.Context, path string) (int, error)
}

type ICommerceService interface {
	SearchProducts(ctx context.Context, query string) ([]datamodels.Record, error)
	CreateCheckout(ctx context.Context, buyerId string, productIds []string) (datamodels.Record, error)
	VerifyMandate(ctx context.Context, mandateId string) (bool, error)
}
