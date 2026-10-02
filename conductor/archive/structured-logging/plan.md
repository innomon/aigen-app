# Implementation Plan: Structured Logging & Log Rotation

This plan outlines the steps required to implement the structured logging pattern.

## Phase 1: Core Logger and Rotator Implementation
- [x] Create `utils/logger/rotator.go` with size-based rotation and thread safety.
- [x] Create `utils/logger/logger.go` containing `MultiHandler`, custom logging configurations, and global logger initialization.
- [x] Write unit tests for `LogRotator` in `utils/logger/rotator_test.go`.
- [x] Write unit tests for `MultiHandler` and caller frames trace in `utils/logger/logger_test.go`.

## Phase 2: Configuration Integration
- [x] Update config structure in `framework/config.go` (or wherever configuration structure is defined) to read logging config parameters.
- [x] Update default `config.yaml` and `config.yaml.sample` with log settings under a `log:` namespace.

## Phase 3: Codebase Integration & Replacement
- [x] Initialize the structured logger in `framework/init.go` or `main.go`.
- [x] Replace standard `log.Print/Printf/Println` and `log.Fatal` usages with the new logger.
- [x] Ensure compilation and pass all standard tests.
