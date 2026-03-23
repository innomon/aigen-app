# Refactor Specification: AiGen CMS Extensibility

## 1. Goal
The primary goal is to refactor the `AiGen CMS` framework to support **extensibility** from consuming applications (like Rare Rack). Applications should be able to initialize the framework's core services and router, and then register their own custom routes, middleware, and logic *before* the server starts.

## 2. Problem Statement
Currently, `framework.Start(cfg)` is a "black box" that:
1.  Initializes all internal services and DAOs.
2.  Creates and configures the `chi` router.
3.  Registers all built-in API routes.
4.  Starts the HTTP/HTTPS server and blocks.

This prevents the consumer from:
-   Adding custom routes to the main router.
-   Accessing the initialized services (like `EntityService`) for custom handlers.
-   Injecting custom middleware.

## 3. Technical Requirements

### 3.1 Decouple Initialization and Execution
-   Introduce an `App` struct (or similar) that holds the initialized state:
    -   `Router (chi.Router)`: The main application router.
    -   `Services`: Expose core service interfaces (Entity, Schema, Auth, etc.).
    -   `Config`: The loaded framework configuration.
-   Create a `NewApp(cfg *Config)` function that performs the setup logic but does *not* start the server.

### 3.2 Service Exposure
-   The `App` struct should provide access to its internal services via public fields or getter methods.
-   Service interfaces (defined in `core/services/interface.go`) should be the primary way to interact with the framework's logic.

### 3.3 Flexible Server Execution
-   The `App` struct should have a `Run()` (or `Start()`) method to begin the HTTP/HTTPS listener.
-   Support both standard HTTP and `autocert` (HTTPS) as per the current implementation.

### 3.4 Backward Compatibility
-   Keep the existing `framework.Start(cfg)` function as a convenience wrapper that calls `NewApp` and then `app.Run()`.

## 4. Use Case: Rare Rack
Rare Rack should be able to:
1.  Call `app, err := framework.NewApp(cfg)`.
2.  Use `app.Router.Get("/products/{id}", ...)` to add custom routes.
3.  Use `app.EntityService.Single(...)` inside those custom routes.
4.  Call `app.Run()` to start the server.
