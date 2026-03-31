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

func TestIsValidVersion(t *testing.T) {
	tests := []struct {
		strategy string
		version  string
		want     bool
	}{
		// Semver valid
		{"semver", "1.2.3", true},
		{"semver", "0.1.0", true},
		{"semver", "10.20.30", true},
		{"semver", "1.2.3-beta.1", true},
		// Semver invalid
		{"semver", "go-service-v1.13.0", false},
		{"semver", "deploy/go-service/prod/1.0.0", false},
		{"semver", "abc", false},
		{"semver", "", false},

		// Date-rolling valid
		{"date-rolling", "2026.03.25", true},
		{"date-rolling", "2026.03.25.2", true},
		// Date-rolling invalid
		{"date-rolling", "1.2.3", false},
		{"date-rolling", "go-service-v1.13.0", false},
		{"date-rolling", "", false},

		// Numeric-rolling valid
		{"numeric-rolling", "1", true},
		{"numeric-rolling", "42", true},
		// Numeric-rolling invalid
		{"numeric-rolling", "1.2.3", false},
		{"numeric-rolling", "abc", false},
		{"numeric-rolling", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.strategy+"/"+tt.version, func(t *testing.T) {
			got := IsValidVersion(tt.strategy, tt.version)
			if got != tt.want {
				t.Errorf("IsValidVersion(%q, %q) = %v, want %v", tt.strategy, tt.version, got, tt.want)
			}
		})
	}
}

func TestFilterTags(t *testing.T) {
	tests := []struct {
		name     string
		tags     []string
		prefix   string
		strategy string
		want     []string
	}{
		{
			name:     "semver filters out cross-contaminated tags",
			tags:     []string{"python-api-v1.3.0", "python-api-vgo-service-v1.13.0", "python-api-vdeploy/go-service/prod/1.0.0"},
			prefix:   "python-api-v",
			strategy: "semver",
			want:     []string{"python-api-v1.3.0"},
		},
		{
			name:     "semver keeps all valid tags",
			tags:     []string{"v2.0.0", "v1.1.0", "v1.0.0"},
			prefix:   "v",
			strategy: "semver",
			want:     []string{"v2.0.0", "v1.1.0", "v1.0.0"},
		},
		{
			name:     "date-rolling filters non-date versions",
			tags:     []string{"ts-spa-2026.03.31", "ts-spa-go-service-v1.13.0"},
			prefix:   "ts-spa-",
			strategy: "date-rolling",
			want:     []string{"ts-spa-2026.03.31"},
		},
		{
			name:     "numeric-rolling filters non-numeric versions",
			tags:     []string{"build-5", "build-4", "build-abc"},
			prefix:   "build-",
			strategy: "numeric-rolling",
			want:     []string{"build-5", "build-4"},
		},
		{
			name:     "returns nil when no tags match",
			tags:     []string{"go-service-v1.13.0"},
			prefix:   "python-api-v",
			strategy: "semver",
			want:     nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FilterTags(tt.tags, tt.prefix, tt.strategy)
			if len(got) != len(tt.want) {
				t.Fatalf("FilterTags() returned %d tags, want %d: %v", len(got), len(tt.want), got)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("FilterTags()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestTagPatternRegex(t *testing.T) {
	tests := []struct {
		prefix   string
		strategy string
		wantRe   string
	}{
		{"python-api-v", "semver", `python-api-v\d+\.\d+\.\d+`},
		{"v", "semver", `v\d+\.\d+\.\d+`},
		{"ts-spa-", "date-rolling", `ts-spa-\d{4}\.\d{2}\.\d{2}`},
		{"build-", "numeric-rolling", `build-\d+$`},
	}
	for _, tt := range tests {
		t.Run(tt.prefix+"/"+tt.strategy, func(t *testing.T) {
			got := TagPatternRegex(tt.prefix, tt.strategy)
			if got != tt.wantRe {
				t.Errorf("TagPatternRegex(%q, %q) = %q, want %q", tt.prefix, tt.strategy, got, tt.wantRe)
			}
		})
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
