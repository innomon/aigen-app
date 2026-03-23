package datamodels

type ColumnType string

const (
	Id               ColumnType = "Id"               // primary key and auto increase
	StringPrimaryKey ColumnType = "StringPrimaryKey" // primary key but not auto increase
	Int              ColumnType = "Int"
	Boolean          ColumnType = "Boolean"
	Datetime         ColumnType = "Datetime"
	CreatedTime      ColumnType = "CreatedTime" // default as current datetime
	UpdatedTime      ColumnType = "UpdatedTime" // default/onupdate set as current datetime
	Text             ColumnType = "Text"        // slow performance compare to string
	String           ColumnType = "String"      // has length limit 255
	Float            ColumnType = "Float"
)

type Column struct {
	Name   string     `json:"name"`
	Type   ColumnType `json:"type"`
	Length int        `json:"length"`
}

type Record map[string]interface{}
