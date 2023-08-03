package w3bstreamclient

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHello(t *testing.T) {
	require := require.New(t)
	c := &Client{}
	err := c.hello()
	require.NoError(err)
}
