package convert

import (
	"os"
	"testing"
)

// TestRoundtripOrgToMdToOrg tests that converting org->md->org preserves content
func TestRoundtripOrgToMdToOrg(t *testing.T) {
	// Read original org content
	orgContent, err := os.ReadFile("testdata/sample.org")
	if err != nil {
		t.Fatalf("Failed to read org fixture: %v", err)
	}

	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
		"987fcdeb-51a2-43f7-8123-456789abcdef": "987fcdeb-51a2-43f7-8123-456789abcdef",
	}

	// Convert org -> markdown
	mdContent, err := OrgToMarkdown(string(orgContent), idMap)
	if err != nil {
		t.Fatalf("OrgToMarkdown failed: %v", err)
	}

	// Convert markdown -> org
	orgContentRoundtrip, err := MarkdownToOrg(mdContent, idMap)
	if err != nil {
		t.Fatalf("MarkdownToOrg failed: %v", err)
	}

	// Compare
	expected := normalizeWhitespace(string(orgContent))
	actual := normalizeWhitespace(orgContentRoundtrip)

	if actual != expected {
		t.Errorf("Roundtrip org->md->org failed to preserve content.\n\nOriginal:\n%s\n\nAfter roundtrip:\n%s",
			expected, actual)
		showDiff(t, expected, actual)
	}
}

// TestRoundtripMdToOrgToMd tests that converting md->org->md preserves content
func TestRoundtripMdToOrgToMd(t *testing.T) {
	// Read original markdown content
	mdContent, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatalf("Failed to read markdown fixture: %v", err)
	}

	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
		"987fcdeb-51a2-43f7-8123-456789abcdef": "987fcdeb-51a2-43f7-8123-456789abcdef",
	}

	// Convert markdown -> org
	orgContent, err := MarkdownToOrg(string(mdContent), idMap)
	if err != nil {
		t.Fatalf("MarkdownToOrg failed: %v", err)
	}

	// Convert org -> markdown
	mdContentRoundtrip, err := OrgToMarkdown(orgContent, idMap)
	if err != nil {
		t.Fatalf("OrgToMarkdown failed: %v", err)
	}

	// Compare
	expected := normalizeWhitespace(string(mdContent))
	actual := normalizeWhitespace(mdContentRoundtrip)

	if actual != expected {
		t.Errorf("Roundtrip md->org->md failed to preserve content.\n\nOriginal:\n%s\n\nAfter roundtrip:\n%s",
			expected, actual)
		showDiff(t, expected, actual)
	}
}

// TestIdempotence tests that converting multiple times produces the same result
func TestIdempotenceOrgToMd(t *testing.T) {
	orgContent, err := os.ReadFile("testdata/sample.org")
	if err != nil {
		t.Fatalf("Failed to read org fixture: %v", err)
	}

	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
		"987fcdeb-51a2-43f7-8123-456789abcdef": "987fcdeb-51a2-43f7-8123-456789abcdef",
	}

	// Convert once
	md1, err := OrgToMarkdown(string(orgContent), idMap)
	if err != nil {
		t.Fatalf("First OrgToMarkdown failed: %v", err)
	}

	// Convert back
	org1, err := MarkdownToOrg(md1, idMap)
	if err != nil {
		t.Fatalf("MarkdownToOrg failed: %v", err)
	}

	// Convert again
	md2, err := OrgToMarkdown(org1, idMap)
	if err != nil {
		t.Fatalf("Second OrgToMarkdown failed: %v", err)
	}

	// First and second markdown should be identical
	if normalizeWhitespace(md1) != normalizeWhitespace(md2) {
		t.Errorf("Conversion is not idempotent.\n\nFirst conversion:\n%s\n\nSecond conversion:\n%s",
			md1, md2)
		showDiff(t, normalizeWhitespace(md1), normalizeWhitespace(md2))
	}
}
