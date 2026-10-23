package callname

import (
	"strings"
	"testing"
	"time"
)

func TestNewPRT(t *testing.T) {
	prt := NewPRT()

	if !strings.HasPrefix(prt, "PRT") {
		t.Errorf("NewPRT() = %q, expected prefix 'PRT'", prt)
	}

	ts := prt[3:]
	if len(ts) != 12 {
		t.Errorf("NewPRT() timestamp length = %d, want 12", len(ts))
	}

	// Verify the timestamp parses correctly
	_, err := time.Parse("060102150405", ts)
	if err != nil {
		t.Errorf("NewPRT() timestamp %q failed to parse: %v", ts, err)
	}
}

func TestPRTRegex(t *testing.T) {
	tests := []struct {
		input string
		match bool
	}{
		{"PRT260928200000", true},
		{"PRT261001120000", true},
		{"PRT000101000000", true},
		{"PRT123456789012", true},
		{"PRT123", false},
		{"PRT12345678901", false},
		{"PRT1234567890123", false},
		{"prt260928200000", false},
		{"XRT260928200000", false},
		{"PRT26092820000", false},
		{"PRT260928abc000", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := prtRe.MatchString(tt.input)
			if got != tt.match {
				t.Errorf("prtRe.MatchString(%q) = %v, want %v", tt.input, got, tt.match)
			}
		})
	}
}

func TestSplitExtension(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantStem  string
		wantExt   string
	}{
		{"has extension", "12345.mp4", "12345", "mp4"},
		{"double extension", "show.s01e01.mp4", "show.s01e01", "mp4"},
		{"no extension", "filename", "filename", ""},
		{"extension at start", ".hidden", ".hidden", ""},
		{"multiple dots", "a.b.c.d", "a.b.c", "d"},
		{"empty string", "", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stem, ext := splitExtension(tt.filename)
			if stem != tt.wantStem {
				t.Errorf("splitExtension(%q) stem = %q, want %q", tt.filename, stem, tt.wantStem)
			}
			if ext != tt.wantExt {
				t.Errorf("splitExtension(%q) ext = %q, want %q", tt.filename, ext, tt.wantExt)
			}
		})
	}
}
