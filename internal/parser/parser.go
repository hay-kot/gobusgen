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
		found      []model.EventDef
		foundPkg   string
		matchCount int
	)

	for pkgName, pkg := range pkgs {
		for _, file := range pkg.Files {
			// Skip generated files
			if isGeneratedFile(file) {
				continue
			}

			events, ok := findMapVar(file, varName)
			if !ok {
				continue
			}

			matchCount++
			found = events
			foundPkg = pkgName
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

	return model.GenerateInput{
		PackageName: foundPkg,
		VarName:     varName,
		Events:      found,
	}, nil
}

// isGeneratedFile checks for the standard "Code generated" comment.
func isGeneratedFile(file *ast.File) bool {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if strings.Contains(c.Text, "Code generated") {
				return true
			}
		}
	}
	return false
}

// findMapVar looks for a top-level var declaration matching:
//
//	var <varName> = map[string]any{ ... }
func findMapVar(file *ast.File, varName string) ([]model.EventDef, bool) {
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

			events, err := extractEvents(comp)
			if err != nil {
				continue
			}

			return events, true
		}
	}

	return nil, false
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
func extractEvents(comp *ast.CompositeLit) ([]model.EventDef, error) {
	var events []model.EventDef

	for _, elt := range comp.Elts {
		kv, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			return nil, fmt.Errorf("expected key-value expression")
		}

		keyLit, ok := kv.Key.(*ast.BasicLit)
		if !ok || keyLit.Kind != token.STRING {
			return nil, fmt.Errorf("map key must be a string literal")
		}

		name, err := strconv.Unquote(keyLit.Value)
		if err != nil {
			return nil, fmt.Errorf("unquoting map key %s: %w", keyLit.Value, err)
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
		return "", fmt.Errorf("unsupported type expression %T", expr)
	}
}

func validate(events []model.EventDef) error {
	seen := make(map[string]bool, len(events))
	symbols := make(map[string]string, len(events)) // normalized symbol -> original name

	for _, e := range events {
		if e.Name == "" {
			return fmt.Errorf("event name must not be empty")
		}

		for _, r := range e.Name {
			if unicode.IsSpace(r) {
				return fmt.Errorf("event name %q contains whitespace", e.Name)
			}
		}

		if seen[e.Name] {
			return fmt.Errorf("duplicate event name %q", e.Name)
		}
		seen[e.Name] = true

		sym := normalizeName(e.Name)
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

// normalizeName converts an event name to its PascalCase symbol form,
// matching the generator template's pascalCase function.
func normalizeName(s string) string {
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
