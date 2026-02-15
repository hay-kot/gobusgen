package model

import "testing"

func TestDerivePrefix(t *testing.T) {
	tests := []struct {
		varName string
		want    string
	}{
		{"Events", ""},
		{"OrderEvents", "Order"},
		{"UserEvents", "User"},
		{"Commands", "Commands"},
		{"Notifications", "Notifications"},
		{"MyBus", "MyBus"},
		{"E", "E"},
	}

	for _, tt := range tests {
		t.Run(tt.varName, func(t *testing.T) {
			got := DerivePrefix(tt.varName)
			if got != tt.want {
				t.Errorf("DerivePrefix(%q) = %q, want %q", tt.varName, got, tt.want)
			}
		})
	}
}
