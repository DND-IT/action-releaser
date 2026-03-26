package strategy

import (
	"fmt"

	"github.com/dnd-it/action-releaser/internal/config"
)

// Result holds the outcome of a version calculation.
type Result struct {
	Version         string // Next version (without tag prefix). Empty if skipped.
	PreviousVersion string // Previous version tag. Empty if first release.
	Skipped         bool   // True if no release is needed.
}

// VersionStrategy determines the next version based on existing tags and commits.
type VersionStrategy interface {
	// NextVersion calculates the next version.
	// Returns a Result with Skipped=true if no release is needed.
	NextVersion(tags []string, cfg config.Config) (Result, error)

	// AlwaysReleases returns true if this strategy always produces a release
	// (date-rolling, numeric-rolling) vs. conditionally (semver).
	AlwaysReleases() bool

	// Name returns the strategy name for logging.
	Name() string
}

// New creates a VersionStrategy for the given strategy name.
func New(name string) (VersionStrategy, error) {
	switch name {
	case "semver":
		return &Semver{}, nil
	case "date-rolling":
		return &DateRolling{}, nil
	case "numeric-rolling":
		return &NumericRolling{}, nil
	default:
		return nil, fmt.Errorf("unknown strategy %q: use semver, date-rolling, or numeric-rolling", name)
	}
}
