package formatter

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/gnoswap-labs/lint/internal"
	tt "github.com/gnoswap-labs/lint/internal/types"
)

// rule set
const (
	UnnecessaryElse     = "unnecessary-else"
	UnnecessaryTypeConv = "unnecessary-type-conversion"
	SimplifySliceExpr   = "simplify-slice-range"
	CycloComplexity     = "high-cyclomatic-complexity"
)

// IssueFormatter is the interface that wraps the Format method.
// Implementations of this interface are responsible for formatting specific types of lint issues.
type IssueFormatter interface {
	Format(issue tt.Issue, snippet *internal.SourceCode) string
}

// FormatIssuesWithArrows formats a slice of issues into a human-readable string.
// It uses the appropriate formatter for each issue based on its rule.
func FormatIssuesWithArrows(issues []tt.Issue, snippet *internal.SourceCode) string {
	var builder strings.Builder
	for _, issue := range issues {
		builder.WriteString(formatIssueHeader(issue))
		formatter := getFormatter(issue.Rule)
		builder.WriteString(formatter.Format(issue, snippet))
	}
	return builder.String()
}

// getFormatter is a factory function that returns the appropriate IssueFormatter
// based on the given rule.
// If no specific formatter is found for the given rule, it returns a GeneralIssueFormatter.
func getFormatter(rule string) IssueFormatter {
	switch rule {
	case UnnecessaryElse:
		return &UnnecessaryElseFormatter{}
	case SimplifySliceExpr:
		return &SimplifySliceExpressionFormatter{}
	case UnnecessaryTypeConv:
		return &UnnecessaryTypeConversionFormatter{}
	case CycloComplexity:
		return &CyclomaticComplexityFormatter{}
	default:
		return &GeneralIssueFormatter{}
	}
}

// formatIssueHeader creates a formatted header string for a given issue.
// The header includes the rule and the filename. (e.g. "error: unused-variable\n --> test.go")
func formatIssueHeader(issue tt.Issue) string {
	return errorStyle.Sprint("error: ") + ruleStyle.Sprint(issue.Rule) + "\n" +
		lineStyle.Sprint(" --> ") + fileStyle.Sprint(issue.Filename) + "\n"
}

func buildSuggestion(result *strings.Builder, issue tt.Issue, lineStyle, suggestionStyle *color.Color) {
	result.WriteString(suggestionStyle.Sprintf("Suggestion:\n"))
	result.WriteString(lineStyle.Sprintf("%d | ", issue.Start.Line))
	result.WriteString(fmt.Sprintf("%s\n", issue.Suggestion))
	result.WriteString("\n")
}

func buildNote(result *strings.Builder, issue tt.Issue, suggestionStyle *color.Color) {
	result.WriteString(suggestionStyle.Sprint("Note: "))
	result.WriteString(fmt.Sprintf("%s\n", issue.Note))
	result.WriteString("\n")
}
