package descriptors

import (
	"time"
)

type ChannelType string

const (
	ChannelWhatsApp ChannelType = "whatsapp"
	ChannelEmail    ChannelType = "email"
	ChannelSignal   ChannelType = "signal"
	ChannelTelegram ChannelType = "telegram"
	ChannelX        ChannelType = "x"
	ChannelBluesky  ChannelType = "bluesky"
)

const UserChannelTableName = "__user_channels"
const AuthLogTableName = "__auth_logs"

type UserChannel struct {
	Id              int64       `json:"id" mapstructure:"id"`
	UserId          int64       `json:"userId" mapstructure:"user_id"`
	AgentID         string      `json:"agentId" mapstructure:"agent_id"` // A2A Agent ID
	ChannelType     ChannelType `json:"channelType" mapstructure:"channel_type"`
	Identifier      string      `json:"identifier" mapstructure:"identifier"`
	IsAuthenticated bool        `json:"isAuthenticated" mapstructure:"is_authenticated"`
	Metadata        string      `json:"metadata" mapstructure:"metadata"` // JSON string
	CreatedAt       time.Time   `json:"createdAt" mapstructure:"created_at"`
	UpdatedAt       time.Time   `json:"updatedAt" mapstructure:"updated_at"`
}

type AuthLog struct {
	Id          int64       `json:"id" mapstructure:"id"`
	UserId      *int64      `json:"userId,omitempty" mapstructure:"user_id"`
	ChannelType ChannelType `json:"channelType" mapstructure:"channel_type"`
	Action      string      `json:"action" mapstructure:"action"` // login, verify, etc.
	IPAddress   string      `json:"ipAddress" mapstructure:"ip_address"`
	UserAgent   string      `json:"userAgent" mapstructure:"user_agent"`
	Success     bool        `json:"success" mapstructure:"success"`
	Metadata    string      `json:"metadata" mapstructure:"metadata"` // JSON string, e.g. failure reason, nonce hash
	CreatedAt   time.Time   `json:"createdAt" mapstructure:"created_at"`
}

type ChannelConfig struct {
	Enabled             bool     `json:"enabled" mapstructure:"enabled"`
	GatewayURL          string   `json:"gateway_url" mapstructure:"gateway_url"`
	PublicKey           string   `json:"public_key" mapstructure:"public_key"`
	IMAPServer          string   `json:"imap_server" mapstructure:"imap_server"`
	VerificationRequired bool     `json:"verification_required" mapstructure:"verification_required"`
}

type GuestAccessConfig struct {
	AllowedChannels []string `json:"allowed_channels" mapstructure:"allowed_channels"`
	DefaultRole     string   `json:"default_role" mapstructure:"default_role"`
}

type TrustedKey struct {
	Id        string `json:"id" mapstructure:"id"`
	PublicKey string `json:"publicKey" mapstructure:"public_key"`
}

type ChannelsConfig struct {
	WhatsApp    ChannelConfig     `json:"whatsapp" mapstructure:"whatsapp"`
	Email       ChannelConfig     `json:"email" mapstructure:"email"`
	Signal      ChannelConfig     `json:"signal" mapstructure:"signal"`
	Telegram    ChannelConfig     `json:"telegram" mapstructure:"telegram"`
	X           ChannelConfig     `json:"x" mapstructure:"x"`
	Bluesky     ChannelConfig     `json:"bluesky" mapstructure:"bluesky"`
	GuestAccess GuestAccessConfig `json:"guest_access" mapstructure:"guest_access"`
	A2AEnabled  bool              `json:"a2aEnabled" mapstructure:"a2a_enabled"`
	TrustedKeys []TrustedKey      `json:"trustedKeys" mapstructure:"trusted_keys"`
}

type APIKeyConfig struct {
	Key    string `json:"key" mapstructure:"key"`
	UserId int64  `json:"userId" mapstructure:"user_id"`
}

type MCPConfig struct {
	Enabled bool           `json:"enabled" mapstructure:"enabled"`
	APIKeys []APIKeyConfig `json:"apiKeys" mapstructure:"api_keys"`
}
