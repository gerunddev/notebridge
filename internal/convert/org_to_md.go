package convert

import (
	"strings"
)

// OrgToMarkdown converts org-mode content to markdown
func OrgToMarkdown(orgContent string, idMap map[string]string) (string, error) {
	var md strings.Builder

	// TODO: Implement org-mode to markdown conversion
	// This will handle:
	// - Headers: * → #
	// - Properties drawer → YAML front matter
	// - Links: [[id:uuid][desc]] → [[filename|desc]]
	// - Tasks: * TODO → - [ ]
	// - Code blocks: #+BEGIN_SRC → ```
	// - Tags: :tag1:tag2: → front matter
	// - Scheduled/Deadline dates
	// - Priorities
	// - Embeds
	// - Quotes: #+BEGIN_QUOTE → >

	return md.String(), nil
}

// ConvertOrgHeader converts org-mode header to markdown
// * Header → # Header
// ** Subheading → ## Subheading
func ConvertOrgHeader(line string) string {
	// Count asterisks
	stars := 0
	for _, c := range line {
		if c == '*' {
			stars++
		} else if c == ' ' {
			break
		}
	}

	if stars == 0 {
		return line
	}

	// Extract content after asterisks
	content := strings.TrimSpace(line[stars:])

	// Build markdown header
	return strings.Repeat("#", stars) + " " + content
}

// ConvertOrgLink converts org-roam link to Obsidian wikilink
// [[id:uuid][Description]] → [[filename|Description]]
// [[id:uuid]] → [[filename]]
func ConvertOrgLink(link string, idMap map[string]string) string {
	// TODO: Implement link conversion using ID map
	return link
}

// ConvertOrgTask converts org-mode task to markdown checkbox
// * TODO Task → - [ ] Task
// * DONE Task → - [x] Task
func ConvertOrgTask(line string) string {
	// TODO: Implement task conversion
	return line
}

// ExtractOrgProperties extracts properties drawer and converts to YAML front matter
func ExtractOrgProperties(content string) (frontMatter string, bodyContent string) {
	// TODO: Parse :PROPERTIES: drawer and convert to YAML
	return "", content
}
