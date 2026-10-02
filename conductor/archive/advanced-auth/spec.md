# Specification: Advanced Authentication (WhatsApp & Guest Roles)

## 1. Overview
Extend the `AIGenApp` authentication system to support Guest roles by default, WhatsApp-based authentication (Reverse OTP and TOTP), and a mechanism to link multiple authentication identities to a single user account.

## 2. Requirements

### 2.1. Guest Role by Default
- Unauthenticated users must be assigned the `guest` role automatically.
- The system should support a configuration to define what `guest` can access (DocPerms for 'guest' role).
- Ensure the `JWTMiddleware` correctly identifies guest state (currently sets `userId=0` and `roles=["guest"]`).

### 2.2. WhatsApp Authentication
Implement the flows described in `conductor/docs/WHATSAPP_AUTH_INSTRUCTIONS.md`.

#### 2.2.1. Reverse OTP (Mobile-Originated)
- **Backend Token Generation**: Endpoint to generate RS256 JWT for the user's phone number.
- **Callback Endpoint**: `POST /api/v1/auth/whatsapp/callback` to receive the Gateway-signed JWT and return a 4-6 digit OTP.
- **Verification**: Endpoint to verify the OTP entered by the user in the frontend.

#### 2.2.2. TOTP (Time-based OTP)
- **Enrollment**: Generate and store a per-user 32-byte secret.
- **Verification**: Server-side verification of 6-digit codes using HMAC-SHA256, following the algorithm provided in the implementation guide.

### 2.3. User Identity Resolution & Linking
- **Scenario**: A user has an email account and later wants to use WhatsApp.
- **Logic**:
    - During WhatsApp login, search for an existing `UserChannel` with the given identifier.
    - If not found, search for a `User` record that has the same phone number in their profile/metadata.
    - If found, link the new `UserChannel` to that `User`.
    - If still not found, create a new `User` with a random/placeholder email and the `UserChannel` linked.
- **Manual Linking**: Provide an API for authenticated users to "Add/Verify WhatsApp" to their account.

## 3. Architecture Changes

### 3.1. Data Model
- **UserChannel**: Use existing `UserChannel` descriptor to store the link between `userId` and WhatsApp identifier (phone number).
- **User**: Ensure the `User` descriptor can store a phone number (e.g., in a `Phone` field or `Metadata`).

### 3.2. Services
- **AuthService**:
    - Implement `LoginByChannel`.
    - Implement `LinkChannel` logic.
- **WhatsAppService (New)**:
    - Handle JWT signing/verification for Reverse OTP.
    - Manage OTP generation and temporary storage (cache/goroutine).
    - TOTP secret management and verification.

### 3.3. API Endpoints
- `POST /api/auth/whatsapp/init`: Initialize Reverse OTP flow (returns JWT for deep link).
- `POST /api/auth/whatsapp/callback`: Callback for the WhatsApp Gateway.
- `POST /api/auth/whatsapp/verify`: Verify the OTP and issue a session JWT.
- `POST /api/auth/whatsapp/totp/enroll`: Start TOTP enrollment.
- `POST /api/auth/whatsapp/totp/verify`: Verify TOTP code.

## 4. Security
- **RS256**: Use RSA for JWT signatures between Backend and Gateway.
- **Secrets**: Store the WhatsApp Gateway public key and Backend private key securely (Config).
- **Rate Limiting**: Implement rate limiting for OTP and TOTP verification attempts.
- **Short-lived Tokens**: Ensure JWTs used in the deep link have a very short expiration (e.g., 5 mins).
