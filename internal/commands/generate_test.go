package commands

import "testing"

func TestDefaultOutputFilename(t *testing.T) {
	tests := []struct {
		prefix string
		want   string
	}{
		{"", "eventbus.gen.go"},
		{"Command", "commandbus.gen.go"},
		{"Order", "orderbus.gen.go"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			got := defaultOutputFilename(tt.prefix)
			if got != tt.want {
				t.Errorf("defaultOutputFilename(%q) = %q, want %q", tt.prefix, got, tt.want)
			}
		})
	}
}

func TestParseTarget(t *testing.T) {
	tests := []struct {
		input   string
		wantDir string
		wantVar string
		wantErr bool
	}{
		{
			input:   "./internal/events.Events",
			wantDir: "./internal/events",
			wantVar: "Events",
		},
		{
			input:   ".Events",
			wantDir: ".",
			wantVar: "Events",
		},
		{
			input:   "./pkg/v2.0/events.MyBus",
			wantDir: "./pkg/v2.0/events",
			wantVar: "MyBus",
		},
		{
			input:   "novar",
			wantErr: true,
		},
		{
			input:   "path.",
			wantErr: true,
		},
		{
			input:   ".",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			dir, varName, err := parseTarget(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseTarget(%q) = (%q, %q, nil), want error", tt.input, dir, varName)
				}
				return
			}

			if err != nil {
				t.Fatalf("parseTarget(%q) error: %v", tt.input, err)
			}
			if dir != tt.wantDir {
				t.Errorf("dir = %q, want %q", dir, tt.wantDir)
			}
			if varName != tt.wantVar {
				t.Errorf("varName = %q, want %q", varName, tt.wantVar)
			}
		})
	}
}
