package convert

import (
	"strings"
	"testing"
)

func TestCalloutConversion(t *testing.T) {
	tests := []struct {
		name     string
		org      string
		markdown string
	}{
		{
			name: "note callout",
			org: `#+BEGIN_NOTE
This is a note.
#+END_NOTE`,
			markdown: `> [!note]
> This is a note.
`,
		},
		{
			name: "tip callout",
			org: `#+BEGIN_TIP
This is a tip.
#+END_TIP`,
			markdown: `> [!tip]
> This is a tip.
`,
		},
		{
			name: "warning callout",
			org: `#+BEGIN_WARNING
This is a warning.
#+END_WARNING`,
			markdown: `> [!warning]
> This is a warning.
`,
		},
		{
			name: "multi-line callout",
			org: `#+BEGIN_NOTE
Line 1
Line 2
Line 3
#+END_NOTE`,
			markdown: `> [!note]
> Line 1
> Line 2
> Line 3
`,
		},
	}

	idMap := map[string]string{}

	for _, tt := range tests {
		t.Run(tt.name+" org->md", func(t *testing.T) {
			result, err := OrgToMarkdown(tt.org, idMap)
			if err != nil {
				t.Fatalf("OrgToMarkdown failed: %v", err)
			}

			expected := strings.TrimSpace(tt.markdown)
			result = strings.TrimSpace(result)

			if result != expected {
				t.Errorf("Conversion mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
			}
		})

		t.Run(tt.name+" md->org", func(t *testing.T) {
			result, err := MarkdownToOrg(tt.markdown, idMap)
			if err != nil {
				t.Fatalf("MarkdownToOrg failed: %v", err)
			}

			expected := strings.TrimSpace(tt.org)
			result = strings.TrimSpace(result)

			if result != expected {
				t.Errorf("Conversion mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
			}
		})
	}
}

func TestCalloutWithTitle(t *testing.T) {
	md := `> [!tip] Pro Tip
> This is helpful advice.`

	result, err := MarkdownToOrg(md, map[string]string{})
	if err != nil {
		t.Fatalf("MarkdownToOrg failed: %v", err)
	}

	// Should convert to org block with title as first line
	expected := `#+BEGIN_TIP
Pro Tip
This is helpful advice.
#+END_TIP`

	result = strings.TrimSpace(result)
	expected = strings.TrimSpace(expected)

	if result != expected {
		t.Errorf("Callout with title mismatch.\nExpected:\n%s\n\nGot:\n%s", expected, result)
	}
}
