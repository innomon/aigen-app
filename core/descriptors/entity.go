package descriptors

import (
	"github.com/Masterminds/squirrel"
)

type PublicationStatus string

const (
	Draft       PublicationStatus = "Draft"
	Published   PublicationStatus = "Published"
	Unpublished PublicationStatus = "Unpublished"
	Scheduled   PublicationStatus = "Scheduled"
)

type Entity struct {
	Attributes               []Attribute       `json:"attributes"`
	Name                     string            `json:"name"`
	DisplayName              string            `json:"displayName"`
	TableName                string            `json:"tableName"`
	LabelAttributeName       string            `json:"labelAttributeName"`
	PrimaryKey               string            `json:"primaryKey"`
	DefaultPageSize          int               `json:"defaultPageSize"`
	DefaultPublicationStatus PublicationStatus `json:"defaultPublicationStatus"`
	PageUrl                  string            `json:"pageUrl"`
	TagsQuery                string            `json:"tagsQuery"`
	TagsQueryParam           string            `json:"tagsQueryParam"`
	TitleTagField            string            `json:"titleTagField"`
	SubtitleTagField         string            `json:"subtitleTagField"`
	ContentTagField          string            `json:"contentTagField"`
	ImageTagField            string            `json:"imageTagField"`
	PublishTimeTagField      string            `json:"publishTimeTagField"`
}

type LoadedEntity struct {
	Entity
	LoadedAttributes           []LoadedAttribute
	PrimaryKeyAttribute        LoadedAttribute
	LabelAttribute             LoadedAttribute
	DeletedAttribute           LoadedAttribute
	PublicationStatusAttribute LoadedAttribute
	UpdatedAtAttribute         LoadedAttribute
}

func (e *Entity) SelectQuery(builder squirrel.StatementBuilderType) squirrel.SelectBuilder {
	columns := []string{e.PrimaryKey}
	for _, attr := range e.Attributes {
		if attr.DataType.IsLocal() {
			columns = append(columns, attr.Field)
		}
	}
	return builder.Select(columns...).From(e.TableName)
}

func (e *Entity) ToLoadedEntity() *LoadedEntity {
	loadedAttributes := make([]LoadedAttribute, len(e.Attributes))
	var primaryKeyAttribute, labelAttribute, publicationStatusAttribute, updatedAtAttribute LoadedAttribute

	for i, attr := range e.Attributes {
		loaded := attr.ToLoaded(e.TableName)
		loadedAttributes[i] = loaded
		if attr.Field == e.PrimaryKey {
			primaryKeyAttribute = loaded
		}
		if attr.Field == e.LabelAttributeName {
			labelAttribute = loaded
		}
		if attr.Field == "publicationStatus" {
			publicationStatusAttribute = loaded
		}
		if attr.Field == "updatedAt" {
			updatedAtAttribute = loaded
		}
	}

	le := &LoadedEntity{
		Entity:                     *e,
		LoadedAttributes:           loadedAttributes,
		PrimaryKeyAttribute:        primaryKeyAttribute,
		LabelAttribute:             labelAttribute,
		PublicationStatusAttribute: publicationStatusAttribute,
		UpdatedAtAttribute:         updatedAtAttribute,
		DeletedAttribute: LoadedAttribute{
			Attribute: Attribute{
				Field:    "deleted",
				DataType: Int,
			},
			TableName: e.TableName,
		},
	}
	return le
}
