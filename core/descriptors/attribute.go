package descriptors

import (
	"github.com/innomon/aigen-cms/utils/displaymodels"
)

type Attribute struct {
	Field       string                    `json:"field"`
	Header      string                    `json:"header"`
	DataType    DataType                  `json:"dataType"`
	DisplayType displaymodels.DisplayType `json:"displayType"`
	InList      bool                      `json:"inList"`
	InDetail    bool                      `json:"inDetail"`
	IsDefault   bool                      `json:"isDefault"`
	Options     string                    `json:"options"`
	Validation  string                    `json:"validation"`
	PermLevel   int                       `json:"permLevel"`
}

func (a *Attribute) ToLoaded(tableName string) LoadedAttribute {
	return LoadedAttribute{
		Attribute: *a,
		TableName: tableName,
	}
}

type LoadedAttribute struct {
	Attribute
	TableName  string
	Junction   *Junction
	Lookup     *Lookup
	Collection *Collection
}

type Lookup struct {
	TargetEntity *LoadedEntity
}

type Junction struct {
	JunctionEntity  *LoadedEntity
	TargetEntity    *LoadedEntity
	SourceEntity    *LoadedEntity
	SourceAttribute *LoadedAttribute
	TargetAttribute *LoadedAttribute
}

type Collection struct {
	SourceEntity  *LoadedEntity
	TargetEntity  *LoadedEntity
	LinkAttribute *LoadedAttribute
}
