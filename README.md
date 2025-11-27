# NoteBridge

A CLI tool for bidirectional sync between org-roam and Obsidian markdown files.

**Warning!** This project is not even alpha. It's heavily vibe coded and not yet used in anger. If you've stumbled upon this project, use at your own risk with notes you care about.

## Overview

**Purpose**: Keep notes in sync between Emacs org-roam and Obsidian, allowing seamless switching between editors while maintaining a single source of truth.

notebridge monitors your org-roam and Obsidian directories, automatically converting and syncing files bidirectionally. It handles format conversion, link mapping, metadata preservation, and conflict resolution.

**Language**: Go (using [Charm](https://charm.land/) libraries for TUI/styling)

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

## Commands

### `notebridge daemon`

Run background sync loop.

```bash
notebridge daemon --interval 30s
```

**Flags**:
- `--interval` - sync frequency (default: 30s)

### `notebridge sync`

One-shot manual sync.

```bash
notebridge sync
```

### `notebridge status`

Display sync state.

```bash
notebridge status
```

**Output**:
- Last sync time
- Pending changes
- Recent errors
- Files in conflict

## Configuration

**Location**: `~/.notebridge/config.json`

```json
{
  "org_dir": "/path/to/org-roam",
  "obsidian_dir": "/path/to/obsidian/vault",
  "log_file": "/tmp/notebridge.log",
  "state_file": "~/.notebridge/state.json",
  "interval": "30s"
}
```

## State Tracking

**Location**: `~/.notebridge/state.json`

### Strategy: mtime + content hash (hybrid)

1. Check mtime first (fast path)
2. If mtime changed, compute SHA256 hash
3. Compare hash to detect actual content changes

### State file structure

```json
{
  "files": {
    "notes/foo.org": {
      "mtime": 1699900000,
      "hash": "sha256:abc123...",
      "paired_with": "notes/foo.md"
    }
  },
  "id_map": {
    "org-id-uuid-here": "filename-without-extension"
  }
}
```

## Conflict Resolution

**Strategy**: Last-write-wins

1. Check both org and obsidian versions
2. If only one changed → sync that direction
3. If both changed → compare mtime, newer wins
4. Log all conflicts to log file for review

## Format Conversion

### Links

| Org-roam | Obsidian |
|----------|----------|
| `[[id:uuid][Description]]` | `[[filename\|Description]]` |
| `[[id:uuid]]` | `[[filename]]` |

ID-to-filename mapping maintained in state file.

### Tasks

| Org | Obsidian Tasks |
|-----|----------------|
| `* TODO Task` | `- [ ] Task` |
| `* DONE Task` | `- [x] Task` |
| `SCHEDULED: <2024-01-15>` | `⏳ 2024-01-15` |
| `DEADLINE: <2024-01-15>` | `📅 2024-01-15` |
| `[#A]` | `high` priority |
| `[#B]` | `medium` priority |
| `[#C]` | `low` priority |
| `CLOSED: [2024-01-15]` | `✅ 2024-01-15` |

### Metadata

| Org | Obsidian |
|-----|----------|
| `:PROPERTIES:` drawer | YAML frontmatter |
| `:ROAM_ALIASES:` | `aliases:` in frontmatter |
| `:ROAM_REFS:` | `refs:` in frontmatter |
| Heading tags `:tag1:tag2:` | `tags:` in frontmatter |

### Structure

| Org | Obsidian |
|-----|----------|
| `* Heading` | `# Heading` |
| `** Subheading` | `## Subheading` |
| `#+BEGIN_SRC lang` | ``` lang ``` |
| `#+BEGIN_QUOTE` | `>` blockquote |

**Callouts** (12 types + aliases):

| Org Block | Obsidian Callout |
|-----------|------------------|
| `#+BEGIN_NOTE` | `> [!note]` |
| `#+BEGIN_ABSTRACT` | `> [!abstract]` (aliases: summary, tldr) |
| `#+BEGIN_INFO` | `> [!info]` |
| `#+BEGIN_TODO` | `> [!todo]` |
| `#+BEGIN_TIP` | `> [!tip]` (aliases: hint, important) |
| `#+BEGIN_SUCCESS` | `> [!success]` (aliases: check, done) |
| `#+BEGIN_QUESTION` | `> [!question]` (aliases: help, faq) |
| `#+BEGIN_WARNING` | `> [!warning]` (aliases: caution, attention) |
| `#+BEGIN_FAILURE` | `> [!failure]` (aliases: fail, missing) |
| `#+BEGIN_DANGER` | `> [!danger]` (alias: error) |
| `#+BEGIN_BUG` | `> [!bug]` |
| `#+BEGIN_EXAMPLE` | `> [!example]` |

### Embeds

| Obsidian | Org (converted) |
|----------|-----------------|
| `![[note]]` | `# EMBED: note` (comment) |
| `![[image.png]]` | `[[file:image.png]]` |

### Features without equivalents

Preserved as comments when converting:
- Obsidian block references (`^block-id`)
- Obsidian callouts (`> [!NOTE]`)
- Org clock entries (`CLOCK:`)

## Implementation

notebridge uses a **hybrid annotation pattern** for conversion: custom features (org-roam IDs, wikilinks) are extracted and marked with unique placeholders, standard conversion is performed, then markers are replaced with converted features. This provides clean separation between custom org-roam/Obsidian features and standard markdown/org-mode syntax.

**Libraries used**:
- **[go-org](https://github.com/niklasfasching/go-org)**: Org-mode parsing (future enhancement)
- **[goldmark](https://github.com/yuin/goldmark)**: Markdown parsing with frontmatter support
- **[gopkg.in/yaml.v3](https://gopkg.in/yaml.v3)**: YAML frontmatter handling
- **[google/uuid](https://github.com/google/uuid)**: Org-roam ID generation

Current implementation uses manual line-by-line conversion with proper library-based YAML and UUID handling. The hybrid marker system is in place for bidirectional link conversion, with full test coverage validating roundtrip conversion integrity.

**See**: `doc/conversion-options.md` for architectural decisions and `doc/hybrid-implementation.md` for implementation details.

## Logging

**Location**: Configurable, default `/tmp/notebridge.log`

**Format**:
```
2024-01-15 10:30:00 [INFO] Sync started
2024-01-15 10:30:01 [INFO] foo.org → foo.md (org newer)
2024-01-15 10:30:01 [WARN] Conflict: bar.md - both modified, org wins (newer mtime)
2024-01-15 10:30:02 [ERROR] Failed to parse baz.org: invalid property drawer
2024-01-15 10:30:02 [INFO] Sync complete: 3 files synced, 1 conflict, 1 error
```

Designed for `tail -f` monitoring.

## Project Structure

```
notebridge/
├── cmd/
│   └── notebridge/
│       └── main.go
├── internal/
│   ├── config/         # Configuration management
│   │   └── config.go
│   ├── state/          # State tracking (mtime, hash, id_map)
│   │   └── state.go
│   ├── sync/           # Sync logic and conflict resolution
│   │   └── sync.go
│   ├── convert/        # Format conversion
│   │   ├── org_to_md.go
│   │   └── md_to_org.go
│   └── parser/         # Format parsers
│       ├── org.go      # Org-mode parser
│       └── markdown.go # Markdown parser
├── go.mod
├── go.sum
└── README.md
```

## Dependencies

### Charm Libraries

- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)**: TUI framework for interactive CLI experiences
- **[Bubbles](https://github.com/charmbracelet/bubbles)**: Pre-built TUI components (spinners, progress bars, text inputs)
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)**: Styling and layout for beautiful terminal output
- **[Huh](https://github.com/charmbracelet/huh)**: Forms and prompts for CLI configuration
- **[Glamour](https://github.com/charmbracelet/glamour)**: Markdown rendering in terminal
- **[Log](https://github.com/charmbracelet/log)**: Structured logging for daemon operations
- **[Wish](https://github.com/charmbracelet/wish)**: SSH server capabilities

### Other Dependencies

- **[fsnotify](https://github.com/fsnotify/fsnotify)**: File system watching for daemon mode
- **[pkg/sftp](https://github.com/pkg/sftp)**: SSH/SFTP client for remote files
- **golang.org/x/crypto**: SSH client functionality
- Standard library: JSON, file I/O, hashing (SHA256)

See **Implementation** section for conversion library details.

## Development Status

This project is in early development. Core features are being actively built.

### Roadmap

**Phase 1: Core Sync**
- [ ] Configuration management (`~/.notebridge/config.json`)
- [ ] State tracking (mtime + SHA256 hash)
- [x] Org-mode parser (line-by-line with library support)
- [x] Markdown parser (line-by-line with library support)
- [x] Basic org-to-markdown conversion
- [x] Basic markdown-to-org conversion
- [x] ID-to-filename mapping
- [ ] Conflict resolution (last-write-wins)

**Phase 2: Format Conversion**
- [x] Link conversion (org-roam IDs ↔ Obsidian wikilinks)
- [x] Task conversion (TODO/DONE ↔ checkboxes)
- [x] Metadata handling (properties ↔ front matter)
- [x] ROAM_REFS property (URLs, citation keys)
- [x] Scheduled/Deadline dates
- [x] Priority levels
- [x] Tags and aliases
- [x] Code blocks and quotes
- [x] Callouts (Obsidian) ↔ Special blocks (Org)
- [ ] Embeds handling

**Phase 3: Daemon & CLI**
- [ ] `daemon` command with configurable interval
- [ ] `sync` command (one-shot)
- [ ] `status` command with TUI
- [ ] Structured logging
- [ ] Error handling and recovery

**Phase 4: Advanced Features**
- [ ] `watch` - real-time file watcher mode
- [ ] `install` - generate launchd/systemd service
- [ ] `diff <file>` - show pending changes for a file
- [ ] Selective sync (include/exclude patterns)
- [ ] Dry-run mode
- [ ] SSH/Remote support for syncing across machines

## Future Considerations

- Real-time file watching with fsnotify
- System service installation (launchd/systemd)
- Per-file diff preview
- Pattern-based selective sync
- Remote/SSH support for distributed note-taking

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Charm](https://charm.land/) tools
- Inspired by the org-mode and Obsidian communities
