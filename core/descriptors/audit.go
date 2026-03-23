package descriptors

import (
	"time"
)

const AuditLogTableName = "__auditlog"

type ActionType string

const (
	ActionCreate ActionType = "Create"
	ActionUpdate ActionType = "Update"
	ActionDelete ActionType = "Delete"
)

type AuditLog struct {
	Id          int64                  `json:"id" mapstructure:"id"`
	UserId      string                 `json:"userId" mapstructure:"user_id"`
	UserName    string                 `json:"userName" mapstructure:"user_name"`
	Action      ActionType             `json:"action" mapstructure:"action"`
	EntityName  string                 `json:"entityName" mapstructure:"entity_name"`
	RecordId    string                 `json:"recordId" mapstructure:"record_id"`
	RecordLabel string                 `json:"recordLabel" mapstructure:"record_label"`
	Payload     map[string]interface{} `json:"payload" mapstructure:"payload"`
	CreatedAt   time.Time              `json:"createdAt" mapstructure:"created_at"`
}
