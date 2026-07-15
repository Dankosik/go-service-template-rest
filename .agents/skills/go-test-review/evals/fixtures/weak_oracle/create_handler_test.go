package create

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreate(t *testing.T) {
	store := &fakeStore{}
	handler := newHandler(store)

	response, err := handler.Create(validRequest())

	require.NoError(t, err)
	require.NotNil(t, response)
}
