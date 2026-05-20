# Implementation Plan: Admin User Bootstrapping

## Phase 1: Infrastructure & Configuration
- [x] Add `Admin` configurations to `framework/Config` structure.
- [x] Support loading `AIGEN_ADMIN_EMAIL` and `AIGEN_ADMIN_PASSWORD` env overrides.

## Phase 2: AuthService Bootstrapping Core Logic
- [x] Implement `BootstrapAdmin(ctx context.Context, defaultEmail, defaultPassword string, isTestEnv bool) error` in `AuthService`.
- [x] Implement verification logic to check if any admin accounts exist inside `__users`.
- [x] Add `crypto/rand` utility for secure random password generation.
- [x] Print unmissable security warning banner when random passwords are used.

## Phase 3: Framework Integration
- [x] Invoke `BootstrapAdmin` inside `framework/init.go` during server startup.
- [x] Ensure automatic test-mode detection seeds default credentials when `DatabaseDSN == "memory://"`.

## Phase 4: Handcrafted CLI Command Registry
- [x] Implement a custom subcommand router for `admin` CLI.
- [x] Implement `admin create` subcommand to register super-admins securely via terminal.
- [x] Implement `admin reset-pass` subcommand to reset admin credentials via terminal.

## Phase 5: Testing & Validation
- [x] Add unit test suite for bootstrapping logic.
- [x] Verify database-specific behavior for both memory and PostgreSQL drivers.
