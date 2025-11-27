package convert

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

// MarkdownToOrg converts markdown content to org-mode
func MarkdownToOrg(mdContent string, idMap map[string]string) (string, error) {
	lines := strings.Split(mdContent, "\n")

	// Extract YAML front matter and convert to properties
	properties, bodyLines := extractYAMLFromLines(lines)

	var org strings.Builder

	// Write properties if present
	if properties != "" {
		org.WriteString(properties)
	}

	inCodeBlock := false
	inQuoteBlock := false
	codeBlockLang := ""

	for i := 0; i < len(bodyLines); i++ {
		line := bodyLines[i]
		trimmed := strings.TrimSpace(line)

		// Skip emoji date lines and priority lines (already processed as task metadata)
		if strings.HasPrefix(trimmed, "⏳ ") || strings.HasPrefix(trimmed, "📅 ") ||
		   strings.HasPrefix(trimmed, "✅ ") || strings.HasPrefix(trimmed, "Priority: ") {
			continue
		}

		// Handle code blocks
		if strings.HasPrefix(trimmed, "```") {
			if !inCodeBlock {
				// Starting code block
				inCodeBlock = true
				codeBlockLang = strings.TrimSpace(strings.TrimPrefix(trimmed, "```"))
				org.WriteString("#+BEGIN_SRC " + codeBlockLang + "\n")
			} else {
				// Ending code block
				inCodeBlock = false
				codeBlockLang = ""
				org.WriteString("#+END_SRC\n")
			}
			continue
		}
		if inCodeBlock {
			org.WriteString(line + "\n")
			continue
		}

		// Handle blockquotes
		if strings.HasPrefix(trimmed, ">") {
			if !inQuoteBlock {
				org.WriteString("#+BEGIN_QUOTE\n")
				inQuoteBlock = true
			}
			// Remove the > and leading space
			quoteContent := strings.TrimSpace(strings.TrimPrefix(trimmed, ">"))
			org.WriteString(quoteContent + "\n")
			// Check if next line is also a quote
			if i+1 < len(bodyLines) {
				nextTrimmed := strings.TrimSpace(bodyLines[i+1])
				if !strings.HasPrefix(nextTrimmed, ">") {
					org.WriteString("#+END_QUOTE\n")
					inQuoteBlock = false
				}
			} else {
				org.WriteString("#+END_QUOTE\n")
				inQuoteBlock = false
			}
			continue
		}

		// Handle headers (including tasks)
		if strings.HasPrefix(trimmed, "#") {
			hashes := countLeadingChars(trimmed, '#')
			if hashes > 0 {
				rest := strings.TrimSpace(trimmed[hashes:])

				// Check if this is a task header (## - [ ] or ## - [x])
				isTask := false
				isDone := false
				taskContent := ""

				if strings.HasPrefix(rest, "- [ ] ") {
					isTask = true
					isDone = false
					taskContent = strings.TrimSpace(rest[6:])
				} else if strings.HasPrefix(rest, "- [x] ") {
					isTask = true
					isDone = true
					taskContent = strings.TrimSpace(rest[6:])
				}

				stars := strings.Repeat("*", hashes)

				if isTask {
					// Task header
					status := "TODO"
					if isDone {
						status = "DONE"
					}

					// Look ahead for scheduling info and priority
					var scheduledDate, deadlineDate, closedDate, priority string
					j := i + 1
					for j < len(bodyLines) {
						nextLine := strings.TrimSpace(bodyLines[j])
						if strings.HasPrefix(nextLine, "⏳ ") {
							scheduledDate = strings.TrimSpace(strings.TrimPrefix(nextLine, "⏳"))
							j++
						} else if strings.HasPrefix(nextLine, "📅 ") {
							deadlineDate = strings.TrimSpace(strings.TrimPrefix(nextLine, "📅"))
							j++
						} else if strings.HasPrefix(nextLine, "✅ ") {
							closedDate = strings.TrimSpace(strings.TrimPrefix(nextLine, "✅"))
							j++
						} else if strings.HasPrefix(nextLine, "Priority: ") {
							priorityLevel := strings.TrimSpace(strings.TrimPrefix(nextLine, "Priority:"))
							if priorityLevel == "high" {
								priority = "A"
							} else if priorityLevel == "medium" {
								priority = "B"
							} else if priorityLevel == "low" {
								priority = "C"
							}
							j++
						} else {
							break
						}
					}

					// Write task header
					org.WriteString(stars + " " + status + " ")
					if priority != "" {
						org.WriteString("[#" + priority + "] ")
					}
					org.WriteString(taskContent + "\n")

					// Write scheduling info
					if scheduledDate != "" {
						org.WriteString("SCHEDULED: <" + scheduledDate + ">\n")
					}
					if deadlineDate != "" {
						org.WriteString("DEADLINE: <" + deadlineDate + ">\n")
					}
					if closedDate != "" {
						org.WriteString("CLOSED: [" + closedDate + "]\n")
					}
				} else {
					// Regular header
					org.WriteString(stars + " " + rest + "\n")
				}
				continue
			}
		}

		// Convert wikilinks in regular content
		convertedLine := convertMarkdownLinks(line, idMap)

		// Write the line
		org.WriteString(convertedLine + "\n")
	}

	return strings.TrimSpace(org.String()), nil
}

