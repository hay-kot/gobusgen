package generator_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/hay-kot/gobusgen/internal/generator"
	"github.com/hay-kot/gobusgen/internal/model"
)

var update = flag.Bool("update", false, "update golden files")

func TestGenerate(t *testing.T) {
	tests := []struct {
		name  string
		input model.GenerateInput
	}{
		{
			name: "single_event",
			input: model.GenerateInput{
				PackageName: "events",
				VarName:     "Events",
				Events: []model.EventDef{
					{Name: "recipe.mutation", PayloadType: "MutationEvent"},
				},
			},
		},
		{
			name: "multiple_events",
			input: model.GenerateInput{
				PackageName: "events",
				VarName:     "Events",
				Events: []model.EventDef{
					{Name: "alert.fired", PayloadType: "AlertEvent"},
					{Name: "order.placed", PayloadType: "OrderEvent"},
					{Name: "user.created", PayloadType: "UserEvent"},
				},
			},
		},
		{
			name: "underscore_names",
			input: model.GenerateInput{
				PackageName: "mybus",
				VarName:     "MyBus",
				Events: []model.EventDef{
					{Name: "data-sync.complete", PayloadType: "SyncEvent"},
					{Name: "shopping_list.cleanup", PayloadType: "CleanupEvent"},
				},
			},
		},
		{
			name: "prefixed_bus",
			input: model.GenerateInput{
				PackageName: "commands",
				VarName:     "Commands",
				Prefix:      "Command",
				Events: []model.EventDef{
					{Name: "order.create", PayloadType: "CreateOrderCmd"},
					{Name: "order.cancel", PayloadType: "CancelOrderCmd"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generator.Generate(tt.input)
			if err != nil {
				t.Fatalf("Generate() error: %v", err)
			}

			goldenPath := filepath.Join("testdata", tt.name+".golden")

			if *update {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatal(err)
				}
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("reading golden file (run with -update to create): %v", err)
			}

			if string(got) != string(want) {
				t.Errorf("output does not match golden file %s\n\ngot:\n%s\n\nwant:\n%s", goldenPath, got, want)
			}
		})
	}
}

func TestPascalCase(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"recipe.mutation", "RecipeMutation"},
		{"user.registration", "UserRegistration"},
		{"shopping_list.cleanup", "ShoppingListCleanup"},
		{"data-sync.complete", "DataSyncComplete"},
		{"simple", "Simple"},
		{"a.b.c", "ABC"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := model.PascalCase(tt.input)
			if got != tt.want {
				t.Errorf("PascalCase(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
