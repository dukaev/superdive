package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCleanArgs(t *testing.T) {
	t.Run("basic trimming", func(t *testing.T) {
		input := []string{"  hello  ", "  world", "test  "}
		expected := []string{"hello", "world", "test"}
		result := CleanArgs(input)
		assert.Equal(t, expected, result)
	})

	t.Run("empty strings are removed", func(t *testing.T) {
		input := []string{"hello", "", "world", ""}
		expected := []string{"hello", "world"}
		result := CleanArgs(input)
		assert.Equal(t, expected, result)
	})

	t.Run("whitespace only strings are kept but trimmed", func(t *testing.T) {
		// Note: CleanArgs only removes empty strings (""), not whitespace-only strings
		// It only trims spaces, not tabs
		input := []string{"  ", "   ", "hello", "   "}
		expected := []string{"", "", "hello", ""}
		result := CleanArgs(input)
		assert.Equal(t, expected, result)
	})

	t.Run("empty slice", func(t *testing.T) {
		input := []string{}
		var expected []string // nil
		result := CleanArgs(input)
		assert.Equal(t, expected, result)
	})

	t.Run("nil slice", func(t *testing.T) {
		var input []string
		var expected []string // nil
		result := CleanArgs(input)
		assert.Equal(t, expected, result)
	})

	t.Run("all empty strings", func(t *testing.T) {
		input := []string{"", "", ""}
		var expected []string // nil - all empty strings are removed
		result := CleanArgs(input)
		assert.Equal(t, expected, result)
	})

	t.Run("no trimming needed", func(t *testing.T) {
		input := []string{"hello", "world"}
		expected := []string{"hello", "world"}
		result := CleanArgs(input)
		assert.Equal(t, expected, result)
	})

	t.Run("only spaces are trimmed, not tabs", func(t *testing.T) {
		// Note: CleanArgs only trims spaces (" "), not tabs or other whitespace
		input := []string{"  hello  ", "  world  ", "\thello\t"}
		expected := []string{"hello", "world", "\thello\t"}
		result := CleanArgs(input)
		assert.Equal(t, expected, result)
	})
}
