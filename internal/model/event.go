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
	Prefix      string // prefix for generated symbols (e.g. "Command" → CommandEvent, CommandBus)
	Events      []EventDef
}

// DerivePrefix returns a prefix from the var name.
// Only the "Events" suffix is recognized; everything else uses the var name
// as-is. Use the //gobusgen:prefix directive to override.
//
//	"Events"      → "" (backward compatible)
//	"OrderEvents" → "Order"
//	"Commands"    → "Commands"
//	"MyBus"       → "MyBus"
func DerivePrefix(varName string) string {
	if varName == "Events" {
		return ""
	}

	if after, ok := strings.CutSuffix(varName, "Events"); ok && after != "" {
		return after
	}

	return varName
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
