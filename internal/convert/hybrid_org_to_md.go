package convert

import (
	"strings"
)

// HybridOrgToMarkdown converts org-mode to markdown using hybrid annotation pattern
func HybridOrgToMarkdown(orgContent string, idMap map[string]string) (string, error) {
	converter := NewHybridConverter(idMap)
	defer converter.reset()

	// Step 1: Extract custom features and replace with markers
	marked := orgContent

	// Extract org-roam ID links
	marked = converter.extractOrgRoamLinks(marked)

	// Step 2: Use existing converter for standard conversion
	// Note: go-org library doesn't have a markdown writer, so we use our existing converter
	// In the future, we could implement a custom markdown renderer that works with go-org AST
	converted, err := OrgToMarkdown(marked, idMap)
	if err != nil {
		return "", err
	}

	// Step 3: Replace markers with converted features
	final := converter.applyMarkers(converted)

	return strings.TrimSpace(final), nil
}
