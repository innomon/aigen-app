package framework

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/http"
	"path/filepath"

	"github.com/innomon/aigen-cms/core/api"
	"github.com/innomon/aigen-cms/core/apps"
	"github.com/innomon/aigen-cms/core/descriptors"
	"github.com/innomon/aigen-cms/core/services"
	"github.com/innomon/aigen-cms/infrastructure/filestore"
	"github.com/innomon/aigen-cms/infrastructure/relationdbdao"
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
	Router            chi.Router
	Config            *Config
	DAO               relationdbdao.IPrimaryDao
	EntityService     services.IEntityService
	SchemaService     services.ISchemaService
	AuthService       services.IAuthService
	PageService       *services.PageService
	PermissionService *services.PermissionService
	A2UIService       *services.A2UIService
}

// NewApp initializes all services and the router, but does not start the server.
func NewApp(cfg *Config) (*App, error) {
	// Initialize Database
	dao, err := relationdbdao.CreateDao(descriptors.DatabaseProvider(cfg.DatabaseType), cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to create dao: %w", err)
	}

	// Ensure core tables exist
	_, err = dao.GetDb().ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS __schemas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			schema_id TEXT,
			name TEXT,
			type TEXT,
			settings TEXT,
			description TEXT,
			is_latest BOOLEAN,
			publication_status TEXT,
			created_at DATETIME,
			created_by TEXT,
			deleted BOOLEAN
		);
		CREATE TABLE IF NOT EXISTS __users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			role TEXT NOT NULL,
			avatar_path TEXT,
			created_at DATETIME,
			updated_at DATETIME
		);
		CREATE TABLE IF NOT EXISTS __user_channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			agent_id TEXT,
			channel_type TEXT NOT NULL,
			identifier TEXT NOT NULL,
			is_authenticated BOOLEAN DEFAULT 0,
			metadata TEXT,
			created_at DATETIME,
			updated_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES __users(id)
		);
		CREATE TABLE IF NOT EXISTS __auth_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			channel_type TEXT NOT NULL,
			action TEXT NOT NULL,
			ip_address TEXT,
			user_agent TEXT,
			success BOOLEAN,
			metadata TEXT,
			created_at DATETIME,
			FOREIGN KEY (user_id) REFERENCES __users(id)
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create core tables: %w", err)
	}

	// Initialize Services
	systemSettings := descriptors.DefaultSystemSettings()
	systemSettings.LocalFileStoreOptions.PathPrefix = filepath.Join(cfg.WWWRoot, "files")
	fileStore := filestore.NewLocalFileStore(systemSettings.LocalFileStoreOptions.PathPrefix, systemSettings.LocalFileStoreOptions.UrlPrefix)

	schemaService := services.NewSchemaService(dao)
	permissionService := services.NewPermissionService(dao, schemaService)

	enabledApps, err := apps.LoadAppsConfig(cfg.AppsDir)
	if err != nil {
		log.Printf("Warning: failed to load apps config from %s: %v", cfg.AppsDir, err)
		enabledApps = []string{} // Proceed without apps if failed
	}

	for _, appName := range enabledApps {
		log.Printf("Setting up app schemas: %s", appName)
		if err := apps.SetupApp(context.Background(), cfg.AppsDir, appName, schemaService, dao); err != nil {
			log.Printf("Warning: failed to setup app %s schemas: %v\n", appName, err)
		}
	}

	entityService := services.NewEntityService(schemaService, dao, permissionService)

	for _, appName := range enabledApps {
		log.Printf("Setting up test data for app: %s", appName)
		if err := apps.SetupAppTestData(context.Background(), cfg.AppsDir, appName, entityService); err != nil {
			log.Printf("Warning: failed to setup test data for app %s: %v\n", appName, err)
		}
	}

	graphqlService := services.NewGraphQLService(schemaService, entityService)
	assetService := services.NewAssetService(dao, fileStore, systemSettings)
	engagementService := services.NewEngagementService(dao)
	commentService := services.NewCommentService(dao)
	notificationService := services.NewNotificationService(dao)
	channelService := services.NewChannelService(dao, cfg.Channels)
	authService := services.NewAuthService(dao, "your-secret-key", channelService)
	auditService := services.NewAuditService(dao)
	pageService := services.NewPageService(schemaService, graphqlService)
	a2uiService := services.NewA2UIService()

	chatService, err := services.NewChatService(cfg.AgenticConfigPath, entityService, schemaService, a2uiService)
	if err != nil {
		log.Printf("Warning: failed to initialize chat service (agentic config missing or invalid): %v", err)
	}

	a2aService := services.NewA2AService(chatService, cfg.Domain)
	mcpService := services.NewMCPService(schemaService, entityService, authService, cfg.MCP)

	// Initialize APIs
	authApi := api.NewAuthApi(authService, permissionService)
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
	a2aApi := api.NewA2AApi(a2aService, authService, cfg.Channels)
	mcpApi := api.NewMCPApi(mcpService)
	staticApi := api.NewStaticApi()
	pageApi := api.NewPageApi(pageService, authService, authApi)
	a2uiApi := api.NewA2UIApi(a2uiService, authApi)
	var chatApi *api.ChatApi
	if chatService != nil {
		chatApi = api.NewChatApi(chatService, authApi)
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
	channelApi.Register(r)
	a2aApi.Register(r)
	mcpApi.Register(r)
	staticApi.Register(r)
	pageApi.Register(r)
	a2uiApi.Register(r)
	if chatApi != nil {
		chatApi.Register(r)
	}

	return &App{
		Router:            r,
		Config:            cfg,
		DAO:               dao,
		EntityService:     entityService,
		SchemaService:     schemaService,
		AuthService:       authService,
		PageService:       pageService,
		PermissionService: permissionService,
		A2UIService:       a2uiService,
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
