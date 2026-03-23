# Refactor Plan: AiGen CMS Extensibility

## Step 1: Define `App` Struct (core/framework)
-   Create a new `App` struct in `framework/init.go`.
-   Include:
    -   `Router (chi.Router)`
    -   `Config (*Config)`
    -   `DAO (relationdbdao.IPrimaryDao)`
    -   `EntityService (services.IEntityService)`
    -   `SchemaService (services.ISchemaService)`
    -   `AuthService (services.IAuthService)`
    -   `PageService (*services.PageService)`
    -   `PermissionService (*services.PermissionService)`

## Step 2: Refactor `Start` Logic into `NewApp` (core/framework)
-   Create a new function: `func NewApp(cfg *Config) (*App, error)`.
-   Move all initialization logic from `Start` to `NewApp`:
    -   Database DAO creation and core table setup.
    -   Service initialization (Schema, Permission, Entity, Auth, etc.).
    -   App setup and test data population.
    -   API registration to a `chi.Router`.
-   Ensure all services are properly assigned to the `App` struct fields.
-   Return the `App` instance and any initialization errors.

## Step 3: Implement `App.Run` (core/framework)
-   Create a new method on `App`: `func (a *App) Run() error`.
-   Move the server listening logic from `Start` to `Run`:
    -   HTTP vs. HTTPS (autocert) logic.
    -   Binding to the configured port or domain.

## Step 4: Update `Start` for Backward Compatibility (core/framework)
-   Modify `Start(cfg *Config) error` to:
    ```go
    app, err := NewApp(cfg)
    if err != nil {
        return err
    }
    defer app.DAO.Close()
    return app.Run()
    ```

## Step 5: Verification
-   **Unit Tests**: Run all existing tests to ensure no regressions.
-   **Integration Test**: Verify a small Go application can successfully call `NewApp`, add a route, and start the server.
-   **Rare Rack Integration**: Use the refactored framework in the `rare-rack` project's `main.go`.

## Step 6: Upstream Sync
-   Once verified locally in `tmp`, apply the same changes to the upstream repository at `/home/innomon/orez/aigen-cms`.
