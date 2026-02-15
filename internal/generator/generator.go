package generator

import (
	"bytes"
	"fmt"
	"go/format"

	"github.com/hay-kot/gobusgen/internal/model"
)

// Generate executes the template with the given input and returns formatted Go source.
func Generate(input model.GenerateInput) ([]byte, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, input); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	src, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting generated code: %w", err)
	}

	return src, nil
}
