package diff

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/glamour"
	"github.com/gerunddev/notebridge/convert"
	"github.com/gerunddev/notebridge/state"
	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
)

// Format represents the output format for diffs
type Format int

const (
	// FormatMarkdown renders diffs in markdown (default)
	FormatMarkdown Format = iota
	// FormatOrg renders diffs in org-mode (future feature)
	FormatOrg
)

// Generate creates a diff between an org file and markdown file
// The format parameter determines how the diff is rendered
func Generate(orgPath, mdPath string, st *state.State, format Format) (string, error) {
	switch format {
	case FormatMarkdown:
		return generateMarkdown(orgPath, mdPath, st)
	case FormatOrg:
		return generateOrg(orgPath, mdPath, st)
	default:
		return "", fmt.Errorf("unsupported diff format: %d", format)
	}
}

// generateMarkdown converts both files to markdown and diffs them
func generateMarkdown(orgPath, mdPath string, st *state.State) (string, error) {
	// Read the org file
	orgContent, err := os.ReadFile(orgPath)
	if err != nil {
		return "", fmt.Errorf("failed to read org file: %w", err)
	}

	// Read the markdown file
	mdContent, err := os.ReadFile(mdPath)
	if err != nil {
		return "", fmt.Errorf("failed to read markdown file: %w", err)
	}

	// Convert org to markdown for comparison
	orgAsMd, err := convert.OrgToMarkdown(string(orgContent), st.IDMap)
	if err != nil {
		return "", fmt.Errorf("failed to convert org to markdown: %w", err)
	}

	// Generate unified diff
	orgFileName := filepath.Base(orgPath)
	mdFileName := filepath.Base(mdPath)

	edits := myers.ComputeEdits(span.URIFromPath(orgFileName), orgAsMd, string(mdContent))
	unified := gotextdiff.ToUnified(orgFileName, mdFileName, orgAsMd, edits)

	// Wrap in markdown diff code fence
	diffMarkdown := fmt.Sprintf("```diff\n%s```\n", unified)

	// Render with Glamour for beautiful terminal output
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		// Fallback to plain diff if glamour fails
		return diffMarkdown, nil
	}

	rendered, err := renderer.Render(diffMarkdown)
	if err != nil {
		// Fallback to plain diff if rendering fails
		return diffMarkdown, nil
	}

	return rendered, nil
}

// generateOrg converts both files to org and diffs them
// TODO: This will be implemented when org-mode diff rendering is added
func generateOrg(orgPath, mdPath string, st *state.State) (string, error) {
	return "", fmt.Errorf("org-mode diff format not yet implemented - will be added as a configurable option")
}
