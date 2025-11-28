package convert

import (
	"fmt"
	"regexp"
	"strings"
)

// OrgToMarkdown converts org-mode content to markdown
func OrgToMarkdown(orgContent string, idMap map[string]string) (string, error) {
	lines := strings.Split(orgContent, "\n")

	// Extract properties drawer and convert to front matter
	frontMatter, bodyLines := extractOrgPropertiesFromLines(lines)

	var md strings.Builder

	// Write front matter if present
	if frontMatter != "" {
		md.WriteString("---\n")
		md.WriteString(frontMatter)
		md.WriteString("---\n\n")
	}

	inCodeBlock := false
	inQuoteBlock := false
	inSpecialBlock := false
	codeBlockLang := ""
	specialBlockType := ""

	for i := 0; i < len(bodyLines); i++ {
		line := bodyLines[i]
		trimmed := strings.TrimSpace(line)

		// Handle code blocks
		if strings.HasPrefix(trimmed, "#+BEGIN_SRC") {
			inCodeBlock = true
			// Extract language
			parts := strings.Fields(trimmed)
			if len(parts) > 1 {
				codeBlockLang = parts[1]
			}
			md.WriteString("```" + codeBlockLang + "\n")
			continue
		}
		if strings.HasPrefix(trimmed, "#+END_SRC") {
			inCodeBlock = false
			codeBlockLang = ""
			md.WriteString("```\n")
			continue
		}
		if inCodeBlock {
			md.WriteString(line + "\n")
			continue
		}

		// Handle quote blocks
		if strings.HasPrefix(trimmed, "#+BEGIN_QUOTE") {
			inQuoteBlock = true
			continue
		}
		if strings.HasPrefix(trimmed, "#+END_QUOTE") {
			inQuoteBlock = false
			continue
		}
		if inQuoteBlock {
			md.WriteString("> " + trimmed + "\n")
			continue
		}

		// Handle special blocks -> Obsidian callouts
		// Supports all default Obsidian callout types (except quote/cite which are standard blockquotes)
		if strings.HasPrefix(trimmed, "#+BEGIN_") {
			blockType := strings.ToLower(strings.TrimPrefix(trimmed, "#+BEGIN_"))
			// All supported callout types
			// Note: "quote" and "cite" excluded as they map to standard #+BEGIN_QUOTE
			validCallouts := map[string]bool{
				"note": true, "abstract": true, "summary": true, "tldr": true,
				"info": true, "todo": true, "tip": true, "hint": true, "important": true,
				"success": true, "check": true, "done": true,
				"question": true, "help": true, "faq": true,
				"warning": true, "caution": true, "attention": true,
				"failure": true, "fail": true, "missing": true,
				"danger": true, "error": true, "bug": true,
				"example": true,
			}
			if validCallouts[blockType] {
				inSpecialBlock = true
				specialBlockType = blockType
				md.WriteString("> [!" + blockType + "]\n")
				continue
			}
		}
		if strings.HasPrefix(trimmed, "#+END_") {
			blockType := strings.ToLower(strings.TrimPrefix(trimmed, "#+END_"))
			if blockType == specialBlockType {
				inSpecialBlock = false
				specialBlockType = ""
				md.WriteString("\n")
				continue
			}
		}
		if inSpecialBlock {
			md.WriteString("> " + trimmed + "\n")
			continue
		}

		// Skip #+title and #+filetags (already in front matter)
		if strings.HasPrefix(trimmed, "#+title:") || strings.HasPrefix(trimmed, "#+filetags:") {
			continue
		}

		// Skip scheduling info lines (will be processed with tasks)
		if strings.HasPrefix(trimmed, "SCHEDULED:") || strings.HasPrefix(trimmed, "DEADLINE:") || strings.HasPrefix(trimmed, "CLOSED:") {
			continue
		}

		// Handle headers (with potential TODO/DONE and priorities)
		if strings.HasPrefix(trimmed, "*") {
			stars := countLeadingChars(trimmed, '*')
			if stars > 0 {
				rest := strings.TrimSpace(trimmed[stars:])

				// Check for TODO/DONE
				isTodo := false
				isDone := false
				if strings.HasPrefix(rest, "TODO ") {
					isTodo = true
					rest = strings.TrimSpace(rest[5:])
				} else if strings.HasPrefix(rest, "DONE ") {
					isDone = true
					rest = strings.TrimSpace(rest[5:])
				}

				// Check for priority
				priority := ""
				if strings.HasPrefix(rest, "[#") {
					if len(rest) >= 4 && rest[3] == ']' {
						priority = string(rest[2])
						rest = strings.TrimSpace(rest[4:])
					}
				}

				// Build markdown header
				hashes := strings.Repeat("#", stars)

				if isTodo || isDone {
					// Task format
					checkbox := "[ ]"
					if isDone {
						checkbox = "[x]"
					}
					md.WriteString(hashes + " - " + checkbox + " " + rest + "\n")

					// Look ahead for scheduling info on next lines
					j := i + 1
					for j < len(bodyLines) {
						nextLine := strings.TrimSpace(bodyLines[j])
						if strings.HasPrefix(nextLine, "SCHEDULED:") {
							date := extractOrgDate(nextLine)
							md.WriteString("⏳ " + date + "\n")
							j++
						} else if strings.HasPrefix(nextLine, "DEADLINE:") {
							date := extractOrgDate(nextLine)
							md.WriteString("📅 " + date + "\n")
							j++
						} else if strings.HasPrefix(nextLine, "CLOSED:") {
							date := extractOrgDate(nextLine)
							md.WriteString("✅ " + date + "\n")
							j++
						} else {
							break
						}
					}

					// Add priority if present
					if priority != "" {
						priorityLevel := "medium"
						switch priority {
						case "A":
							priorityLevel = "high"
						case "C":
							priorityLevel = "low"
						}
						md.WriteString("Priority: " + priorityLevel + "\n")
					}
				} else {
					// Regular header
					md.WriteString(hashes + " " + rest + "\n")
				}
				continue
			}
		}

		// Convert embeds and links in regular content
		convertedLine := convertOrgEmbeds(line)
		convertedLine = convertOrgLinks(convertedLine, idMap)

		// Write the line (preserve blank lines)
		md.WriteString(convertedLine + "\n")
	}

	return strings.TrimSpace(md.String()), nil
}

