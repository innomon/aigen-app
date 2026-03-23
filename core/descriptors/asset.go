package descriptors

import (
	"time"
)

const AssetTableName = "__assets"
const AssetLinkTableName = "__assetLinks"

type Asset struct {
	Id        int64                  `json:"id" mapstructure:"id"`
	Path      string                 `json:"path" mapstructure:"path"`
	Url       string                 `json:"url" mapstructure:"url"`
	Name      string                 `json:"name" mapstructure:"name"`
	Title     string                 `json:"title" mapstructure:"title"`
	Size      int64                  `json:"size" mapstructure:"size"`
	Type      string                 `json:"type" mapstructure:"type"`
	Metadata  map[string]interface{} `json:"metadata" mapstructure:"metadata"`
	CreatedAt time.Time              `json:"createdAt" mapstructure:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt" mapstructure:"updatedAt"`
	CreatedBy string                 `json:"createdBy" mapstructure:"createdBy"`
}

type AssetLink struct {
	Id         int64     `json:"id" mapstructure:"id"`
	EntityName string    `json:"entityName" mapstructure:"entityName"`
	RecordId   int64     `json:"recordId" mapstructure:"recordId"`
	AssetId    int64     `json:"assetId" mapstructure:"assetId"`
	CreatedAt  time.Time `json:"createdAt" mapstructure:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt" mapstructure:"updatedAt"`
}
