package tools

import (
	"fmt"
	"github.com/pmezard/go-difflib/difflib"
)

// GenerateDiff creates a unified diff given old and new content.
func GenerateDiff(path, oldContent, newContent string) string {
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(oldContent),
		B:        difflib.SplitLines(newContent),
		FromFile: path,
		ToFile:   path,
		Context:  3,
	}
	text, _ := difflib.GetUnifiedDiffString(diff)
	if text == "" {
		return fmt.Sprintf("No changes in %s\n", path)
	}
	return text
}
