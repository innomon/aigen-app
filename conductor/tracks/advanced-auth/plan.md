# Implementation Plan: Advanced Authentication

## Phase 1: Infrastructure & Configuration
- [x] Add `Phone` field to `descriptors.User` and update related logic.
- [x] Add WhatsApp Gateway configuration (URL, Public Key, Private Key) to `framework/config.go`.
- [x] Verify `JWTMiddleware` handles `guest` role correctly in all scenarios.

## Phase 2: Reverse OTP (Mobile-Originated)
- [x] Implement `WhatsAppService` for RS256 JWT generation and verification.
- [x] Create `POST /api/auth/whatsapp/init` to initiate the flow.
- [x] Create `POST /api/auth/whatsapp/callback` for Gateway interaction.
- [x] Implement temporary OTP storage (e.g., in-memory map with TTL or a simple table).
- [x] Create `POST /api/auth/whatsapp/verify` to finalise login.

## Phase 3: TOTP Implementation
- [x] Implement TOTP algorithm (HMAC-SHA256) in `WhatsAppService`.
- [x] Add enrollment logic: generate 32-byte secret and store in `User` metadata/record.
- [x] Create enrollment and verification endpoints.

## Phase 4: Identity Resolution & Linking
- [x] Implement `AuthService.LoginByChannel` to look up `UserChannel`.
- [x] Implement fuzzy matching/search for `User` by `Phone` if `UserChannel` is missing.
- [x] Implement `AuthService.LinkChannel` for existing authenticated users.

## Phase 5: Testing & Validation
- [x] Unit tests for `WhatsAppService` (JWT and TOTP logic).
- [ ] Integration tests for the full Reverse OTP flow using a mock Gateway.
- [ ] Verify `guest` role permissions in RBAC.
