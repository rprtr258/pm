package daemon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFilepathJoin(t *testing.T) {
	t.Parallel()
	assert.Equal(t,
		`/home/user/dir`,
		getProcCwd("/home/user/dir", "/home/user/dir"))
}
