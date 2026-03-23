# User Channel Context

A `user_channel` defines the link between a core system user and their identity on an external communication or social platform.

## Supported Platforms
- **Messaging**: WhatsApp, Signal, Telegram.
- **Social**: Email, X.com, Bluesky.

## Lifecycle
1.  **Registration**: A user provides their identifier (e.g., phone number or handle).
2.  **Verification**: The system verifies ownership of the identifier (e.g., via an Ed25519 JWT challenge in A2A or an OTP in Email).
3.  **Authentication**: Once verified, the `is_authenticated` flag is set to true, allowing the user to log in or receive sensitive notifications via this channel.

## Relationships
- **User**: Belongs to a single system user.
- **A2A Agent**: If the channel communicates via the **Agent2Agent (A2A)** protocol, the `agent_id` links it to the corresponding external A2A agent card.

## Key Fields
- **Identifier**: The platform-specific address (e.g., `+123456789`, `user@example.com`, `@handle`).
- **Metadata**: A JSON blob storing session-specific data, such as public keys for signature verification or platform-specific tokens.
