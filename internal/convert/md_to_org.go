package convert

import (
	"crypto/rand"
	"fmt"
	"regexp"
	"strings"
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

	inFrontMatter := false
	frontMatterEnd := -1
	var id, title string
	var aliases []string
	var tags []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if i == 0 && trimmed == "---" {
			inFrontMatter = true
			continue
		}

		if inFrontMatter && trimmed == "---" {
			inFrontMatter = false
			frontMatterEnd = i
			continue
		}

		if inFrontMatter {
			// Parse YAML (simple key: value parsing)
			if strings.HasPrefix(trimmed, "id: ") {
				id = strings.TrimSpace(trimmed[4:])
			} else if strings.HasPrefix(trimmed, "title: ") {
				title = strings.TrimSpace(trimmed[7:])
			} else if strings.HasPrefix(trimmed, "- ") && (len(aliases) > 0 || len(tags) > 0 || isInListContext(lines, i)) {
				// This is a list item for aliases or tags
				item := strings.TrimSpace(trimmed[2:])
				// Determine if we're in aliases or tags context
				// This is a simple heuristic - in real YAML we'd need proper parsing
				if i > 0 && strings.Contains(lines[i-1], "aliases") {
					aliases = append(aliases, item)
				} else if i > 0 && strings.Contains(lines[i-1], "tags") {
					tags = append(tags, item)
				} else {
					// Check previous non-empty line
					for k := i - 1; k >= 0; k-- {
						prev := strings.TrimSpace(lines[k])
						if strings.HasPrefix(prev, "aliases:") {
							aliases = append(aliases, item)
							break
						} else if strings.HasPrefix(prev, "tags:") {
							tags = append(tags, item)
							break
						} else if prev != "" && !strings.HasPrefix(prev, "-") {
							break
						}
					}
				}
			} else if strings.HasPrefix(trimmed, "aliases:") {
				// Start of aliases list
				continue
			} else if strings.HasPrefix(trimmed, "tags:") {
				// Start of tags list
				continue
			}
			continue
		}

		// Not in front matter, add to body
		if i > frontMatterEnd {
			bodyLines = append(bodyLines, line)
		}
	}

	// Skip leading blank lines in body
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
		bodyLines = bodyLines[1:]
	}

	// Build properties drawer
	if id != "" || len(aliases) > 0 {
		properties.WriteString(":PROPERTIES:\n")
		if id != "" {
			properties.WriteString(":ID: " + id + "\n")
		}
		if len(aliases) > 0 {
			aliasStr := ""
			for i, alias := range aliases {
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
	if title != "" {
		properties.WriteString("#+title: " + title + "\n")
	}

	// Add tags
	if len(tags) > 0 {
		tagStr := ":" + strings.Join(tags, ":") + ":"
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

// isInListContext checks if we're currently in a YAML list (aliases or tags)
func isInListContext(lines []string, currentIndex int) bool {
	// Look backwards for "aliases:" or "tags:"
	for i := currentIndex - 1; i >= 0; i-- {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "---" {
			return false
		}
		if strings.HasPrefix(trimmed, "aliases:") || strings.HasPrefix(trimmed, "tags:") {
			return true
		}
		if trimmed != "" && !strings.HasPrefix(trimmed, "-") {
			return false
		}
	}
	return false
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
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		// Fallback to a deterministic UUID if random fails
		return "00000000-0000-0000-0000-000000000000"
	}

	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4],
		uuid[4:6],
		uuid[6:8],
		uuid[8:10],
		uuid[10:16])
}
