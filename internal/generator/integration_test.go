package generator_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hay-kot/gobusgen/internal/generator"
	"github.com/hay-kot/gobusgen/internal/parser"
)

// TestIntegration_ParseAndGenerate exercises the full pipeline: write Go source
// with event types and a map variable, parse it, generate the event bus, write
// both files into a temp directory, and verify the result compiles.
func TestIntegration_ParseAndGenerate(t *testing.T) {
	dir := t.TempDir()

	source := `package demo

type OrderCreated struct {
	OrderID string
	Amount  float64
}

type OrderShipped struct {
	OrderID    string
	TrackingNo string
}

var Events = map[string]any{
	"order.created": OrderCreated{},
	"order.shipped": OrderShipped{},
}
`
	if err := os.WriteFile(filepath.Join(dir, "events.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	input, err := parser.Parse(dir, "Events")
	if err != nil {
		t.Fatalf("parser.Parse: %v", err)
	}

	if input.PackageName != "demo" {
		t.Fatalf("PackageName = %q, want %q", input.PackageName, "demo")
	}

	if len(input.Events) != 2 {
		t.Fatalf("Events count = %d, want 2", len(input.Events))
	}

	src, err := generator.Generate(input)
	if err != nil {
		t.Fatalf("generator.Generate: %v", err)
	}

	outPath := filepath.Join(dir, "eventbus.gen.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a go.mod so `go vet` works in the temp directory
	goMod := "module demo\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Verify the generated code compiles
	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code does not compile:\n%s\n%v", out, err)
	}
}

func TestIntegration_ConstKeys(t *testing.T) {
	dir := t.TempDir()

	consts := `package demo

type EventName string

const (
	OrderCreatedEvent EventName = "order.created"
	OrderShippedEvent EventName = "order.shipped"
)
`

	source := `package demo

type OrderCreated struct {
	OrderID string
	Amount  float64
}

type OrderShipped struct {
	OrderID    string
	TrackingNo string
}

var Events = map[string]any{
	string(OrderCreatedEvent): OrderCreated{},
	string(OrderShippedEvent): OrderShipped{},
}
`
	if err := os.WriteFile(filepath.Join(dir, "consts.go"), []byte(consts), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "events.go"), []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}

	input, err := parser.Parse(dir, "Events")
	if err != nil {
		t.Fatalf("parser.Parse: %v", err)
	}

	if len(input.Events) != 2 {
		t.Fatalf("Events count = %d, want 2", len(input.Events))
	}

	src, err := generator.Generate(input)
	if err != nil {
		t.Fatalf("generator.Generate: %v", err)
	}

	outPath := filepath.Join(dir, "eventbus.gen.go")
	if err := os.WriteFile(outPath, src, 0o644); err != nil {
		t.Fatal(err)
	}

	goMod := "module demo\n\ngo 1.22\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command("go", "vet", "./...")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generated code does not compile:\n%s\n%v", out, err)
	}
}
