# Test Fixtures

This directory contains test fixtures for conversion testing.

## Files

- `sample.org` - Org-mode test file with comprehensive examples
- `sample.md` - Markdown equivalent of sample.org

## Coverage

The fixtures cover:

- **Headers**: Multiple levels (*, **, ***, etc.)
- **Tasks**: TODO, DONE with checkboxes
- **Scheduling**: SCHEDULED, DEADLINE, CLOSED dates
- **Priorities**: [#A], [#B], [#C]
- **Links**: Org-roam ID links and wikilinks
- **Properties**: :PROPERTIES: drawer and YAML front matter
- **Tags**: File tags and aliases
- **Code blocks**: Multiple languages
- **Quotes**: #+BEGIN_QUOTE and blockquotes
- **Lists**: Ordered and unordered

## ID Mappings

The test files use these org-roam IDs:

- `550e8400-e29b-41d4-a716-446655440000` - Main note ID
- `123e4567-e89b-12d3-a456-426614174000` - "Related Note"
- `987fcdeb-51a2-43f7-8123-456789abcdef` - Generic reference

## Usage

These fixtures are used in:
- `org_to_md_test.go` - Test org-to-markdown conversion
- `md_to_org_test.go` - Test markdown-to-org conversion
- `roundtrip_test.go` - Test bidirectional conversion integrity
