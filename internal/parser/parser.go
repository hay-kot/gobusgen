package parser

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/hay-kot/gobusgen/internal/model"
)

// Parse scans all Go files in dir for a map[string]any variable named varName,
// extracts the event definitions, and returns a GenerateInput.
func Parse(dir string, varName string) (model.GenerateInput, error) {
	fset := token.NewFileSet()

	pkgs, err := parser.ParseDir(fset, dir, func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, parser.ParseComments)
	if err != nil {
		return model.GenerateInput{}, fmt.Errorf("parsing directory %s: %w", dir, err)
	}

	var (
		found       []model.EventDef
		foundPkg    string
		foundPrefix *string
		matchCount  int
	)

	for pkgName, pkg := range pkgs {
		consts := collectStringConsts(pkg.Files)

		for _, file := range pkg.Files {
			if isGeneratedFile(file) {
				continue
			}

			events, prefix, ok, extractErr := findMapVar(file, varName, consts)
			if extractErr != nil {
				return model.GenerateInput{}, fmt.Errorf("variable %s: %w", varName, extractErr)
			}
			if !ok {
				continue
			}

			matchCount++
			found = events
			foundPkg = pkgName
			foundPrefix = prefix
		}
	}

	if matchCount == 0 {
		return model.GenerateInput{}, fmt.Errorf("no map[string]any variable %q found in %s", varName, dir)
	}

	if matchCount > 1 {
		return model.GenerateInput{}, fmt.Errorf("multiple map[string]any variables named %q found in %s", varName, dir)
	}

	if err := validate(found); err != nil {
		return model.GenerateInput{}, err
	}

	sort.Slice(found, func(i, j int) bool {
		return found[i].Name < found[j].Name
	})

	prefix := model.DerivePrefix(varName)
	if foundPrefix != nil {
		prefix = *foundPrefix
	}

	return model.GenerateInput{
		PackageName: foundPkg,
		VarName:     varName,
		Prefix:      prefix,
		Events:      found,
	}, nil
}

// isGeneratedFile checks for the standard Go generated file header.
func isGeneratedFile(file *ast.File) bool {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "Code generated") && strings.Contains(c.Text, "DO NOT EDIT") {
				return true
			}
		}
	}
	return false
}

// collectStringConsts scans all files for const declarations with string values.
// Returns a map from const name to unquoted string value. Handles multi-name
// declarations (const A, B = "a", "b") and inherited values in const blocks.
func collectStringConsts(files map[string]*ast.File) map[string]string {
	consts := make(map[string]string)

	// Two passes: first collects literals and resolves ident references where
	// the referenced const was already seen. Second pass resolves any remaining
	// ident references that couldn't resolve due to file iteration order.
	for range 2 {
		for _, file := range files {
			for _, decl := range file.Decls {
				genDecl, ok := decl.(*ast.GenDecl)
				if !ok || genDecl.Tok != token.CONST {
					continue
				}

				// Track the last explicit values for inherited const specs.
				var lastValues []string

				for _, spec := range genDecl.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok || len(vs.Names) == 0 {
						continue
					}

					if len(vs.Values) > 0 {
						lastValues = resolveStringValues(vs.Values, consts)
					}

					// Apply resolved values (explicit or inherited) to names.
					for i, name := range vs.Names {
						if i < len(lastValues) {
							consts[name.Name] = lastValues[i]
						}
					}
				}
			}
		}
	}

	return consts
}

// resolveStringValues extracts unquoted string values from a list of
// expressions. Non-string entries are represented as empty slots so that
// positional indexing is preserved for multi-name declarations.
func resolveStringValues(exprs []ast.Expr, consts map[string]string) []string {
	vals := make([]string, len(exprs))
	for i, expr := range exprs {
		switch e := expr.(type) {
		case *ast.BasicLit:
			if e.Kind != token.STRING {
				continue
			}
			val, err := strconv.Unquote(e.Value)
			if err != nil {
				continue
			}
			vals[i] = val
		case *ast.Ident:
			if val, ok := consts[e.Name]; ok {
				vals[i] = val
			}
		}
	}
	return vals
}

// resolveStringKey extracts the string value from a map key expression.
// Supports string literals, bare const references, and string() conversions.
func resolveStringKey(key ast.Expr, consts map[string]string) (string, error) {
	switch k := key.(type) {
	case *ast.BasicLit:
		if k.Kind != token.STRING {
			return "", fmt.Errorf("map key must be a string literal, got %s", k.Kind)
		}
		return strconv.Unquote(k.Value)

	case *ast.Ident:
		val, ok := consts[k.Name]
		if !ok {
			return "", fmt.Errorf("constant %q not found in package; only package-level string constants are supported", k.Name)
		}
		return val, nil

	case *ast.CallExpr:
		// Handle string(ConstName) conversion
		fnIdent, ok := k.Fun.(*ast.Ident)
		if !ok || fnIdent.Name != "string" || len(k.Args) != 1 {
			return "", fmt.Errorf("map key must be a string literal, constant, or string() conversion")
		}

		argIdent, ok := k.Args[0].(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("map key must be a string literal, constant, or string() conversion")
		}

		val, ok := consts[argIdent.Name]
		if !ok {
			return "", fmt.Errorf("constant %q not found in package; only package-level string constants are supported", argIdent.Name)
		}
		return val, nil

	default:
		return "", fmt.Errorf("map key must be a string literal, constant, or string() conversion")
	}
}

