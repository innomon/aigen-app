package descriptors

import (
	"time"
)

const CommentTableName = "__comments"

type Comment struct {
	Id             string     `json:"id" mapstructure:"id"`
	EntityName     string     `json:"entityName" mapstructure:"entityName"`
	RecordId       int64      `json:"recordId" mapstructure:"recordId"`
	CreatedBy      string     `json:"createdBy" mapstructure:"createdBy"`
	Content        string     `json:"content" mapstructure:"content"`
	Parent         *string    `json:"parent" mapstructure:"parent"`
	Mention        *string    `json:"mention" mapstructure:"mention"`
	PublishedAt    *time.Time `json:"publishedAt" mapstructure:"publishedAt"`
	CreatedAt      time.Time  `json:"createdAt" mapstructure:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt" mapstructure:"updatedAt"`
}
