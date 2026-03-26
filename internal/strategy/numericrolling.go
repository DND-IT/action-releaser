package strategy

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/dnd-it/action-releaser/internal/config"
)

// NumericRolling implements VersionStrategy with simple incrementing numbers.
type NumericRolling struct{}

func (n *NumericRolling) Name() string        { return "numeric-rolling" }
func (n *NumericRolling) AlwaysReleases() bool { return true }

func (n *NumericRolling) NextVersion(tags []string, cfg config.Config) (Result, error) {
	prefix := cfg.TagPrefix

	// Find the highest numeric tag.
	var highest int
	var latestTag string
	for _, t := range tags {
		v := strings.TrimPrefix(t, prefix)
		num, err := strconv.Atoi(v)
		if err != nil {
			continue // Skip non-numeric tags.
		}
		if num > highest {
			highest = num
			latestTag = t
		}
	}

	next := highest + 1

	// Guard against overflow (theoretical, but defensive).
	if next < highest {
		return Result{}, fmt.Errorf("numeric version overflow at %d", highest)
	}

	return Result{
		Version:         fmt.Sprintf("%d", next),
		PreviousVersion: latestTag,
	}, nil
}
