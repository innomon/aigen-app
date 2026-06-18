package framework

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"

	"github.com/innomon/aigen-app/utils/logger"

	"github.com/innomon/aigen-app/core/agentic/agents"
	"github.com/innomon/aigen-app/core/api"
	"github.com/innomon/aigen-app/core/bizdefs"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/app_extensions"
	"github.com/innomon/aigen-app/core/services"
	"github.com/innomon/aigen-app/infrastructure/filestore"
	"github.com/innomon/aigen-app/infrastructure/relationdbdao"
	chi "github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/crypto/acme/autocert"
)

func isExternalDomain(domain string) bool {
	if domain == "" || domain == "localhost" {
		return false
	}
	// Check if it's an IP address
	if net.ParseIP(domain) != nil {
		return false
	}
	return true
}

// App represents the initialized AiGen CMS application.
type App struct {
	Router             chi.Router
	Config             *Config
	DAO                relationdbdao.IPrimaryDao
	EntityService      services.IEntityService
	SchemaService      services.ISchemaService
	AuthService        services.IAuthService
	InteractionService services.IInteractionService
	PageService        *services.PageService
	PermissionService  *services.PermissionService
	A2UIService        services.IA2UIService
	ExtensionService   *app_extensions.AppExtensionService
}

// NewApp initializes all services and the router, but does not start the server.
func NewApp(cfg *Config) (*App, error) {
	// Initialize logger
	_, err := logger.Init(logger.Config{
		Level:          cfg.Log.Level,
		ConsoleEnabled: cfg.Log.ConsoleEnabled,
		FileEnabled:    cfg.Log.FileEnabled,
		Dir:            cfg.Log.Dir,
		FileName:       cfg.Log.FileName,
		MaxSizeMB:      cfg.Log.MaxSizeMB,
		MaxBackups:     cfg.Log.MaxBackups,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Initialize Database
	dao, err := relationdbdao.CreateDao(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create dao: %w", err)
	}

	// Ensure core records table exists
	if err := dao.EnsureTable(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ensure records table: %w", err)
	}

	// Initialize Services
	systemSettings := descriptors.DefaultSystemSettings()

	fsCfg := filestore.Config{
		Driver: cfg.Storage.Driver,
	}
	fsCfg.FS.PathPrefix = cfg.Storage.FS.Root
	fsCfg.FS.UrlPrefix = cfg.Storage.FS.UrlPrefix
	fsCfg.S3.Bucket = cfg.Storage.S3.Bucket
	fsCfg.S3.Region = cfg.Storage.S3.Region
	fsCfg.S3.AccessKeyID = cfg.Storage.S3.AccessKeyID
	fsCfg.S3.SecretAccessKey = cfg.Storage.S3.SecretAccessKey
	fsCfg.S3.Endpoint = cfg.Storage.S3.Endpoint
	fsCfg.GCS.Bucket = cfg.Storage.GCS.Bucket
	fsCfg.GCS.CredentialsFile = cfg.Storage.GCS.CredentialsFile
	fsCfg.Postgres.URL = cfg.Storage.Postgres.URL

	// If root is default and WWWRoot was overridden, update it
	if cfg.WWWRoot != "wwwroot" && cfg.Storage.FS.Root == "wwwroot/files" {
		fsCfg.FS.PathPrefix = filepath.Join(cfg.WWWRoot, "files")
	}

	fileStore, err := filestore.CreateFileStore(context.Background(), fsCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create file store: %w", err)
	}
	systemSettings.LocalFileStoreOptions.PathPrefix = fsCfg.FS.PathPrefix
	systemSettings.LocalFileStoreOptions.UrlPrefix = fsCfg.FS.UrlPrefix

	schemaService := services.NewSchemaService(dao)
	evolutionService := services.NewEvolutionService(dao, schemaService)
	permissionService := services.NewPermissionService(dao, schemaService)
	schemaService.SetPermissionService(permissionService)

	// Create privileged seed context with 'sa' role to bypass permissions filtering during startup/seeding
	seedCtx := context.WithValue(context.Background(), "roles", []string{"sa"})
	seedCtx = context.WithValue(seedCtx, "userId", int64(1))

	enabledBizDefs, err := bizdefs.LoadBizDefsConfig(cfg.BizDefsDir)
	if err != nil {
		logger.Printf("Warning: failed to load bizdefs config from %s: %v", cfg.BizDefsDir, err)
		enabledBizDefs = []string{} // Proceed without bizdefs if failed
	}

	for _, bizdefName := range enabledBizDefs {
		logger.Printf("Setting up bizdef schemas: %s", bizdefName)
		if err := bizdefs.SetupBizDef(seedCtx, cfg.BizDefsDir, bizdefName, schemaService, dao); err != nil {
			logger.Printf("Warning: failed to setup bizdef %s schemas: %v\n", bizdefName, err)
		}
		if err := bizdefs.SetupBizDefEvolution(seedCtx, cfg.BizDefsDir, bizdefName, evolutionService); err != nil {
			logger.Printf("Warning: failed to setup bizdef %s evolution manifests: %v\n", bizdefName, err)
		}
	}

	entityService := services.NewEntityService(schemaService, evolutionService, dao, permissionService)

	for _, bizdefName := range enabledBizDefs {
		logger.Printf("Setting up test data for bizdef: %s", bizdefName)
		if err := bizdefs.SetupBizDefTestData(seedCtx, cfg.BizDefsDir, bizdefName, entityService); err != nil {
			logger.Printf("Warning: failed to setup test data for bizdef %s: %v\n", bizdefName, err)
		}
	}

	graphqlService := services.NewGraphQLService(schemaService, entityService)
	assetService := services.NewAssetService(dao, fileStore, systemSettings)
	engagementService := services.NewEngagementService(dao)
	commentService := services.NewCommentService(dao)
	notificationService := services.NewNotificationService(dao)
	interactionService := services.NewInteractionService(dao)
	whatsappService, err := services.NewWhatsAppService(cfg.Channels.WhatsApp)
	if err != nil {
		logger.Printf("Warning: failed to initialize WhatsApp service: %v", err)
	}
	channelService := services.NewChannelService(dao, cfg.Channels, interactionService, assetService)
	authService := services.NewAuthService(dao, "your-secret-key", channelService, whatsappService)

	// Bootstrap administrator account
	isTestEnv := cfg.DatabaseDSN == "memory://" || os.Getenv("FORMCMS_ENV") == "test"
	if err := authService.BootstrapAdmin(seedCtx, cfg.Admin.Email, cfg.Admin.Password, isTestEnv); err != nil {
		logger.Printf("Warning: failed to bootstrap admin user: %v", err)
	}

	// Bootstrap default premium home page
	if err := schemaService.BootstrapDefaultHomePage(seedCtx); err != nil {
		logger.Printf("Warning: failed to bootstrap default home page: %v", err)
	}

	auditService := services.NewAuditService(dao)
	pageService := services.NewPageService(schemaService, graphqlService)
	a2uiService := services.NewA2UIService()
	tempAccessService := services.NewTempAccessService(cfg.TemporaryAccess, fileStore)
	commerceService := services.NewCommerceService(entityService)

	chatService, err := services.NewChatService(cfg.AgenticConfigPath, entityService, schemaService, evolutionService, a2uiService, interactionService, commerceService)
	if err != nil {
		logger.Printf("Warning: failed to initialize chat service (agentic config missing or invalid): %v", err)
	}

	extensionService := app_extensions.NewAppExtensionService(cfg.AppExtensionsDir, schemaService, evolutionService, chatService, entityService, a2uiService, auditService)
	if err := extensionService.Start(context.Background()); err != nil {
		logger.Printf("Warning: failed to start app extension service: %v", err)
	}

	// Register Router Agent globally with App Extension provider and Permissions provider
	agents.RegisterRouterAgent(interactionService, extensionService, permissionService)

	a2aService := services.NewA2AService(chatService, cfg.Domain)
	mcpService := services.NewMCPService(schemaService, entityService, authService, cfg.MCP)

	// Initialize APIs
	authApi := api.NewAuthApi(authService, permissionService, whatsappService)
	rbacApi := api.NewRBACApi(entityService, authApi)
	schemaApi := api.NewSchemaApi(schemaService, authApi)
	entityApi := api.NewEntityApi(entityService, authApi)
	graphqlApi := api.NewGraphQLApi(graphqlService, authApi)
	queryApi := api.NewQueryApi(graphqlService, authApi)
	assetApi := api.NewAssetApi(assetService)
	engagementApi := api.NewEngagementApi(engagementService, authApi)
	commentApi := api.NewCommentApi(commentService, authApi)
	notificationApi := api.NewNotificationApi(notificationService, authApi)
	auditApi := api.NewAuditApi(auditService, authApi)
	channelApi := api.NewChannelApi(channelService, authApi)
	commerceApi := api.NewCommerceApi(commerceService, authApi)
	a2aApi := api.NewA2AApi(a2aService, authService, cfg.Channels)
	mcpApi := api.NewMCPApi(mcpService, authApi, tempAccessService, fileStore)
	tempAccessApi := api.NewTempAccessApi(cfg.TemporaryAccess, tempAccessService, fileStore)
	staticApi := api.NewStaticApi(cfg.WWWRoot, cfg.CustomUIPath, cfg.Storage.FS.UrlPrefix, extensionService)
	pageApi := api.NewPageApi(pageService, authService, authApi)
	a2uiApi := api.NewA2UIApi(a2uiService, authApi)
	extensionApi := api.NewAppExtensionApi(extensionService, authApi)
	var chatApi *api.ChatApi
	var adk2appApi *api.ADK2AppApi
	if chatService != nil {
		chatApi = api.NewChatApi(chatService, authApi)
		adk2appApi, err = api.NewADK2AppApi(chatService, authService, permissionService, whatsappService, extensionService)
		if err != nil {
			logger.Printf("Warning: failed to initialize adk2app API: %v", err)
		}
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	// Register APIs
	schemaApi.Register(r)
	entityApi.Register(r)
	graphqlApi.Register(r)
	queryApi.Register(r)
	assetApi.Register(r)
	engagementApi.Register(r)
	commentApi.Register(r)
	authApi.Register(r)
	rbacApi.Register(r)
	notificationApi.Register(r)
	auditApi.Register(r)
	commerceApi.Register(r)
	channelApi.Register(r)
	a2aApi.Register(r)
	mcpApi.Register(r)
	tempAccessApi.Register(r)
	staticApi.Register(r)
	pageApi.Register(r)
	a2uiApi.Register(r)
	extensionApi.Register(r)
	if chatApi != nil {
		chatApi.Register(r)
	}
	if adk2appApi != nil {
		adk2appApi.Register(r)
	}

	return &App{
		Router:             r,
		Config:             cfg,
		DAO:                dao,
		EntityService:      entityService,
		SchemaService:      schemaService,
		AuthService:        authService,
		InteractionService: interactionService,
		PageService:        pageService,
		PermissionService:  permissionService,
		A2UIService:        a2uiService,
		ExtensionService:   extensionService,
	}, nil
}

// Run starts the HTTP/HTTPS server.
func (a *App) Run() error {
	if isExternalDomain(a.Config.Domain) {
		certManager := autocert.Manager{
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(a.Config.Domain),
			Cache:      autocert.DirCache("certs"),
		}

		server := &http.Server{
			Addr:    ":443",
			Handler: a.Router,
			TLSConfig: &tls.Config{
				GetCertificate: certManager.GetCertificate,
			},
		}

		fmt.Printf("Starting AiGen CMS on %s with autocert...\n", a.Config.Domain)
		// Redirect HTTP to HTTPS
		go http.ListenAndServe(":80", certManager.HTTPHandler(nil))
		return server.ListenAndServeTLS("", "")
	} else {
		fmt.Printf("Starting AiGen CMS on :%s...\n", a.Config.Port)
		return http.ListenAndServe(":"+a.Config.Port, a.Router)
	}
}

// Start is a convenience wrapper that initializes and runs the app.
func Start(cfg *Config) error {
	app, err := NewApp(cfg)
	if err != nil {
		return err
	}
	defer app.DAO.Close()
	return app.Run()
}
