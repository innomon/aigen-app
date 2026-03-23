package displaymodels

type DisplayType string

const (
	Text          DisplayType = "Text"
	Textarea      DisplayType = "Textarea"
	Editor        DisplayType = "Editor"
	Number        DisplayType = "Number"
	Date          DisplayType = "Date"
	Datetime      DisplayType = "Datetime"
	LocalDatetime DisplayType = "LocalDatetime"
	Image         DisplayType = "Image"
	Gallery       DisplayType = "Gallery"
	File          DisplayType = "File"
	Dictionary    DisplayType = "Dictionary"
	Dropdown      DisplayType = "Dropdown"
	Multiselect   DisplayType = "Multiselect"
	Lookup        DisplayType = "Lookup"
	TreeSelect    DisplayType = "TreeSelect"
	Picklist      DisplayType = "Picklist"
	Tree          DisplayType = "Tree"
	EditTable     DisplayType = "EditTable"
)

func (d DisplayType) IsAsset() bool {
	return d == File || d == Image || d == Gallery
}

func (d DisplayType) IsCsv() bool {
	return d == Gallery || d == Multiselect
}
