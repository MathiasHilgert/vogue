package rules

import (
	"slices"
	"testing"
	"time"
	_ "time/tzdata" // the zones must load without a system database too.

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZones(t *testing.T) {
	t.Parallel()

	t.Run("are sorted, which the binary search relies on", func(t *testing.T) {
		t.Parallel()

		// Assert
		assert.True(t, slices.IsSorted(timeZoneNames[:]))
	})

	t.Run("each loads under the very name it is listed as", func(t *testing.T) {
		t.Parallel()

		for _, name := range timeZoneNames {
			// Act
			location, err := time.LoadLocation(name)

			// Assert
			require.NoError(t, err, name)
			assert.Equal(t, name, location.String())
		}
	})
}
