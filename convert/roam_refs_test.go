package convert

import (
	"strings"
	"testing"
)

func TestRoamRefsConversion(t *testing.T) {
	org := `:PROPERTIES:
:ID: test-id-123
:ROAM_REFS: https://example.com/article cite:smith2024
:END:
#+title: Test Note

Content here.`

	expected := `---
id: test-id-123
title: Test Note
refs:
  - https://example.com/article
  - cite:smith2024
---

Content here.`

	// Test org -> md
	result, err := OrgToMarkdown(org, map[string]string{})
	if err != nil {
		t.Fatalf("OrgToMarkdown failed: %v", err)
	}

	result = strings.TrimSpace(result)
	expected = strings.TrimSpace(expected)

	if result != expected {
		t.Errorf("Org->MD mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
	}

	// Test md -> org (roundtrip)
	resultOrg, err := MarkdownToOrg(expected, map[string]string{})
	if err != nil {
		t.Fatalf("MarkdownToOrg failed: %v", err)
	}

	resultOrg = strings.TrimSpace(resultOrg)
	org = strings.TrimSpace(org)

	if resultOrg != org {
		t.Errorf("MD->Org roundtrip mismatch.\nExpected:\n%s\n\nGot:\n%s", org, resultOrg)
	}
}

func TestRoamRefsSingleRef(t *testing.T) {
	org := `:PROPERTIES:
:ID: test-id
:ROAM_REFS: https://example.com
:END:
#+title: Web Clip`

	expected := `---
id: test-id
title: Web Clip
refs:
  - https://example.com
---`

	result, err := OrgToMarkdown(org, map[string]string{})
	if err != nil {
		t.Fatalf("OrgToMarkdown failed: %v", err)
	}

	result = strings.TrimSpace(result)
	expected = strings.TrimSpace(expected)

	if result != expected {
		t.Errorf("Conversion mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
	}
}
