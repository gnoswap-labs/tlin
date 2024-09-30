package fixer

import (
	"go/token"
	"os"
	"path/filepath"
	"testing"

	tt "github.com/gnolang/tlin/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const confidenceThreshold = 0.8

func TestAutoFixer(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		issues   []tt.Issue
		expected string
		dryRun   bool
	}{
		{
			name: "Fix - Simple case",
			input: `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 24},
					Suggestion: "_ = slice[:]",
					Confidence: 0.9,
				},
			},
			expected: `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:]
}
`,
		},
		{
			name: "Don't Fix - Not enough confidence",
			input: `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 24},
					Suggestion: "_ = slice[:]",
					Confidence: 0.3,
				},
			},
			expected: `package main

func main() {
	slice := []int{1, 2, 3}
	_ = slice[:len(slice)]
}
`,
		},
		{
			name: "Fix - Multiple issues",
			input: `package main

func main() {
	slice1 := []int{1, 2, 3}
	_ = slice1[:len(slice1)]

	slice2 := []string{"a", "b", "c"}
	_ = slice2[:len(slice2)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 26},
					Suggestion: "_ = slice1[:]",
					Confidence: 0.9,
				},
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 8, Column: 5},
					End:        token.Position{Line: 8, Column: 26},
					Suggestion: "_ = slice2[:]",
					Confidence: 0.9,
				},
			},
			expected: `package main

func main() {
	slice1 := []int{1, 2, 3}
	_ = slice1[:]

	slice2 := []string{"a", "b", "c"}
	_ = slice2[:]
}
`,
		},
		{
			name: "Fix - Preserve indentation",
			input: `package main

func main() {
	if true {
		slice := []int{1, 2, 3}
		_ = slice[:len(slice)]
	}
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 6, Column: 3},
					End:        token.Position{Line: 6, Column: 22},
					Suggestion: "_ = slice[:]",
					Confidence: 0.9,
				},
			},
			expected: `package main

func main() {
	if true {
		slice := []int{1, 2, 3}
		_ = slice[:]
	}
}
`,
		},
		{
			name: "DryRun",
			input: `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`,
			issues: []tt.Issue{
				{
					Rule:       "simplify-slice-range",
					Message:    "unnecessary use of len() in slice expression, can be simplified",
					Start:      token.Position{Line: 5, Column: 5},
					End:        token.Position{Line: 5, Column: 24},
					Suggestion: "_ = slice[:]",
					Confidence: 0.9,
				},
			},
			expected: `package main

func main() {
    slice := []int{1, 2, 3}
    _ = slice[:len(slice)]
}`,
			dryRun: true,
		},
		{
			name: "FixIssues - Emit function formatting",
			input: `package main

import "std"

func main() {
    newOwner := "Alice"
    oldOwner := "Bob"
    std.Emit("OwnershipChange",
	"newOwner", newOwner, "oldOwner", oldOwner)
}`,
			issues: []tt.Issue{
				{
					Rule:    "emit-format",
					Message: "Consider formatting std.Emit call for better readability",
					Start:   token.Position{Line: 8, Column: 5},
					End:     token.Position{Line: 9, Column: 44},
					Suggestion: `std.Emit(
    "OwnershipChange",
    "newOwner", newOwner,
    "oldOwner", oldOwner,
)`,
					Confidence: 0.9,
				},
			},
			expected: `package main

import "std"

func main() {
	newOwner := "Alice"
	oldOwner := "Bob"
	std.Emit(
		"OwnershipChange",
		"newOwner", newOwner,
		"oldOwner", oldOwner,
	)
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runTestCase(t, tt.input, tt.issues, tt.expected, tt.dryRun)
		})
	}
}

func runTestCase(t *testing.T, input string, issues []tt.Issue, expected string, dryRun bool) {
	t.Helper()
	_, testFile, cleanup := setupTestFile(t, input)
	defer cleanup()

	for i := range issues {
		issues[i].Filename = testFile
	}

	fixer := New(dryRun, confidenceThreshold)
	err := fixer.Fix(testFile, issues)
	require.NoError(t, err)

	content, err := os.ReadFile(testFile)
	require.NoError(t, err)

	assert.Equal(t, expected, string(content))
}

func setupTestFile(t *testing.T, content string) (string, string, func()) {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "autofixer-test")
	require.NoError(t, err)

	testFile := filepath.Join(tmpDir, "test.go")
	err = os.WriteFile(testFile, []byte(content), 0644)
	require.NoError(t, err)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, testFile, cleanup
}
