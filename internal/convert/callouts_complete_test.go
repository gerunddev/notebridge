package convert

import (
	"strings"
	"testing"
)

// TestAllObsidianCalloutTypes tests all Obsidian callout types
// Note: "quote" and "cite" are excluded as they map to standard #+BEGIN_QUOTE blocks
func TestAllObsidianCalloutTypes(t *testing.T) {
	tests := []struct {
		calloutType string
		orgBlock    string
	}{
		{"note", "NOTE"},
		{"abstract", "ABSTRACT"},
		{"summary", "SUMMARY"},
		{"tldr", "TLDR"},
		{"info", "INFO"},
		{"todo", "TODO"},
		{"tip", "TIP"},
		{"hint", "HINT"},
		{"important", "IMPORTANT"},
		{"success", "SUCCESS"},
		{"check", "CHECK"},
		{"done", "DONE"},
		{"question", "QUESTION"},
		{"help", "HELP"},
		{"faq", "FAQ"},
		{"warning", "WARNING"},
		{"caution", "CAUTION"},
		{"attention", "ATTENTION"},
		{"failure", "FAILURE"},
		{"fail", "FAIL"},
		{"missing", "MISSING"},
		{"danger", "DANGER"},
		{"error", "ERROR"},
		{"bug", "BUG"},
		{"example", "EXAMPLE"},
	}

	idMap := map[string]string{}

	for _, tt := range tests {
		t.Run("md->org "+tt.calloutType, func(t *testing.T) {
			md := "> [!" + tt.calloutType + "]\n> This is content."

			result, err := MarkdownToOrg(md, idMap)
			if err != nil {
				t.Fatalf("MarkdownToOrg failed: %v", err)
			}

			expected := "#+BEGIN_" + tt.orgBlock + "\nThis is content.\n#+END_" + tt.orgBlock

			result = strings.TrimSpace(result)
			expected = strings.TrimSpace(expected)

			if result != expected {
				t.Errorf("Conversion mismatch for %s.\nExpected:\n%s\n\nGot:\n%s",
					tt.calloutType, expected, result)
			}
		})

		t.Run("org->md "+tt.orgBlock, func(t *testing.T) {
			org := "#+BEGIN_" + tt.orgBlock + "\nThis is content.\n#+END_" + tt.orgBlock

			result, err := OrgToMarkdown(org, idMap)
			if err != nil {
				t.Fatalf("OrgToMarkdown failed: %v", err)
			}

			expected := "> [!" + strings.ToLower(tt.orgBlock) + "]\n> This is content."

			result = strings.TrimSpace(result)
			expected = strings.TrimSpace(expected)

			if result != expected {
				t.Errorf("Conversion mismatch for %s.\nExpected:\n%s\n\nGot:\n%s",
					tt.orgBlock, expected, result)
			}
		})
	}
}

func TestCalloutWithCustomTitle(t *testing.T) {
	tests := []struct {
		name        string
		calloutType string
		title       string
	}{
		{"tip with title", "tip", "Pro Tip"},
		{"warning with title", "warning", "Watch Out!"},
		{"example with title", "example", "Code Example"},
	}

	idMap := map[string]string{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			md := "> [!" + tt.calloutType + "] " + tt.title + "\n> Content here."

			result, err := MarkdownToOrg(md, idMap)
			if err != nil {
				t.Fatalf("MarkdownToOrg failed: %v", err)
			}

			// Title becomes first line of content
			orgType := strings.ToUpper(tt.calloutType)
			expected := "#+BEGIN_" + orgType + "\n" + tt.title + "\nContent here.\n#+END_" + orgType

			result = strings.TrimSpace(result)
			expected = strings.TrimSpace(expected)

			if result != expected {
				t.Errorf("Conversion mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestCalloutFoldable(t *testing.T) {
	// Obsidian supports +/- for foldable callouts
	// We preserve this in conversion but don't need special handling
	md := "> [!note]+ Expandable\n> This can be folded."

	result, err := MarkdownToOrg(md, map[string]string{})
	if err != nil {
		t.Fatalf("MarkdownToOrg failed: %v", err)
	}

	// The + is part of the title, so it becomes content
	expected := "#+BEGIN_NOTE\n+ Expandable\nThis can be folded.\n#+END_NOTE"

	result = strings.TrimSpace(result)
	expected = strings.TrimSpace(expected)

	if result != expected {
		t.Errorf("Foldable callout mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
	}
}
