# Specification: Interaction Service (Messaging E-trail)

## Goal
Establish a unified, persistent log of all "Interactions" (messages, commands, media) between users and AIGenApp across all channels (Web/A2UI, WhatsApp, Email, etc.). This "E-trail" serves as the primary memory source for LLM agents and provides a secure audit log for business transactions.

## Data Model: `Interaction`
All interactions will be stored in the `aigen_records` table under the namespace `aigen.core.descriptors.Interaction`.

| Field | Type | Description |
| :--- | :--- | :--- |
| `Id` | `string` | Unique NanoID for the interaction. |
| `UserId` | `*int64` | Optional reference to a system User ID. |
| `ChannelType` | `string` | The source/target channel (e.g., `whatsapp`, `web`, `email`). |
| `Identifier` | `string` | Channel-specific identity (e.g., phone number, email, session ID). |
| `Direction` | `string` | `inbound` (from user) or `outbound` (from agent/system). |
| `ContentType` | `string` | `text`, `image`, `audio`, `video`, `document`, or `action`. |
| `Content` | `string` | The raw text or a URL to the stored asset. |
| `Metadata` | `map` | Channel-specific data (Message SID, MimeType, Browser Info). |
| `Status` | `string` | `pending`, `delivered`, `processed`, `failed`. |
| `AgentID` | `string` | Optional ID of the agent that handled/generated the interaction. |
| `CreatedAt` | `time.Time` | Timestamp of the event. |

## Service Interface: `IInteractionService`
```go
type IInteractionService interface {
    // Log persists a new interaction.
    Log(ctx context.Context, interaction *Interaction) error
    
    // GetHistory retrieves the last N interactions for a specific user/identifier.
    GetHistory(ctx context.Context, identifier string, limit int) ([]*Interaction, error)
    
    // UpdateStatus updates the delivery or processing status of an interaction.
    UpdateStatus(ctx context.Context, id string, status string, errStr string) error
    
    // GetPendingOutbound retrieves interactions waiting for delivery via a gateway.
    GetPendingOutbound(ctx context.Context, channel ChannelType) ([]*Interaction, error)
}
```

## Integration Points

### 1. `ChannelService` (Inbound)
When a gateway calls `HandleInbound`, the `ChannelService` will:
1.  Map the identifier to a `UserId` if possible.
2.  Call `InteractionService.Log` with `direction: inbound`.
3.  Trigger the Agent Router with the logged Interaction ID.

### 2. `ChannelService` (Outbound)
`SendNotification` and Agent replies will:
1.  Call `InteractionService.Log` with `direction: outbound` and `status: pending`.
2.  Attempt delivery via the configured Gateway.
3.  Update status to `delivered` or `failed` based on the response.

### 3. Agentic Workflows
Agents (Router, App, UI) can query `GetHistory` to build a "sliding window" context of the conversation, allowing for multi-turn reasoning (e.g., remembering that the user just asked about "Product X").

## Benefits
- **Persistent Memory**: Agents don't lose context between HTTP requests or Lambda executions.
- **Auditability**: Secure, timestamped logs of all user commands.
- **Reliability**: Asynchronous "Outbox" pattern prevents message loss during gateway downtime.
- **Transparency**: Admin UI can render the full conversation history for any user.
