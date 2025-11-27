package convert

import (
	"os"
	"testing"
)

func TestMarkdownToOrg(t *testing.T) {
	// Read markdown fixture
	mdContent, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatalf("Failed to read markdown fixture: %v", err)
	}

	// Read expected org output
	expectedOrg, err := os.ReadFile("testdata/sample.org")
	if err != nil {
		t.Fatalf("Failed to read org fixture: %v", err)
	}

	// Create ID map for conversion (reverse of org-to-md)
	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
		"987fcdeb-51a2-43f7-8123-456789abcdef": "987fcdeb-51a2-43f7-8123-456789abcdef",
	}

	// Convert markdown to org
	actualOrg, err := MarkdownToOrg(string(mdContent), idMap)
	if err != nil {
		t.Fatalf("MarkdownToOrg failed: %v", err)
	}

	// Normalize whitespace for comparison
	expected := normalizeWhitespace(string(expectedOrg))
	actual := normalizeWhitespace(actualOrg)

	if actual != expected {
		t.Errorf("Conversion mismatch.\n\nExpected:\n%s\n\nGot:\n%s", expected, actual)

		// Show diff for debugging
		showDiff(t, expected, actual)
	}
}

func TestConvertMarkdownHeader(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "level 1 header",
			input:    "# Introduction",
			expected: "* Introduction",
		},
		{
			name:     "level 2 header",
			input:    "## Tasks and Projects",
			expected: "** Tasks and Projects",
		},
		{
			name:     "level 3 header",
			input:    "### Subsection",
			expected: "*** Subsection",
		},
		{
			name:     "plain text",
			input:    "This is plain text",
			expected: "This is plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ConvertMarkdownHeader(tt.input)
			if actual != tt.expected {
				t.Errorf("ConvertMarkdownHeader(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestConvertMarkdownTask(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unchecked task",
			input:    "- [ ] Write tests",
			expected: "* TODO Write tests",
		},
		{
			name:     "checked task",
			input:    "- [x] Complete setup",
			expected: "* DONE Complete setup",
		},
		{
			name:     "unchecked with header level",
			input:    "## - [ ] High priority",
			expected: "** TODO High priority",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ConvertMarkdownTask(tt.input)
			if actual != tt.expected {
				t.Errorf("ConvertMarkdownTask(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestConvertWikilink(t *testing.T) {
	// For markdown to org, we need reverse lookup (filename -> ID)
	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "wikilink with alias",
			input:    "[[Related Note|Related Note]]",
			expected: "[[id:123e4567-e89b-12d3-a456-426614174000][Related Note]]",
		},
		{
			name:     "simple wikilink",
			input:    "[[Related Note]]",
			expected: "[[id:123e4567-e89b-12d3-a456-426614174000]]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := ConvertWikilink(tt.input, idMap)
			if actual != tt.expected {
				t.Errorf("ConvertWikilink(%q) = %q, want %q", tt.input, actual, tt.expected)
			}
		})
	}
}

func TestGenerateOrgID(t *testing.T) {
	id := GenerateOrgID()
	if id == "" {
		t.Error("GenerateOrgID returned empty string")
	}

	// Basic UUID format check (not exhaustive)
	if len(id) != 36 {
		t.Errorf("GenerateOrgID returned %q with length %d, expected UUID format with length 36", id, len(id))
	}
}

func TestExtractYAMLFrontMatter(t *testing.T) {
	input := `---
title: My Note
tags:
  - test
  - sample
---

# Content

This is the body.
`

	properties, body := ExtractYAMLFrontMatter(input)

	// Check that properties were extracted
	if properties == "" {
		t.Error("Expected non-empty properties")
	}

	// Check that body doesn't contain front matter
	if body == "" {
		t.Error("Expected non-empty body")
	}

	// Body should not start with ---
	if len(body) > 0 && body[0:3] == "---" {
		t.Error("Body should not contain YAML front matter delimiter")
	}
}

func TestExtractOrgProperties(t *testing.T) {
	input := `:PROPERTIES:
:ID: 123
:TITLE: My Note
:END:

* Content

This is the body.
`

	frontMatter, body := ExtractOrgProperties(input)

	// Check that front matter was extracted
	if frontMatter == "" {
		t.Error("Expected non-empty front matter")
	}

	// Check that body doesn't contain properties drawer
	if body == "" {
		t.Error("Expected non-empty body")
	}

	// Body should not start with :PROPERTIES:
	if len(body) > 12 && body[0:12] == ":PROPERTIES:" {
		t.Error("Body should not contain properties drawer")
	}
}
