package browser

import (
	"testing"

	"github.com/go-rod/rod/lib/input"
)

func TestParseKeyName(t *testing.T) {
	t.Helper()

	tests := []struct {
		name    string
		keyName string
		want    input.Key
		wantErr bool
	}{
		{"enter key", "enter", input.Enter, false},
		{"Enter uppercase", "Enter", input.Enter, false},
		{"escape key", "escape", input.Escape, false},
		{"esc shorthand", "esc", input.Escape, false},
		{"backspace", "backspace", input.Backspace, false},
		{"delete", "delete", input.Delete, false},
		{"del shorthand", "del", input.Delete, false},
		{"tab", "tab", input.Tab, false},
		{"arrow up", "arrowup", input.ArrowUp, false},
		{"arrow_up underscore", "arrow_up", input.ArrowUp, false},
		{"arrow down", "arrowdown", input.ArrowDown, false},
		{"arrow left", "arrowleft", input.ArrowLeft, false},
		{"arrow right", "arrowright", input.ArrowRight, false},
		{"space", "space", input.Space, false},
		{"home", "home", input.Home, false},
		{"end", "end", input.End, false},
		{"pageup", "pageup", input.PageUp, false},
		{"pagedown", "pagedown", input.PageDown, false},
		{"unknown key", "invalid", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseKeyName(tt.keyName)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseKeyName() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseKeyName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseModifier(t *testing.T) {
	t.Helper()

	tests := []struct {
		name     string
		modifier string
		want     input.Key
		wantErr  bool
	}{
		{"cmd modifier", "cmd", input.MetaLeft, false},
		{"meta modifier", "meta", input.MetaLeft, false},
		{"ctrl modifier", "ctrl", input.ControlLeft, false},
		{"control modifier", "control", input.ControlLeft, false},
		{"alt modifier", "alt", input.AltLeft, false},
		{"shift modifier", "shift", input.ShiftLeft, false},
		{"Cmd uppercase", "Cmd", input.MetaLeft, false},
		{"unknown modifier", "invalid", 0, true},
		{"empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseModifier(tt.modifier)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseModifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("parseModifier() = %v, want %v", got, tt.want)
			}
		})
	}
}