// extractYAMLFromLines extracts YAML front matter and returns properties + body lines
func extractYAMLFromLines(lines []string) (string, []string) {
	var properties strings.Builder
	var bodyLines []string

	// Check for front matter delimiters
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return "", lines
	}

	// Find end of front matter
	frontMatterEnd := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			frontMatterEnd = i
			break
		}
	}

	if frontMatterEnd == -1 {
		return "", lines
	}

	// Parse YAML front matter
	yamlContent := strings.Join(lines[1:frontMatterEnd], "\n")

	var frontMatter struct {
		ID      string   `yaml:"id"`
		Title   string   `yaml:"title"`
		Aliases []string `yaml:"aliases"`
		Tags    []string `yaml:"tags"`
	}

	if err := yaml.Unmarshal([]byte(yamlContent), &frontMatter); err != nil {
		// If YAML parsing fails, fall back to empty
		return "", lines
	}

	// Extract body lines (skip front matter)
	for i := frontMatterEnd + 1; i < len(lines); i++ {
		bodyLines = append(bodyLines, lines[i])
	}

	// Skip leading blank lines in body
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
		bodyLines = bodyLines[1:]
	}

	// Build properties drawer
	if frontMatter.ID != "" || len(frontMatter.Aliases) > 0 {
		properties.WriteString(":PROPERTIES:\n")
		if frontMatter.ID != "" {
			properties.WriteString(":ID: " + frontMatter.ID + "\n")
		}
		if len(frontMatter.Aliases) > 0 {
			aliasStr := ""
			for i, alias := range frontMatter.Aliases {
				if i > 0 {
					aliasStr += " "
				}
				aliasStr += fmt.Sprintf(`"%s"`, alias)
			}
			properties.WriteString(":ROAM_ALIASES: " + aliasStr + "\n")
		}
		properties.WriteString(":END:\n")
	}

	// Add title
	if frontMatter.Title != "" {
		properties.WriteString("#+title: " + frontMatter.Title + "\n")
	}

	// Add tags
	if len(frontMatter.Tags) > 0 {
		tagStr := ":" + strings.Join(frontMatter.Tags, ":") + ":"
		properties.WriteString("#+filetags: " + tagStr + "\n")
	}

	// Add blank line after properties
	if properties.Len() > 0 {
		properties.WriteString("\n")
	}

	return properties.String(), bodyLines
}

// convertMarkdownLinks converts wikilinks to org-roam links
func convertMarkdownLinks(line string, idMap map[string]string) string {
	// Pattern: [[filename|description]] or [[filename]]
	re := regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

	// Create reverse map (filename -> id)
	reverseMap := make(map[string]string)
	for id, filename := range idMap {
		reverseMap[filename] = id
	}

	return re.ReplaceAllStringFunc(line, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		filename := submatches[1]
		description := ""
		if len(submatches) > 2 && submatches[2] != "" {
			description = submatches[2]
		}

		// Look up ID from filename
		uuid, ok := reverseMap[filename]
		if !ok {
			// Filename not in map, check if it's already a UUID
			if isUUID(filename) {
				uuid = filename
			} else {
				// Generate new UUID
				uuid = GenerateOrgID()
			}
		}

		// Build org-roam link
		if description != "" {
			return fmt.Sprintf("[[id:%s][%s]]", uuid, description)
		}
		return fmt.Sprintf("[[id:%s]]", uuid)
	})
}

// isUUID checks if a string looks like a UUID
func isUUID(s string) bool {
	return len(s) == 36 && strings.Count(s, "-") == 4
}

// ConvertMarkdownHeader converts markdown header to org-mode
// # Header → * Header
// ## Subheading → ** Subheading
func ConvertMarkdownHeader(line string) string {
	trimmed := strings.TrimSpace(line)
	hashes := countLeadingChars(trimmed, '#')

	if hashes == 0 {
		return line
	}

	// Extract content after hashes
	content := strings.TrimSpace(trimmed[hashes:])

	// Build org header
	return strings.Repeat("*", hashes) + " " + content
}

// ConvertWikilink converts Obsidian wikilink to org-roam link
// [[filename|Description]] → [[id:uuid][Description]]
// [[filename]] → [[id:uuid]]
func ConvertWikilink(link string, idMap map[string]string) string {
	return convertMarkdownLinks(link, idMap)
}

// ConvertMarkdownTask converts markdown checkbox to org-mode task
// - [ ] Task → * TODO Task
// - [x] Task → * DONE Task
func ConvertMarkdownTask(line string) string {
	trimmed := strings.TrimSpace(line)

	// Check for task at header level (## - [ ])
	hashes := countLeadingChars(trimmed, '#')
	if hashes > 0 {
		rest := strings.TrimSpace(trimmed[hashes:])
		stars := strings.Repeat("*", hashes)

		if strings.HasPrefix(rest, "- [ ] ") {
			content := strings.TrimSpace(rest[6:])
			return stars + " TODO " + content
		} else if strings.HasPrefix(rest, "- [x] ") {
			content := strings.TrimSpace(rest[6:])
			return stars + " DONE " + content
		}
	}

	// Check for standalone task (- [ ])
	if strings.HasPrefix(trimmed, "- [ ] ") {
		content := strings.TrimSpace(trimmed[6:])
		return "* TODO " + content
	} else if strings.HasPrefix(trimmed, "- [x] ") {
		content := strings.TrimSpace(trimmed[6:])
		return "* DONE " + content
	}

	return line
}

// ExtractYAMLFrontMatter extracts YAML front matter and converts to properties drawer
func ExtractYAMLFrontMatter(content string) (properties string, bodyContent string) {
	lines := strings.Split(content, "\n")
	props, body := extractYAMLFromLines(lines)
	return props, strings.Join(body, "\n")
}

// GenerateOrgID generates a new org-mode ID (UUID v4)
func GenerateOrgID() string {
	return uuid.New().String()
}
