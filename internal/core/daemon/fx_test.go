package daemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
)

func TestNewApp(t *testing.T) {
	t.Parallel()

	assert.NoError(t, fx.ValidateApp(NewApp()))
}
