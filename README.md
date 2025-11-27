# notebridge

A bidirectional CLI tool for converting notes between org-mode/org-roam and markdown formats.

## Overview

**notebridge** enables seamless conversion between org-mode (org-roam) files used in Emacs and markdown files used in Obsidian and other markdown-based note-taking tools. The converter supports bidirectional conversion while preserving note structure, links, and metadata.

## Architecture

notebridge is designed as a daemon-first tool with CLI configuration capabilities:

- **Daemon Mode**: Background process that watches directories and automatically converts files
- **CLI Tool**: Configuration interface and one-off command execution
- **SSH Support**: Access and convert files on remote systems

## Dependencies

### Charm Libraries

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: TUI framework for interactive CLI experiences
- **[Bubbles](https://github.com/charmbracelet/bubbles)**: Pre-built TUI components (spinners, progress bars, text inputs, file pickers)
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout for beautiful terminal output
- **[Huh](https://github.com/charmbracelet/huh)**: Forms and prompts for CLI configuration wizards
- **[Glamour](https://github.com/charmbracelet/glamour)**: Markdown rendering in the terminal for previewing conversions
- **[Log](https://github.com/charmbracelet/log)**: Structured logging for daemon operations
- **[Wish](https://github.com/charmbracelet/wish)**: SSH server capabilities (for future remote access features)

### Other Dependencies

- **[fsnotify](https://github.com/fsnotify/fsnotify)**: File system watching for daemon mode
- **[pkg/sftp](https://github.com/pkg/sftp)**: SSH/SFTP client for remote file access
- **golang.org/x/crypto**: SSH client functionality

## Features (Planned)

- **Bidirectional Conversion**: Convert org-mode to markdown and vice versa
- **Org-roam Support**: Preserve org-roam specific features (IDs, backlinks, properties)
- **Link Handling**: Convert between org-mode links `[[id:...][description]]` and markdown wikilinks `[[...]]`
- **Metadata Preservation**: Handle front matter (YAML/TOML) and org properties
- **Daemon Mode**: Background process with directory watching for automatic conversion
- **SSH/Remote Support**: Access and convert files on remote systems via SSH/SFTP
- **Interactive CLI**: Configuration wizards and file previews
- **Batch Processing**: Convert entire directories of notes
- **Beautiful Terminal UI**: Built with [Charm](https://charm.land/) tools for an elegant experience

## Installation

### Prerequisites

- Go 1.21 or higher

### Build from Source

```bash
git clone https://github.com/gerunddev/notebridge.git
cd notebridge
go mod download  # Download dependencies
go build -o notebridge ./cmd/notebridge
```

## Usage

### One-off Conversions

```bash
# Convert single file org-mode to markdown
notebridge org-to-md notes/file.org
notebridge o2m notes/file.org --out notes/file.md

# Convert single file markdown to org-mode
notebridge md-to-org notes/file.md
notebridge m2o notes/file.md --out notes/file.org

# Batch conversion
notebridge org-to-md --dir notes/org --out notes/md
notebridge md-to-org --dir notes/md --out notes/org
```

### Daemon Mode

```bash
# Interactive configuration
notebridge config

# Start daemon with directory watching
notebridge start --watch ~/notes/org --output ~/notes/md

# Check daemon status
notebridge status

# Stop daemon
notebridge stop
```

### Remote/SSH Support

```bash
# Configure SSH remote
notebridge config add-remote my-server user@host:/path/to/notes

# Convert remote file
notebridge org-to-md --remote my-server:file.org

# Sync and convert remote directory
notebridge sync --remote my-server --watch
```

### Preview

```bash
# Preview conversion in terminal before saving
notebridge preview file.org
notebridge preview file.md --format org
```

## Project Structure

```
notebridge/
├── cmd/
│   ├── notebridge/     # CLI entry point
│   │   └── main.go
│   └── notebridged/    # Daemon process (planned)
│       └── main.go
├── internal/
│   ├── daemon/         # Daemon logic and file watching (planned)
│   ├── ssh/            # SSH/SFTP client for remote files (planned)
│   ├── ui/             # Bubble Tea TUI components (planned)
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

**Phase 1: Core Conversion**
- [ ] Org-mode parser
- [ ] Markdown parser
- [ ] Basic org-to-markdown conversion
- [ ] Basic markdown-to-org conversion
- [ ] Link resolution and mapping
- [ ] Metadata handling (properties ↔ front matter)

**Phase 2: CLI & User Experience**
- [ ] Integrate Charm libraries (Bubble Tea, Lip Gloss, Huh)
- [ ] Interactive configuration wizard
- [ ] File preview with Glamour
- [ ] Batch processing
- [ ] Configuration file support

**Phase 3: Daemon Mode**
- [ ] File system watching with fsnotify
- [ ] Daemon process (start/stop/status)
- [ ] Background conversion service
- [ ] Structured logging with Charm Log

**Phase 4: Remote/SSH Support**
- [ ] SSH client implementation
- [ ] SFTP file access
- [ ] Remote configuration management
- [ ] Sync capabilities

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Charm](https://charm.land/) tools
- Inspired by the org-mode and Obsidian communities
