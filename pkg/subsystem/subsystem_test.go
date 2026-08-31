package subsystem

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeSubsystem struct{ name string }

func (f fakeSubsystem) Name() string { return f.name }
func (f fakeSubsystem) Collect(_ context.Context, _ Options) ([]ComponentResult, error) {
	return nil, nil
}

func TestRegisterGetNames(t *testing.T) {
	// Use unique names so this test doesn't collide with real subsystems
	// registered by other packages' init() functions.
	Register(fakeSubsystem{name: "test-zeta"})
	Register(fakeSubsystem{name: "test-alpha"})

	s, ok := Get("test-zeta")
	require.True(t, ok)
	assert.Equal(t, "test-zeta", s.Name())

	_, ok = Get("test-does-not-exist")
	assert.False(t, ok)

	names := Names()
	assert.Contains(t, names, "test-alpha")
	assert.Contains(t, names, "test-zeta")

	// Names must be sorted.
	alphaIdx, zetaIdx := -1, -1
	for i, n := range names {
		if n == "test-alpha" {
			alphaIdx = i
		}
		if n == "test-zeta" {
			zetaIdx = i
		}
	}
	assert.Less(t, alphaIdx, zetaIdx)
}

func TestRegister_PanicsOnDuplicate(t *testing.T) {
	Register(fakeSubsystem{name: "test-dup"})
	assert.Panics(t, func() {
		Register(fakeSubsystem{name: "test-dup"})
	})
}
