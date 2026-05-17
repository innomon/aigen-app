# Reorganization Plan: Reusable Framework

The primary goal of this reorganization is to allow other developers to use this framework by creating a `main.go`, adding `apps`, and configuring agentic settings.

## Phase 1: Configuration Management (DONE)
The first step is to define the new configuration structure that downstream projects will use to configure the CMS framework.

- [x] **Create `framework` package:** Create a new directory `framework` (e.g., `framework/config.go`).
- [x] **Define `Config` struct:** Include the necessary fields for downstream customization:
  ```go
  type Config struct {
      AppsDir           string `json:"apps_dir" yaml:"apps_dir"`
      WWWRoot           string `json:"www_root" yaml:"www_root"`
      DatabaseDSN       string `json:"database_dsn" yaml:"database_dsn"` // Database name or connection string
      Domain            string `json:"domain" yaml:"domain"`
      Port              string `json:"port" yaml:"port"`
      AgenticConfigPath string `json:"agentic_config_path" yaml:"agentic_config_path"`
  }
  ```
- [x] **Implement Config Loader:** Write a function `LoadConfig(path string) (*Config, error)` to parse the YAML/JSON config file, with fallbacks to environment variables.

## Phase 2: Refactoring `main.go` to `framework/init.go` (DONE)
Move the monolithic `main.go` script into a reusable framework lifecycle function.

- [x] **Create `framework/init.go`:** Move the core logic of `main.go` into a new exported function: `func Start(cfg *Config) error`.
- [x] **Dynamic Database Initialization:** Update the `relationdbdao.CreateDao` call to use `cfg.DatabaseDSN`.
- [x] **Dynamic Apps Loading:** Update `apps.LoadAppsConfig()` and the `apps.SetupApp()` loops to use `cfg.AppsDir`.
- [x] **Dynamic Static Files:** Update `api.NewStaticApi()` and other components that rely on static files to serve from `cfg.WWWRoot`.
- [x] **Dynamic Agentic Config:** Update `services.NewChatService` to use `cfg.AgenticConfigPath`.
- [x] **Dynamic Server Port/Domain:** Replace `os.Getenv` calls with `cfg` values.

## Phase 3: Removing Hardcoded Paths in Core Packages (IN PROGRESS)
Ensure that underlying core packages don't rely on the current monolithic repository structure.

- [x] **Update `apps` package:** Modify `LoadAppsConfig()` and `SetupApp()` to accept an absolute or relative directory path (`appsDir`).
- [x] **Update `filestore` package:** Ensure local file storage respects directory configurations provided by the new config.

## Phase 4: Create the new downstream `main.go` (DONE)
Create a clean, minimalistic entry point that represents how a downstream project will use the framework.

- [x] **Create new `main.go`**:
  - Parse the configuration file location.
  - Load the configuration.
  - Call `framework.Start(config)`.

## Phase 5: Testing & Cleanup (TODO)
Verify that the standalone application still behaves as it originally did.

- [ ] **Create a sample `config.yaml`**: Create a default configuration in the repository root for testing.
- [ ] **Test the server startup**: Run `go run main.go config.yaml` to verify all APIs, databases, and apps load correctly.
- [ ] **Test external apps**: Temporarily move the `apps` and `wwwroot` directories outside of the project root and start the application using an updated `config.yaml` to ensure it works externally.
