package strategy

import (
	"testing"
	"time"

	"github.com/dnd-it/action-releaser/internal/config"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"semver", false},
		{"date-rolling", false},
		{"numeric-rolling", false},
		{"unknown", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := New(tt.name)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if s.Name() != tt.name {
				t.Errorf("Name() = %q, want %q", s.Name(), tt.name)
			}
		})
	}
}

func TestDateRolling_NextVersion(t *testing.T) {
	fixedTime := time.Date(2026, 3, 25, 14, 0, 0, 0, time.UTC)
	cfg := config.Config{TagPrefix: "v"}

	tests := []struct {
		name    string
		tags    []string
		want    string
		wantPre string
	}{
		{
			name: "first release of the day",
			tags: []string{"v2026.03.24"},
			want: "2026.03.25",
			wantPre: "v2026.03.24",
		},
		{
			name: "second release same day",
			tags: []string{"v2026.03.25", "v2026.03.24"},
			want: "2026.03.25.2",
			wantPre: "v2026.03.25",
		},
		{
			name: "third release same day",
			tags: []string{"v2026.03.25.2", "v2026.03.25", "v2026.03.24"},
			want: "2026.03.25.3",
			wantPre: "v2026.03.25.2",
		},
		{
			name:    "bootstrap - no tags",
			tags:    nil,
			want:    "2026.03.25",
			wantPre: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := &DateRolling{Now: func() time.Time { return fixedTime }}
			result, err := d.NextVersion(tt.tags, cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Skipped {
				t.Fatal("unexpected skip")
			}
			if result.Version != tt.want {
				t.Errorf("Version = %q, want %q", result.Version, tt.want)
			}
			if result.PreviousVersion != tt.wantPre {
				t.Errorf("PreviousVersion = %q, want %q", result.PreviousVersion, tt.wantPre)
			}
		})
	}
}

func TestDateRolling_AlwaysReleases(t *testing.T) {
	d := &DateRolling{}
	if !d.AlwaysReleases() {
		t.Error("DateRolling should always release")
	}
}

func TestNumericRolling_NextVersion(t *testing.T) {
	cfg := config.Config{TagPrefix: "v"}

	tests := []struct {
		name    string
		tags    []string
		want    string
		wantPre string
	}{
		{
			name:    "bootstrap - no tags",
			tags:    nil,
			want:    "1",
			wantPre: "",
		},
		{
			name:    "increment from 1",
			tags:    []string{"v1"},
			want:    "2",
			wantPre: "v1",
		},
		{
			name:    "increment from 42",
			tags:    []string{"v42", "v41"},
			want:    "43",
			wantPre: "v42",
		},
		{
			name:    "skip non-numeric tags",
			tags:    []string{"v1.0.0", "v5", "v3", "latest"},
			want:    "6",
			wantPre: "v5",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &NumericRolling{}
			result, err := n.NextVersion(tt.tags, cfg)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Skipped {
				t.Fatal("unexpected skip")
			}
			if result.Version != tt.want {
				t.Errorf("Version = %q, want %q", result.Version, tt.want)
			}
			if result.PreviousVersion != tt.wantPre {
				t.Errorf("PreviousVersion = %q, want %q", result.PreviousVersion, tt.wantPre)
			}
		})
	}
}

func TestNumericRolling_AlwaysReleases(t *testing.T) {
	n := &NumericRolling{}
	if !n.AlwaysReleases() {
		t.Error("NumericRolling should always release")
	}
}

func TestSemver_AlwaysReleases(t *testing.T) {
	s := &Semver{}
	if s.AlwaysReleases() {
		t.Error("Semver should NOT always release")
	}
}

func TestParseDeleteVersion(t *testing.T) {
	tests := []struct {
		input       string
		wantDate    string
		wantCounter int
		wantErr     bool
	}{
		{"2026.03.25", "2026.03.25", 0, false},
		{"2026.03.25.2", "2026.03.25", 2, false},
		{"2026.03.25.10", "2026.03.25", 10, false},
		{"bad", "", 0, true},
		{"2026.03.25.abc", "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			date, counter, err := ParseDateVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if date != tt.wantDate {
				t.Errorf("date = %q, want %q", date, tt.wantDate)
			}
			if counter != tt.wantCounter {
				t.Errorf("counter = %d, want %d", counter, tt.wantCounter)
			}
		})
	}
}
