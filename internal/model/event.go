package model

import (
	"strings"
	"unicode"
)

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

// PascalCase converts a dotted/underscored/hyphenated name to PascalCase.
// e.g. "recipe.mutation" -> "RecipeMutation"
// e.g. "shopping_list.cleanup" -> "ShoppingListCleanup"
// e.g. "data-sync.complete" -> "DataSyncComplete"
func PascalCase(s string) string {
	var b strings.Builder
	upper := true

	for _, r := range s {
		switch {
		case r == '.' || r == '_' || r == '-':
			upper = true
		case upper:
			b.WriteRune(unicode.ToUpper(r))
			upper = false
		default:
			b.WriteRune(r)
		}
	}

	return b.String()
}
