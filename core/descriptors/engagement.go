package descriptors

import (
	"time"
)

const EngagementStatusTableName = "__engagements"
const EngagementCountTableName = "__engagement_counts"

type EngagementStatus struct {
	Id             int64      `json:"id" mapstructure:"id"`
	EntityName     string     `json:"entityName" mapstructure:"entityName"`
	RecordId       string     `json:"recordId" mapstructure:"recordId"`
	EngagementType string     `json:"engagementType" mapstructure:"engagementType"`
	UserId         string     `json:"userId" mapstructure:"userId"`
	IsActive       bool       `json:"isActive" mapstructure:"isActive"`
	Title          string     `json:"title" mapstructure:"title"`
	Url            string     `json:"url" mapstructure:"url"`
	Image          string     `json:"image" mapstructure:"image"`
	Subtitle       string     `json:"subtitle" mapstructure:"subtitle"`
	PublishedAt    *time.Time `json:"publishedAt" mapstructure:"publishedAt"`
	CreatedAt      time.Time  `json:"createdAt" mapstructure:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt" mapstructure:"updatedAt"`
}

type EngagementCount struct {
	Id             int64     `json:"id" mapstructure:"id"`
	EntityName     string    `json:"entityName" mapstructure:"entityName"`
	RecordId       string    `json:"recordId" mapstructure:"recordId"`
	EngagementType string    `json:"engagementType" mapstructure:"engagementType"`
	Count          int64     `json:"count" mapstructure:"count"`
	CreatedAt      time.Time `json:"createdAt" mapstructure:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt" mapstructure:"updatedAt"`
}
