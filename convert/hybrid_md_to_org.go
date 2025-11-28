package convert

import (
	"strings"
)

// HybridMarkdownToOrg converts markdown to org-mode using hybrid annotation pattern
func HybridMarkdownToOrg(mdContent string, idMap map[string]string) (string, error) {
	converter := NewHybridConverter(idMap)
	defer converter.reset()

	// Step 1: Extract custom features and replace with markers
	marked := mdContent

	// Extract wikilinks
	marked = converter.extractWikilinks(marked)

	// Step 2: Use existing converter for standard conversion
	// Note: goldmark doesn't have an org-mode writer, so we use our existing parser
	// In the future, we could write a custom goldmark renderer that outputs org-mode
	converted, err := MarkdownToOrg(marked, idMap)
	if err != nil {
		return "", err
	}

	// Step 3: Replace markers with converted features
	final := converter.applyMarkers(converted)

	return strings.TrimSpace(final), nil
}
