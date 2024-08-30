package internal

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tt "github.com/gnoswap-labs/tlin/internal/types"
)

func TestCache(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cacheDir := filepath.Join(tmpDir, "cache")
	cache, err := NewCache(cacheDir)
	require.NoError(t, err)

	t.Run("SaveAndLoad", func(t *testing.T) {
		issues := []tt.Issue{
			{
				Rule:     "test-rule",
				Category: "test-category",
				Filename: "test.go",
				Message:  "test issue",
				Start:    token.Position{Line: 10, Column: 1, Filename: "test.go"},
				End:      token.Position{Line: 10, Column: 10, Filename: "test.go"},
			},
		}

		filename := filepath.Join(tmpDir, "test.go")
		err := os.WriteFile(filename, []byte("package main\n\nfunc main() {}\n"), 0644)
		require.NoError(t, err)

		err = cache.Set(filename, issues)
		assert.NoError(t, err)

		loadedIssues, found := cache.Get(filename)
		assert.True(t, found)
		assert.Equal(t, issues, loadedIssues)
	})

	t.Run("NotFound", func(t *testing.T) {
		_, found := cache.Get("nonexistent.go")
		assert.False(t, found)
	})

	t.Run("FileModified", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "modified.go")
		err := os.WriteFile(filename, []byte("package main\n\nfunc main() {}\n"), 0644)
		require.NoError(t, err)

		issues := []tt.Issue{
			{
				Rule:     "test-rule",
				Category: "test-category",
				Filename: filename,
				Message:  "test issue",
				Start:    token.Position{Line: 1, Column: 1, Filename: filename},
				End:      token.Position{Line: 1, Column: 10, Filename: filename},
			},
		}

		err = cache.Set(filename, issues)
		assert.NoError(t, err)

		// modify file
		time.Sleep(time.Second) // ensure file modification time is different
		err = os.WriteFile(filename, []byte("package main\n\nfunc main() { println(\"Hello\") }\n"), 0644)
		require.NoError(t, err)

		_, found := cache.Get(filename)
		assert.False(t, found)
	})
}

func TestCacheWithEngine(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cache-engine-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cacheDir := filepath.Join(tmpDir, "cache")
	engine, err := NewEngine(tmpDir, cacheDir)
	require.NoError(t, err)

	t.Run("CacheHit", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "test.go")
		content := []byte(`package main

func main() {
    var x int
    x = x
}`)
		err = os.WriteFile(filename, content, 0644)
		require.NoError(t, err)

		// First run
		issues, err := engine.Run(filename)
		require.NoError(t, err)
		assert.NotEmpty(t, issues) // must contain self-assigned variable issue

		// Second run (should hit cache)
		cachedIssues, err := engine.Run(filename)
		require.NoError(t, err)
		assert.Equal(t, issues, cachedIssues)
	})

	t.Run("CacheMiss", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "test2.go")
		content := []byte(`package main

func main() {
    var x int
    x = x
}`)
		err = os.WriteFile(filename, content, 0644)
		require.NoError(t, err)

		// First run
		issues, err := engine.Run(filename)
		require.NoError(t, err)
		assert.NotEmpty(t, issues)

		// Modify file
		time.Sleep(time.Second) // Ensure file modification time is different
		newContent := []byte(`package main

func main() {
    var x int
    y := x
    _ = y
}`)
		err = os.WriteFile(filename, newContent, 0644)
		require.NoError(t, err)

		// Second run (should miss cache due to file modification)
		newIssues, err := engine.Run(filename)
		require.NoError(t, err)
		assert.NotEqual(t, issues, newIssues)
	})
}
