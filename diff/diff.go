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

// DefaultFormat determines the default diff format based on sync direction
// Uses the destination format: org→md shows markdown, md→org shows org
func DefaultFormat(orgPath, mdPath string) (Format, error) {
	orgInfo, err := os.Stat(orgPath)
	if err != nil {
		return FormatMarkdown, fmt.Errorf("failed to stat org file: %w", err)
	}
	mdInfo, err := os.Stat(mdPath)
	if err != nil {
		return FormatMarkdown, fmt.Errorf("failed to stat md file: %w", err)
	}

	// If org is newer, sync is org→md, so show markdown format
	// If md is newer, sync is md→org, so show org format
	if orgInfo.ModTime().After(mdInfo.ModTime()) {
		return FormatMarkdown, nil
	}
	return FormatOrg, nil
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

	// Determine which file is newer to show diff in correct direction
	orgInfo, err := os.Stat(orgPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat org file: %w", err)
	}
	mdInfo, err := os.Stat(mdPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat md file: %w", err)
	}

	// Generate unified diff with newer file as "new" side
	orgFileName := filepath.Base(orgPath)
	mdFileName := filepath.Base(mdPath)

	var unified string

	if orgInfo.ModTime().After(mdInfo.ModTime()) {
		// Org is newer: show md → org (md is old, org is new)
		edits := myers.ComputeEdits(span.URIFromPath(mdFileName), string(mdContent), orgAsMd)
		unified = fmt.Sprint(gotextdiff.ToUnified(mdFileName, orgFileName, string(mdContent), edits))
	} else {
		// Md is newer: show org → md (org is old, md is new)
		edits := myers.ComputeEdits(span.URIFromPath(orgFileName), orgAsMd, string(mdContent))
		unified = fmt.Sprint(gotextdiff.ToUnified(orgFileName, mdFileName, orgAsMd, edits))
	}

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
func generateOrg(orgPath, mdPath string, st *state.State) (string, error) {
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

	// Convert markdown to org for comparison
	mdAsOrg, err := convert.MarkdownToOrg(string(mdContent), st.IDMap)
	if err != nil {
		return "", fmt.Errorf("failed to convert markdown to org: %w", err)
	}

	// Determine which file is newer to show diff in correct direction
	orgInfo, err := os.Stat(orgPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat org file: %w", err)
	}
	mdInfo, err := os.Stat(mdPath)
	if err != nil {
		return "", fmt.Errorf("failed to stat md file: %w", err)
	}

	// Generate unified diff with newer file as "new" side
	orgFileName := filepath.Base(orgPath)
	mdFileName := filepath.Base(mdPath)

	var unified string

	if orgInfo.ModTime().After(mdInfo.ModTime()) {
		// Org is newer: show md → org (md is old, org is new)
		edits := myers.ComputeEdits(span.URIFromPath(mdFileName), mdAsOrg, string(orgContent))
		unified = fmt.Sprint(gotextdiff.ToUnified(mdFileName, orgFileName, mdAsOrg, edits))
	} else {
		// Md is newer: show org → md (org is old, md is new)
		edits := myers.ComputeEdits(span.URIFromPath(orgFileName), string(orgContent), mdAsOrg)
		unified = fmt.Sprint(gotextdiff.ToUnified(orgFileName, mdFileName, string(orgContent), edits))
	}

	// Wrap in diff code fence for proper syntax highlighting (+ in green, - in red)
	diffOrg := fmt.Sprintf("```diff\n%s```\n", unified)

	// Render with Glamour
	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(120),
	)
	if err != nil {
		// Fallback to plain diff if glamour fails
		return diffOrg, nil
	}

	rendered, err := renderer.Render(diffOrg)
	if err != nil {
		// Fallback to plain diff if rendering fails
		return diffOrg, nil
	}

	return rendered, nil
}
