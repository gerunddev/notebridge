package convert

import (
	"os"
	"strings"
	"testing"
)

func TestOrgToMarkdown(t *testing.T) {
	// Read org fixture
	orgContent, err := os.ReadFile("testdata/sample.org")
	if err != nil {
		t.Fatalf("Failed to read org fixture: %v", err)
	}

	// Read expected markdown output
	expectedMD, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatalf("Failed to read markdown fixture: %v", err)
	}

	// Create ID map for conversion
	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
		"987fcdeb-51a2-43f7-8123-456789abcdef": "987fcdeb-51a2-43f7-8123-456789abcdef",
	}

	// Convert org to markdown
	actualMD, err := OrgToMarkdown(string(orgContent), idMap)
	if err != nil {
		t.Fatalf("OrgToMarkdown failed: %v", err)
	}

	// Normalize whitespace for comparison
	expected := normalizeWhitespace(string(expectedMD))
	actual := normalizeWhitespace(actualMD)

	if actual != expected {
		t.Errorf("Conversion mismatch.\n\nExpected:\n%s\n\nGot:\n%s", expected, actual)

		// Show diff for debugging
		showDiff(t, expected, actual)
	}
}

func TestConvertOrgHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "level 1 header",
			input:    "* Introduction",
			expected: "# Introduction",
		},
		{
			name:     "level 2 header",
			input:    "** Tasks and Projects",
			expected: "## Tasks and Projects",
		},
		{
			name:     "level 3 header",
			input:    "*** Subsection",
			expected: "### Subsection",
		},
		{
			name:     "plain text",
			input:    "This is plain text",
			expected: "This is plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ConvertOrgHeader(tt.input)
			if actual != tt.expected {
				t.Errorf("ConvertOrgHeader(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestConvertOrgTask(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "TODO task",
			input:    "** TODO Write tests",
			expected: "## - [ ] Write tests",
		},
		{
			name:     "DONE task",
			input:    "** DONE Complete setup",
			expected: "## - [x] Complete setup",
		},
		{
			name:     "TODO with priority A",
			input:    "** TODO [#A] High priority",
			expected: "## - [ ] High priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ConvertOrgTask(tt.input)
			if actual != tt.expected {
				t.Errorf("ConvertOrgTask(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestConvertOrgLink(t *testing.T) {
	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "link with description",
			input:    "[[id:123e4567-e89b-12d3-a456-426614174000][Related Note]]",
			expected: "[[Related Note|Related Note]]",
		},
		{
			name:     "link without description",
			input:    "[[id:123e4567-e89b-12d3-a456-426614174000]]",
			expected: "[[Related Note]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ConvertOrgLink(tt.input, idMap)
			if actual != tt.expected {
				t.Errorf("ConvertOrgLink(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

// Helper functions

func normalizeWhitespace(s string) string {
	// Normalize line endings and trim trailing whitespace
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func showDiff(t *testing.T, expected, actual string) {
	expectedLines := strings.Split(expected, "\n")
	actualLines := strings.Split(actual, "\n")

	maxLines := len(expectedLines)
	if len(actualLines) > maxLines {
		maxLines = len(actualLines)
	}

	t.Log("\nLine-by-line diff:")
	for i := 0; i < maxLines; i++ {
		var expLine, actLine string
		if i < len(expectedLines) {
			expLine = expectedLines[i]
		}
		if i < len(actualLines) {
			actLine = actualLines[i]
		}

		if expLine != actLine {
			t.Logf("Line %d:\n  Expected: %q\n  Actual:   %q", i+1, expLine, actLine)
		}
	}
}