// extractOrgPropertiesFromLines extracts properties drawer and returns front matter + body lines
func extractOrgPropertiesFromLines(lines []string) (string, []string) {
	var frontMatter strings.Builder
	var bodyLines []string

	inProperties := false
	propertiesEnd := -1
	var title, id string
	var aliases []string
	var tags []string
	var refs []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == ":PROPERTIES:" {
			inProperties = true
			continue
		}

		if trimmed == ":END:" && inProperties {
			inProperties = false
			propertiesEnd = i
			continue
		}

		if inProperties {
			// Parse property
			if strings.HasPrefix(trimmed, ":ID:") {
				id = strings.TrimSpace(trimmed[4:])
			} else if strings.HasPrefix(trimmed, ":ROAM_ALIASES:") {
				aliasStr := strings.TrimSpace(trimmed[14:])
				// Parse "alias1" "alias2" format
				aliases = parseOrgAliases(aliasStr)
			} else if strings.HasPrefix(trimmed, ":ROAM_REFS:") {
				refStr := strings.TrimSpace(trimmed[11:])
				// Parse space-separated refs (URLs, citation keys, etc.)
				refs = strings.Fields(refStr)
			}
			continue
		}

		// Check for #+title
		if strings.HasPrefix(trimmed, "#+title:") {
			title = strings.TrimSpace(trimmed[8:])
			continue
		}

		// Check for #+filetags
		if strings.HasPrefix(trimmed, "#+filetags:") {
			tagStr := strings.TrimSpace(trimmed[11:])
			tags = parseOrgTags(tagStr)
			continue
		}

		// Not in properties, add to body (but skip already processed lines)
		if i > propertiesEnd {
			bodyLines = append(bodyLines, line)
		}
	}

	// Skip leading blank lines in body
	for len(bodyLines) > 0 && strings.TrimSpace(bodyLines[0]) == "" {
		bodyLines = bodyLines[1:]
	}

	// Build YAML front matter
	if id != "" {
		frontMatter.WriteString("id: " + id + "\n")
	}
	if title != "" {
		frontMatter.WriteString("title: " + title + "\n")
	}
	if len(aliases) > 0 {
		frontMatter.WriteString("aliases:\n")
		for _, alias := range aliases {
			frontMatter.WriteString("  - " + alias + "\n")
		}
	}
	if len(tags) > 0 {
		frontMatter.WriteString("tags:\n")
		for _, tag := range tags {
			frontMatter.WriteString("  - " + tag + "\n")
		}
	}
	if len(refs) > 0 {
		frontMatter.WriteString("refs:\n")
		for _, ref := range refs {
			frontMatter.WriteString("  - " + ref + "\n")
		}
	}

	return frontMatter.String(), bodyLines
}

// parseOrgAliases parses "alias1" "alias2" format
func parseOrgAliases(s string) []string {
	re := regexp.MustCompile(`"([^"]+)"`)
	matches := re.FindAllStringSubmatch(s, -1)
	var aliases []string
	for _, match := range matches {
		if len(match) > 1 {
			aliases = append(aliases, match[1])
		}
	}
	return aliases
}

