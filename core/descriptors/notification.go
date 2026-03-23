package descriptors

import (
	"time"
)

const NotificationTableName = "__notifications"

type Notification struct {
	Id          int64     `json:"id" mapstructure:"id"`
	UserId      string    `json:"userId" mapstructure:"user_id"`
	SenderId    string    `json:"senderId" mapstructure:"sender_id"`
	ActionType  string    `json:"actionType" mapstructure:"action_type"`
	MessageType string    `json:"messageType" mapstructure:"message_type"`
	Message     string    `json:"message" mapstructure:"message"`
	Url         string    `json:"url" mapstructure:"url"`
	IsRead      bool      `json:"isRead" mapstructure:"is_read"`
	CreatedAt   time.Time `json:"createdAt" mapstructure:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" mapstructure:"updated_at"`
}
