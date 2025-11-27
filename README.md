# notebridge

A bidirectional CLI tool for converting notes between org-mode/org-roam and markdown formats.

## Overview

**notebridge** enables seamless conversion between org-mode (org-roam) files used in Emacs and markdown files used in Obsidian and other markdown-based note-taking tools. The converter supports bidirectional conversion while preserving note structure, links, and metadata.

## Features (Planned)

- **Bidirectional Conversion**: Convert org-mode to markdown and vice versa
- **Org-roam Support**: Preserve org-roam specific features (IDs, backlinks, properties)
- **Link Handling**: Convert between org-mode links `[[id:...][description]]` and markdown wikilinks `[[...]]`
- **Metadata Preservation**: Handle front matter (YAML/TOML) and org properties
- **Batch Processing**: Convert entire directories of notes
- **Beautiful CLI**: Built with [Charm](https://charm.land/) tools for an elegant terminal experience

## Installation

### Prerequisites

- Go 1.21 or higher

### Build from Source

```bash
git clone https://github.com/gerunddev/notebridge.git
cd notebridge
go build -o notebridge ./cmd/notebridge
```

## Usage

### Convert org-mode to markdown

```bash
notebridge org-to-md notes/file.org
notebridge o2m notes/file.org --out notes/file.md
```

### Convert markdown to org-mode

```bash
notebridge md-to-org notes/file.md
notebridge m2o notes/file.md --out notes/file.org
```

### Batch conversion

```bash
notebridge org-to-md --dir notes/org --out notes/md
notebridge md-to-org --dir notes/md --out notes/org
```

## Project Structure

```
notebridge/
├── cmd/
│   └── notebridge/     # CLI entry point
│       └── main.go
├── internal/
│   ├── converter/      # Conversion logic
│   │   └── converter.go
│   └── parser/         # Format parsers
│       ├── org.go      # Org-mode parser
│       └── markdown.go # Markdown parser
├── go.mod
└── README.md
```

## Conversion Details

### Org-mode to Markdown

- Headers: `* Header` → `# Header`
- Properties drawer → YAML front matter
- Org links `[[id:uuid][Title]]` → `[[Title]]` with ID mapping
- Tags `:tag1:tag2:` → front matter tags
- Code blocks `#+BEGIN_SRC` → ` ```lang `

### Markdown to Org-mode

- Headers: `# Header` → `* Header`
- YAML front matter → Properties drawer
- Wikilinks `[[Title]]` → `[[id:uuid][Title]]` with ID generation
- Front matter tags → `:tag1:tag2:`
- Code blocks ` ```lang ` → `#+BEGIN_SRC lang`

## Development Status

This project is in early development. Core features are being actively built.

### Roadmap

- [ ] Org-mode parser
- [ ] Markdown parser
- [ ] Basic org-to-markdown conversion
- [ ] Basic markdown-to-org conversion
- [ ] Link resolution and mapping
- [ ] Metadata handling
- [ ] Batch processing
- [ ] Integrate Charm libraries (Bubble Tea, Lip Gloss)
- [ ] Configuration file support
- [ ] Watch mode for continuous conversion

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Charm](https://charm.land/) tools
- Inspired by the org-mode and Obsidian communities
