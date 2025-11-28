package parser

// MarkdownNode represents a node in the markdown AST
type MarkdownNode struct {
	Type     string
	Content  string
	Children []*MarkdownNode
	Meta     map[string]string
}

// ParseMarkdown parses markdown content into an AST
func ParseMarkdown(content string) (*MarkdownNode, error) {
	// TODO: Implement markdown parser
	// This will handle:
	// - Headers (#, ##, ###, etc.)
	// - Front matter (YAML/TOML)
	// - Links [text](url) and [[wikilinks]]
	// - Tags/keywords
	// - Code blocks ```lang / ```
	// - Lists (-, *, 1., etc.)
	return nil, nil
}
