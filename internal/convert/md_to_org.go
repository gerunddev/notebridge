package convert

import (
	"strings"
)

// MarkdownToOrg converts markdown content to org-mode
func MarkdownToOrg(mdContent string, idMap map[string]string) (string, error) {
	var org strings.Builder

	// TODO: Implement markdown to org-mode conversion
	// This will handle:
	// - Headers: # → *
	// - YAML front matter → Properties drawer
	// - Links: [[filename|desc]] → [[id:uuid][desc]]
	// - Tasks: - [ ] → * TODO
	// - Code blocks: ``` → #+BEGIN_SRC
	// - Front matter tags → :tag1:tag2:
	// - Obsidian dates → Scheduled/Deadline
	// - Priorities
	// - Embeds
	// - Blockquotes: > → #+BEGIN_QUOTE

	return org.String(), nil
}

// ConvertMarkdownHeader converts markdown header to org-mode
// # Header → * Header
// ## Subheading → ** Subheading
func ConvertMarkdownHeader(line string) string {
	// Count hash symbols
	hashes := 0
	for _, c := range line {
		if c == '#' {
			hashes++
		} else if c == ' ' {
			break
		}
	}

	if hashes == 0 {
		return line
	}

	// Extract content after hashes
	content := strings.TrimSpace(line[hashes:])

	// Build org header
	return strings.Repeat("*", hashes) + " " + content
}

// ConvertWikilink converts Obsidian wikilink to org-roam link
// [[filename|Description]] → [[id:uuid][Description]]
// [[filename]] → [[id:uuid][filename]]
func ConvertWikilink(link string, idMap map[string]string) string {
	// TODO: Implement wikilink conversion
	// Need to look up filename in idMap (reverse lookup)
	// or generate new UUID if not found
	return link
}

// ConvertMarkdownTask converts markdown checkbox to org-mode task
// - [ ] Task → * TODO Task
// - [x] Task → * DONE Task
func ConvertMarkdownTask(line string) string {
	// TODO: Implement task conversion
	return line
}

// ExtractYAMLFrontMatter extracts YAML front matter and converts to properties drawer
func ExtractYAMLFrontMatter(content string) (properties string, bodyContent string) {
	// TODO: Parse YAML front matter and convert to :PROPERTIES: drawer
	return "", content
}

// GenerateOrgID generates a new org-mode ID (UUID)
func GenerateOrgID() string {
	// TODO: Generate UUID for new org-roam notes
	return ""
}
