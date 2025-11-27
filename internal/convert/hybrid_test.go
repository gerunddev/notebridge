package convert

import (
	"os"
	"strings"
	"testing"
)

func TestHybridOrgToMarkdown(t *testing.T) {
	// Read test fixtures
	orgContent, err := os.ReadFile("testdata/sample.org")
	if err != nil {
		t.Fatalf("Failed to read sample.org: %v", err)
	}

	expectedMd, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatalf("Failed to read sample.md: %v", err)
	}

	// Create ID map for testing (matching the original test setup)
	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
		"987fcdeb-51a2-43f7-8123-456789abcdef": "987fcdeb-51a2-43f7-8123-456789abcdef",
	}

	// Convert org to markdown using hybrid approach
	result, err := HybridOrgToMarkdown(string(orgContent), idMap)
	if err != nil {
		t.Fatalf("HybridOrgToMarkdown failed: %v", err)
	}

	expected := strings.TrimSpace(string(expectedMd))
	result = strings.TrimSpace(result)

	// Compare line by line for better error reporting
	expectedLines := strings.Split(expected, "\n")
	resultLines := strings.Split(result, "\n")

	if len(expectedLines) != len(resultLines) {
		t.Logf("Expected %d lines, got %d lines", len(expectedLines), len(resultLines))
	}

	maxLines := len(expectedLines)
	if len(resultLines) > maxLines {
		maxLines = len(resultLines)
	}

	for i := 0; i < maxLines; i++ {
		var expectedLine, resultLine string
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(resultLines) {
			resultLine = resultLines[i]
		}

		if expectedLine != resultLine {
			t.Errorf("Line %d mismatch:\nExpected: %q\nGot:      %q", i+1, expectedLine, resultLine)
		}
	}

	if result != expected {
		t.Errorf("Full conversion mismatch")
	}
}

func TestHybridMarkdownToOrg(t *testing.T) {
	// Read test fixtures
	mdContent, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatalf("Failed to read sample.md: %v", err)
	}

	expectedOrg, err := os.ReadFile("testdata/sample.org")
	if err != nil {
		t.Fatalf("Failed to read sample.org: %v", err)
	}

	// Create ID map for testing (matching the original test setup)
	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
		"987fcdeb-51a2-43f7-8123-456789abcdef": "987fcdeb-51a2-43f7-8123-456789abcdef",
	}

	// Convert markdown to org using hybrid approach
	result, err := HybridMarkdownToOrg(string(mdContent), idMap)
	if err != nil {
		t.Fatalf("HybridMarkdownToOrg failed: %v", err)
	}

	expected := strings.TrimSpace(string(expectedOrg))
	result = strings.TrimSpace(result)

	// Compare line by line for better error reporting
	expectedLines := strings.Split(expected, "\n")
	resultLines := strings.Split(result, "\n")

	if len(expectedLines) != len(resultLines) {
		t.Logf("Expected %d lines, got %d lines", len(expectedLines), len(resultLines))
	}

	maxLines := len(expectedLines)
	if len(resultLines) > maxLines {
		maxLines = len(resultLines)
	}

	for i := 0; i < maxLines; i++ {
		var expectedLine, resultLine string
		if i < len(expectedLines) {
			expectedLine = expectedLines[i]
		}
		if i < len(resultLines) {
			resultLine = resultLines[i]
		}

		if expectedLine != resultLine {
			t.Errorf("Line %d mismatch:\nExpected: %q\nGot:      %q", i+1, expectedLine, resultLine)
		}
	}

	if result != expected {
		t.Errorf("Full conversion mismatch")
	}
}

func TestHybridRoundtrip(t *testing.T) {
	// Read test fixture
	orgContent, err := os.ReadFile("testdata/sample.org")
	if err != nil {
		t.Fatalf("Failed to read sample.org: %v", err)
	}

	// Create ID map for testing (matching the original test setup)
	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
		"987fcdeb-51a2-43f7-8123-456789abcdef": "987fcdeb-51a2-43f7-8123-456789abcdef",
	}

	// Convert org -> md -> org using hybrid approach
	md, err := HybridOrgToMarkdown(string(orgContent), idMap)
	if err != nil {
		t.Fatalf("HybridOrgToMarkdown failed: %v", err)
	}

	orgRoundtrip, err := HybridMarkdownToOrg(md, idMap)
	if err != nil {
		t.Fatalf("HybridMarkdownToOrg failed: %v", err)
	}

	original := strings.TrimSpace(string(orgContent))
	result := strings.TrimSpace(orgRoundtrip)

	if original != result {
		// Compare line by line for better error reporting
		originalLines := strings.Split(original, "\n")
		resultLines := strings.Split(result, "\n")

		t.Logf("Original: %d lines, Result: %d lines", len(originalLines), len(resultLines))

		maxLines := len(originalLines)
		if len(resultLines) > maxLines {
			maxLines = len(resultLines)
		}

		for i := 0; i < maxLines; i++ {
			var originalLine, resultLine string
			if i < len(originalLines) {
				originalLine = originalLines[i]
			}
			if i < len(resultLines) {
				resultLine = resultLines[i]
			}

			if originalLine != resultLine {
				t.Errorf("Line %d mismatch:\nOriginal: %q\nResult:   %q", i+1, originalLine, resultLine)
			}
		}

		t.Error("Roundtrip conversion failed")
	}
}

func TestMarkerExtraction(t *testing.T) {
	idMap := map[string]string{
		"123e4567-e89b-12d3-a456-426614174000": "Related Note",
	}

	converter := NewHybridConverter(idMap)

	t.Run("OrgRoamLinks", func(t *testing.T) {
		input := "Here is a link: [[id:123e4567-e89b-12d3-a456-426614174000][My Note]] to a file"
		marked := converter.extractOrgRoamLinks(input)

		// Should have one marker
		if len(converter.markers) != 1 {
			t.Fatalf("Expected 1 marker, got %d", len(converter.markers))
		}

		// Marker should be in the output
		if !strings.Contains(marked, converter.markers[0].MarkerID) {
			t.Errorf("Marker ID not found in marked content")
		}

		// Original link should not be in marked content
		if strings.Contains(marked, "[[id:") {
			t.Errorf("Original link found in marked content")
		}

		// Convert should produce wikilink
		converted := converter.markers[0].Convert()
		expected := "[[Related Note|My Note]]"
		if converted != expected {
			t.Errorf("Expected %q, got %q", expected, converted)
		}
	})

	converter.reset()

	t.Run("Wikilinks", func(t *testing.T) {
		input := "Here is a link: [[Related Note|My Note]] to a file"
		marked := converter.extractWikilinks(input)

		// Should have one marker
		if len(converter.markers) != 1 {
			t.Fatalf("Expected 1 marker, got %d", len(converter.markers))
		}

		// Marker should be in the output
		if !strings.Contains(marked, converter.markers[0].MarkerID) {
			t.Errorf("Marker ID not found in marked content")
		}

		// Original link should not be in marked content (except in marker ID prefix)
		// Count occurrences of [[ - should only appear in the marker ID
		count := strings.Count(marked, "[[")
		if count != 0 {
			t.Errorf("Found %d occurrences of '[[' in marked content, expected 0", count)
		}

		// Convert should produce org-roam link
		converted := converter.markers[0].Convert()
		expected := "[[id:123e4567-e89b-12d3-a456-426614174000][My Note]]"
		if converted != expected {
			t.Errorf("Expected %q, got %q", expected, converted)
		}
	})
}
