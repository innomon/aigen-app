package descriptors

type DataType string

const (
	Int                DataType = "Int"
	Datetime           DataType = "Datetime"
	Text               DataType = "Text"
	String             DataType = "String"
	Boolean            DataType = "Boolean"
	Float              DataType = "Float"
	DataTypeLookup     DataType = "Lookup"
	DataTypeJunction   DataType = "Junction"
	DataTypeCollection DataType = "Collection"
)

func (d DataType) IsCompound() bool {
	return d == DataTypeLookup || d == DataTypeJunction || d == DataTypeCollection
}

func (d DataType) IsLocal() bool {
	return d != DataTypeJunction && d != DataTypeCollection
}
