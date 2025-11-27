package convert

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// FeatureMarker represents a custom feature that needs special handling
type FeatureMarker struct {
	MarkerID string              // Unique placeholder: "NOTEBR_MARKER_abc123"
	Feature  string              // Feature type: "org-roam-id", "task-scheduled", etc.
	Original string              // Original syntax
	Convert  func() string       // Conversion function
	Context  map[string]string   // Additional context data
}

// HybridConverter handles conversion using the hybrid annotation pattern
type HybridConverter struct {
	markers []FeatureMarker
	idMap   map[string]string
}

// NewHybridConverter creates a new hybrid converter
func NewHybridConverter(idMap map[string]string) *HybridConverter {
	return &HybridConverter{
		markers: make([]FeatureMarker, 0),
		idMap:   idMap,
	}
}

// createMarker generates a unique marker for a feature
func (c *HybridConverter) createMarker(featureType, original string, context map[string]string, convertFunc func() string) FeatureMarker {
	markerID := fmt.Sprintf("NOTEBR_MARKER_%s", uuid.New().String()[:8])

	return FeatureMarker{
		MarkerID: markerID,
		Feature:  featureType,
		Original: original,
		Convert:  convertFunc,
		Context:  context,
	}
}

// extractOrgRoamLinks extracts org-roam ID links and replaces them with markers
func (c *HybridConverter) extractOrgRoamLinks(content string) string {
	// Pattern: [[id:uuid][description]] or [[id:uuid]]
	re := regexp.MustCompile(`\[\[id:([^\]]+)\](?:\[([^\]]+)\])?\]`)

	result := re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		uuid := submatches[1]
		description := ""
		if len(submatches) > 2 && submatches[2] != "" {
			description = submatches[2]
		}

		context := map[string]string{
			"uuid":        uuid,
			"description": description,
		}

		marker := c.createMarker("org-roam-id", match, context, func() string {
			// Convert to wikilink
			filename, ok := c.idMap[uuid]
			if !ok {
				filename = uuid
			}

			if description != "" {
				return fmt.Sprintf("[[%s|%s]]", filename, description)
			}
			return fmt.Sprintf("[[%s]]", filename)
		})

		c.markers = append(c.markers, marker)
		return marker.MarkerID
	})

	return result
}

// extractWikilinks extracts Obsidian wikilinks and replaces them with markers
func (c *HybridConverter) extractWikilinks(content string) string {
	// Pattern: [[filename|description]] or [[filename]]
	re := regexp.MustCompile(`\[\[([^\]|]+)(?:\|([^\]]+))?\]\]`)

	// Create reverse map (filename -> id)
	reverseMap := make(map[string]string)
	for id, filename := range c.idMap {
		reverseMap[filename] = id
	}

	result := re.ReplaceAllStringFunc(content, func(match string) string {
		submatches := re.FindStringSubmatch(match)
		if len(submatches) < 2 {
			return match
		}

		filename := submatches[1]
		description := ""
		if len(submatches) > 2 && submatches[2] != "" {
			description = submatches[2]
		}

		context := map[string]string{
			"filename":    filename,
			"description": description,
		}

		marker := c.createMarker("wikilink", match, context, func() string {
			// Convert to org-roam link
			id, ok := reverseMap[filename]
			if !ok {
				// Check if it's already a UUID
				if isUUID(filename) {
					id = filename
				} else {
					// Generate new UUID
					id = GenerateOrgID()
				}
			}

			if description != "" {
				return fmt.Sprintf("[[id:%s][%s]]", id, description)
			}
			return fmt.Sprintf("[[id:%s]]", id)
		})

		c.markers = append(c.markers, marker)
		return marker.MarkerID
	})

	return result
}

// extractOrgTasks extracts org-mode task metadata and replaces with markers
func (c *HybridConverter) extractOrgTasks(content string) string {
	// Pattern: * TODO/DONE with optional priority and scheduling
	lines := strings.Split(content, "\n")
	var result []string

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Check if this is a task header
		if strings.HasPrefix(trimmed, "*") {
			stars := countLeadingChars(trimmed, '*')
			if stars > 0 {
				rest := strings.TrimSpace(trimmed[stars:])

				// Check for TODO/DONE
				if strings.HasPrefix(rest, "TODO ") || strings.HasPrefix(rest, "DONE ") {
					// Extract task components
					status := ""
					priority := ""
					taskText := rest

					if strings.HasPrefix(rest, "TODO ") {
						status = "TODO"
						taskText = strings.TrimSpace(rest[5:])
					} else if strings.HasPrefix(rest, "DONE ") {
						status = "DONE"
						taskText = strings.TrimSpace(rest[5:])
					}

					// Check for priority
					if strings.HasPrefix(taskText, "[#") && len(taskText) >= 4 && taskText[3] == ']' {
						priority = string(taskText[2])
						taskText = strings.TrimSpace(taskText[4:])
					}

					// Look ahead for scheduling info
					scheduling := []string{}
					j := i + 1
					for j < len(lines) {
						nextLine := strings.TrimSpace(lines[j])
						if strings.HasPrefix(nextLine, "SCHEDULED:") ||
						   strings.HasPrefix(nextLine, "DEADLINE:") ||
						   strings.HasPrefix(nextLine, "CLOSED:") {
							scheduling = append(scheduling, nextLine)
							j++
						} else {
							break
						}
					}

					// Create marker for entire task block
					originalBlock := []string{line}
					originalBlock = append(originalBlock, scheduling...)

					context := map[string]string{
						"stars":      strings.Repeat("*", stars),
						"status":     status,
						"priority":   priority,
						"taskText":   taskText,
						"scheduling": strings.Join(scheduling, "\n"),
					}

					marker := c.createMarker("org-task", strings.Join(originalBlock, "\n"), context, func() string {
						// This will be handled by the library, just return empty for now
						return ""
					})

					c.markers = append(c.markers, marker)
					result = append(result, marker.MarkerID)

					// Skip the scheduling lines we already processed
					i = j
					continue
				}
			}
		}

		result = append(result, line)
		i++
	}

	return strings.Join(result, "\n")
}

// applyMarkers replaces markers with their converted values
func (c *HybridConverter) applyMarkers(content string) string {
	result := content
	for _, marker := range c.markers {
		replacement := marker.Convert()
		result = strings.Replace(result, marker.MarkerID, replacement, 1)
	}
	return result
}

// reset clears all markers for a new conversion
func (c *HybridConverter) reset() {
	c.markers = make([]FeatureMarker, 0)
}