// findMapVar looks for a top-level var declaration matching:
//
//	var <varName> = map[string]any{ ... }
//
// Returns the extracted events, an optional prefix from a //gobusgen:prefix directive
// (nil means no directive found), whether the variable was found, and any extraction error.
func findMapVar(file *ast.File, varName string, consts map[string]string) ([]model.EventDef, *string, bool, error) {
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.VAR {
			continue
		}

		for _, spec := range genDecl.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 || vs.Names[0].Name != varName {
				continue
			}

			if len(vs.Values) == 0 {
				continue
			}

			comp, ok := vs.Values[0].(*ast.CompositeLit)
			if !ok {
				continue
			}

			if !isMapStringAny(comp.Type) {
				continue
			}

			events, err := extractEvents(comp, consts)
			if err != nil {
				return nil, nil, true, err
			}

			prefix := extractPrefixDirective(genDecl.Doc)
			if prefix == nil {
				prefix = extractPrefixDirective(vs.Doc)
			}

			return events, prefix, true, nil
		}
	}

	return nil, nil, false, nil
}

// extractPrefixDirective scans a comment group for a //gobusgen:prefix directive.
// Returns nil if no directive found, or a pointer to the prefix value.
func extractPrefixDirective(doc *ast.CommentGroup) *string {
	if doc == nil {
		return nil
	}

	for _, c := range doc.List {
		text := strings.TrimSpace(c.Text)
		after, ok := strings.CutPrefix(text, "//gobusgen:prefix")
		if !ok {
			continue
		}

		// "//gobusgen:prefix" with no value means explicit empty prefix
		val := strings.TrimSpace(after)
		return &val
	}

	return nil
}

// isMapStringAny checks if the expression is map[string]any.
func isMapStringAny(expr ast.Expr) bool {
	mt, ok := expr.(*ast.MapType)
	if !ok {
		return false
	}

	keyIdent, ok := mt.Key.(*ast.Ident)
	if !ok || keyIdent.Name != "string" {
		return false
	}

	valIdent, ok := mt.Value.(*ast.Ident)
	if !ok || valIdent.Name != "any" {
		return false
	}

	return true
}

// extractEvents pulls event name and payload type from each key-value pair.
func extractEvents(comp *ast.CompositeLit, consts map[string]string) ([]model.EventDef, error) {
	var events []model.EventDef

	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("expected key-value expression")
		}

		name, err := resolveStringKey(kv.Key, consts)
		if err != nil {
			return nil, err
		}

		valComp, ok := kv.Value.(*ast.CompositeLit)
		if !ok {
			return nil, fmt.Errorf("map value for %q must be a composite literal (e.g. MyType{})", name)
		}

		payloadType, err := typeExprToString(valComp.Type)
		if err != nil {
			return nil, fmt.Errorf("map value for %q: %w", name, err)
		}

		events = append(events, model.EventDef{
			Name:        name,
			PayloadType: payloadType,
		})
	}

	return events, nil
}

// typeExprToString converts a type expression to its string representation.
func typeExprToString(expr ast.Expr) (string, error) {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name, nil
	case *ast.SelectorExpr:
		pkg, ok := t.X.(*ast.Ident)
		if !ok {
			return "", fmt.Errorf("unexpected selector expression")
		}
		return pkg.Name + "." + t.Sel.Name, nil
	default:
		return "", fmt.Errorf("unsupported type expression; only simple types like MyType or pkg.Type are supported")
	}
}

func validate(events []model.EventDef) error {
	if len(events) == 0 {
		return fmt.Errorf("event map contains no event definitions")
	}

	seen := make(map[string]bool, len(events))
	symbols := make(map[string]string, len(events)) // normalized symbol -> original name

	for _, e := range events {
		if e.Name == "" {
			return fmt.Errorf("event name must not be empty")
		}

		if r, ok := containsInvalidEventRune(e.Name); ok {
			return fmt.Errorf("event name %q contains invalid character %q; only letters, digits, dots, hyphens, and underscores are allowed", e.Name, r)
		}

		if seen[e.Name] {
			return fmt.Errorf("duplicate event name %q", e.Name)
		}
		seen[e.Name] = true

		sym := model.PascalCase(e.Name)
		if prev, ok := symbols[sym]; ok {
			return fmt.Errorf("event names %q and %q produce the same generated symbol %q", prev, e.Name, sym)
		}
		symbols[sym] = e.Name

		if !isValidGoIdent(e.PayloadType) {
			return fmt.Errorf("payload type %q for event %q is not a valid Go identifier", e.PayloadType, e.Name)
		}
	}

	return nil
}

func isValidGoIdent(s string) bool {
	if s == "" {
		return false
	}

	// Allow package-qualified types like "pkg.Type"
	parts := strings.SplitN(s, ".", 2)
	for _, part := range parts {
		if part == "" {
			return false
		}
		for i, r := range part {
			if i == 0 && !unicode.IsLetter(r) && r != '_' {
				return false
			}
			if i > 0 && !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_' {
				return false
			}
		}
	}

	return true
}

// containsInvalidEventRune returns the first rune in s that is not a letter,
// digit, or one of the recognized separators (dot, hyphen, underscore).
func containsInvalidEventRune(s string) (rune, bool) {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			continue
		}
		if r == '.' || r == '_' || r == '-' {
			continue
		}
		return r, true
	}
	return 0, false
}
