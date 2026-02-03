package utils

import (
	"errors"
	"github.com/awesome-gocui/gocui"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsNewView(t *testing.T) {
	t.Run("all ErrUnknownView returns true", func(t *testing.T) {
		errs := []error{gocui.ErrUnknownView, gocui.ErrUnknownView}
		result := IsNewView(errs...)
		assert.True(t, result)
	})

	t.Run("single ErrUnknownView returns true", func(t *testing.T) {
		result := IsNewView(gocui.ErrUnknownView)
		assert.True(t, result)
	})

	t.Run("nil error returns false", func(t *testing.T) {
		result := IsNewView(nil)
		assert.False(t, result)
	})

	t.Run("mix of nil and ErrUnknownView returns false", func(t *testing.T) {
		errs := []error{nil, gocui.ErrUnknownView}
		result := IsNewView(errs...)
		assert.False(t, result)
	})

	t.Run("other error returns true", func(t *testing.T) {
		customErr := errors.New("custom error")
		result := IsNewView(customErr)
		assert.True(t, result)
	})

	t.Run("mix of errors returns true", func(t *testing.T) {
		errs := []error{gocui.ErrUnknownView, errors.New("custom error")}
		result := IsNewView(errs...)
		assert.True(t, result)
	})

	t.Run("no errors returns true", func(t *testing.T) {
		result := IsNewView()
		assert.True(t, result)
	})
}
