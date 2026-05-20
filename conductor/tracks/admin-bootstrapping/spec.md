# Specification: Admin User Bootstrapping

## 1. Overview
Implement a secure, robust, and environment-aware bootstrapping mechanism for initial admin credentials in `AIGenApp`. The goal is to allow immediate out-of-the-box operations for local test runs (in-memory) while enforcing high-security standards for PostgreSQL and production runs.

## 2. Requirements

### 2.1. In-Memory Test Runs (`memory://`)
* **Zero Config**: If the database starts clean in in-memory mode, automatically seed a standard administrator.
* **Credentials**:
  * **Email**: `admin@aigen.local`
  * **Password**: `adminpassword`
  * **Roles**: `sa`, `admin`, `user`

### 2.2. PostgreSQL / Production Runs (`postgres://...`)
* **Environment Configuration**: Check for environment variables:
  * `AIGEN_ADMIN_EMAIL`
  * `AIGEN_ADMIN_PASSWORD`
* **Secure Fallback (Auto-Generation)**: If no variables are set and no admin user exists:
  * Generate a secure, high-entropy 16-character password using `crypto/rand`.
  * Register `admin@aigen.local` with the generated password.
  * Print a prominent warning banner containing the credentials to stdout/stderr.

### 2.3. User Table Scanner
* The bootstrapping procedure must run once on server startup.
* It must search for existing users having roles `sa` or `admin`. If any matching user exists, the bootstrapping sequence is skipped immediately to prevent security regressions or account overrides.

### 2.4. Handcrafted CLI Commands
* Provide CLI utilities to create/reset credentials directly from the host terminal.
* **Rule Constraint**: Do NOT use Cobra or Pflag libraries. All argument parsing must be handcrafted using the standard library `flag` and `os.Args`.
* Sub-commands:
  * `admin create --email=<email> --password=<pass>`
  * `admin reset-pass --email=<email> --password=<pass>`

## 3. Architecture Changes

### 3.1. Framework Configuration
Add helper fields to `framework.Config` to optionally declare admin parameters in `config.yaml`.

### 3.2. AuthService
Implement:
* `BootstrapAdmin(ctx context.Context, defaultEmail, defaultPassword string, isTestEnv bool) error`
* Internal password hash generation using bcrypt.

### 3.3. Server Lifecycle Integration
* Run the bootstrapping checks in `framework/init.go` immediately after starting the databases and initializing `AuthService`.
