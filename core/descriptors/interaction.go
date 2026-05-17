package descriptors

import (
	"time"
)

const InteractionNamespace = "aigen.core.descriptors.Interaction"

type Interaction struct {
	Id          string                 `json:"id" mapstructure:"id"`
	UserId      *int64                 `json:"userId,omitempty" mapstructure:"user_id"`
	ChannelType ChannelType            `json:"channelType" mapstructure:"channel_type"`
	Identifier  string                 `json:"identifier" mapstructure:"identifier"`
	Direction   string                 `json:"direction" mapstructure:"direction"`   // inbound or outbound
	ContentType string                 `json:"contentType" mapstructure:"content_type"` // text, image, etc.
	Content     string                 `json:"content" mapstructure:"content"`         // Text or URL
	Metadata    map[string]interface{} `json:"metadata,omitempty" mapstructure:"metadata"`
	Status      string                 `json:"status" mapstructure:"status"`           // pending, processed, failed
	AgentID     string                 `json:"agentId,omitempty" mapstructure:"agent_id"`
	CreatedAt   time.Time              `json:"createdAt" mapstructure:"created_at"`
}
