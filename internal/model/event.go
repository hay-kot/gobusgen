package model

// EventDef describes a single event extracted from the map variable.
type EventDef struct {
	Name        string // e.g. "recipe.mutation"
	PayloadType string // e.g. "MutationEvent"
}

// GenerateInput is the complete input for the code generator.
type GenerateInput struct {
	PackageName string
	VarName     string // name of the source map variable (e.g. "Events")
	Events      []EventDef
}
