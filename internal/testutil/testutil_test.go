package testutil

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_TestDependency(t *testing.T) {
	t.Run("all have names", func(t *testing.T) {
		for dependency := TestDependency(0); dependency < invalidDependency; dependency++ {
			require.NotEmptyf(t, dependencyNames[dependency], "TestDependency(%d) has no name", dependency)
		}
	})

	t.Run("all dependencies have skip vars", func(t *testing.T) {
		for dependency := TestDependency(0); dependency < invalidDependency; dependency++ {
			require.NotEmptyf(t, dependencySkipVars[dependency], "%s is missing from dependencySkipVars", dependency)
		}
	})
}
