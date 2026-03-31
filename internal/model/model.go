package model

type ServiceDef struct {
	PackageName string
	Services    []InterfaceDef
	Structs     []StructDef
}

type InterfaceDef struct {
	Name    string
	Methods []MethodDef
}

type MethodDef struct {
	Name      string
	EventType string
	ReqType   string
}

type StructDef struct {
	Name   string
	Fields []FieldDef
}

type FieldDef struct {
	Name        string
	Type        string
	Tag         string // original tag from schema, or generated json:"snake_case"
	ValidateTag string // value of validate:"..." struct tag
}
