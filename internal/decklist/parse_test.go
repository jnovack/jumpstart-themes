package decklist

import (
	"strings"
	"testing"
)

func TestParseLine(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantQty    int
		wantName   string
		wantSet    string
		wantNum    int
		wantThemes []string
	}{
		{
			name:       "single theme",
			input:      "1 Bumi, King of Three Trials (TLA) 169 #Bumi",
			wantQty:    1,
			wantName:   "Bumi, King of Three Trials",
			wantSet:    "TLA",
			wantNum:    169,
			wantThemes: []string{"Bumi"},
		},
		{
			name:       "multi-word theme with number",
			input:      "1 Badgermole (TLA) 166 #Bumi #Earthbending 2 #Toph",
			wantQty:    1,
			wantName:   "Badgermole",
			wantSet:    "TLA",
			wantNum:    166,
			wantThemes: []string{"Bumi", "Earthbending 2", "Toph"},
		},
		{
			name:       "basic with qty > 1",
			input:      "7 Forest (TLA) 286 #Toph",
			wantQty:    7,
			wantName:   "Forest",
			wantSet:    "TLA",
			wantNum:    286,
			wantThemes: []string{"Toph"},
		},
		{
			name:       "no themes",
			input:      "1 Some Card (TLA) 42",
			wantQty:    1,
			wantName:   "Some Card",
			wantSet:    "TLA",
			wantNum:    42,
			wantThemes: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLine(tt.input)
			if err != nil {
				t.Fatalf("parseLine(%q): %v", tt.input, err)
			}
			if got.Qty != tt.wantQty {
				t.Errorf("Qty = %d, want %d", got.Qty, tt.wantQty)
			}
			if got.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", got.Name, tt.wantName)
			}
			if got.Set != tt.wantSet {
				t.Errorf("Set = %q, want %q", got.Set, tt.wantSet)
			}
			if got.Number != tt.wantNum {
				t.Errorf("Number = %d, want %d", got.Number, tt.wantNum)
			}
			if len(got.Themes) != len(tt.wantThemes) {
				t.Fatalf("Themes = %v, want %v", got.Themes, tt.wantThemes)
			}
			for i, theme := range tt.wantThemes {
				if got.Themes[i] != theme {
					t.Errorf("Themes[%d] = %q, want %q", i, got.Themes[i], theme)
				}
			}
		})
	}
}

func TestParseLineInvalid(t *testing.T) {
	_, err := parseLine("not a valid line")
	if err == nil {
		t.Error("expected error for invalid line, got nil")
	}
}

func TestThemeCards(t *testing.T) {
	input := `7 Forest (TLA) 286 #Toph
1 Badgermole (TLA) 166 #Toph #Bumi
1 Toph, the Blind Bandit (TLA) 198 #Toph`

	cards, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	got := ThemeCards(cards, "Toph")
	if len(got) != 3 {
		t.Fatalf("got %d cards, want 3", len(got))
	}
	// Non-basics come first, basics at the end.
	if got[len(got)-1].Name != "Forest" {
		t.Errorf("last card = %q, want Forest", got[len(got)-1].Name)
	}
}

func TestAllThemes(t *testing.T) {
	input := `1 Badgermole (TLA) 166 #Bumi #Toph
7 Forest (TLA) 286 #Toph`

	cards, err := ParseReader(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	themes := AllThemes(cards)
	if len(themes) != 2 {
		t.Fatalf("got %d themes, want 2: %v", len(themes), themes)
	}
	// AllThemes returns sorted output.
	if themes[0] != "Bumi" || themes[1] != "Toph" {
		t.Errorf("themes = %v, want [Bumi Toph]", themes)
	}
}