// parseOrgTags parses :tag1:tag2:tag3: format
func parseOrgTags(s string) []string {
	s = strings.Trim(s, ":")
	if s == "" {
		return nil
	}
	return strings.Split(s, ":")
}

// extractOrgDate extracts date from SCHEDULED: <2024-01-15> format
func extractOrgDate(line string) string {
	re := regexp.MustCompile(`<(\d{4}-\d{2}-\d{2})`)
	matches := re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}

	// Also handle [2024-01-15] format for CLOSED
	re = regexp.MustCompile(`\[(\d{4}-\d{2}-\d{2})`)
	matches = re.FindStringSubmatch(line)
	if len(matches) > 1 {
		return matches[1]
	}

	return ""
}

// convertOrgLinks converts all org-mode links in a line to markdown wikilinks
func convertOrgLinks(line string, idMap map[string]string) string {
	// Pattern: [[id:uuid][description]] or [[id:uuid]]
	re := regexp.MustCompile(`\[\[id:([^\]]+)\](?:\[([^\]]+)\])?\]`)

	return re.ReplaceAllStringFunc(line, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		uuid := submatches[1]
		description := ""
		if len(submatches) > 2 {
			description = submatches[2]
		}

		// Look up filename from ID
		filename, ok := idMap[uuid]
		if !ok {
			// ID not in map, use uuid as filename
			filename = uuid
		}

		// Build wikilink
		if description != "" {
			return fmt.Sprintf("[[%s|%s]]", filename, description)
		}
		return fmt.Sprintf("[[%s]]", filename)
	})
}

// ConvertOrgHeader converts org-mode header to markdown
// * Header → # Header
// ** Subheading → ## Subheading
func ConvertOrgHeader(line string) string {
	trimmed := strings.TrimSpace(line)
	stars := countLeadingChars(trimmed, '*')

	if stars == 0 {
		return line
	}

	// Extract content after asterisks
	content := strings.TrimSpace(trimmed[stars:])

	// Build markdown header
	return strings.Repeat("#", stars) + " " + content
}

// ConvertOrgLink converts org-roam link to Obsidian wikilink
// [[id:uuid][Description]] → [[filename|Description]]
// [[id:uuid]] → [[filename]]
func ConvertOrgLink(link string, idMap map[string]string) string {
	return convertOrgLinks(link, idMap)
}

// ConvertOrgTask converts org-mode task to markdown checkbox
// * TODO Task → - [ ] Task
// * DONE Task → - [x] Task
func ConvertOrgTask(line string) string {
	trimmed := strings.TrimSpace(line)
	stars := countLeadingChars(trimmed, '*')

	if stars == 0 {
		return line
	}

	rest := strings.TrimSpace(trimmed[stars:])

	// Check for TODO/DONE
	if strings.HasPrefix(rest, "TODO ") {
		content := strings.TrimSpace(rest[5:])
		// Remove priority if present
		if strings.HasPrefix(content, "[#") && len(content) >= 4 && content[3] == ']' {
			content = strings.TrimSpace(content[4:])
		}
		return strings.Repeat("#", stars) + " - [ ] " + content
	} else if strings.HasPrefix(rest, "DONE ") {
		content := strings.TrimSpace(rest[5:])
		// Remove priority if present
		if strings.HasPrefix(content, "[#") && len(content) >= 4 && content[3] == ']' {
			content = strings.TrimSpace(content[4:])
		}
		return strings.Repeat("#", stars) + " - [x] " + content
	}

	return line
}

// ExtractOrgProperties extracts properties drawer and converts to YAML front matter
func ExtractOrgProperties(content string) (frontMatter string, bodyContent string) {
	lines := strings.Split(content, "\n")
	fm, body := extractOrgPropertiesFromLines(lines)
	return fm, strings.Join(body, "\n")
}

// countLeadingChars counts leading occurrences of a character
func countLeadingChars(s string, ch rune) int {
	count := 0
	for _, c := range s {
		if c == ch {
			count++
		} else if c == ' ' {
			break
		} else {
			break
		}
	}
	return count
}

// convertOrgEmbeds converts org-mode embeds to Obsidian embeds
// # EMBED: note → ![[note]]
// [[file:image.png]] → ![[image.png]]
func convertOrgEmbeds(line string) string {
	trimmed := strings.TrimSpace(line)

	// Convert comment-style embeds: # EMBED: note
	if strings.HasPrefix(trimmed, "# EMBED:") {
		embedTarget := strings.TrimSpace(strings.TrimPrefix(trimmed, "# EMBED:"))
		return strings.Replace(line, trimmed, fmt.Sprintf("![[%s]]", embedTarget), 1)
	}

	// Convert file links to image embeds: [[file:image.png]] → ![[image.png]]
	re := regexp.MustCompile(`\[\[file:([^\]]+)\]\]`)
	return re.ReplaceAllString(line, "![[$1]]")
}
