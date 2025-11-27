# NoteBridge

A CLI tool for bidirectional sync between org-roam and Obsidian markdown files.

**Warning!** This project is not even alpha. It's heavily vibe coded and not yet used in anger. If you've stumbled upon this project, use at your own risk with notes you care about.

## Overview

**Purpose**: Keep notes in sync between Emacs org-roam and Obsidian, allowing seamless switching between editors. Notebridge monitors your org-roam and Obsidian directories, automatically converting and syncing files bidirectionally. It handles format conversion, link mapping, metadata preservation, and conflict resolution.

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

### `notebridge start`

Start daemon in background.

```bash
notebridge start --interval 30s
```

**Flags**:
- `--interval` - sync frequency (default: 30s)

### `notebridge stop`

Stop the running daemon.

```bash
notebridge stop
```

### `notebridge daemon`

Run daemon in foreground with live TUI dashboard.

```bash
notebridge daemon --interval 30s
```

Shows the same live dashboard as the `dashboard` command, but runs the sync loop in the current process. Useful for development and debugging.

**Flags**:
- `--interval` - sync frequency (default: 30s)

### `notebridge sync`

One-shot manual sync.

```bash
notebridge sync
```

### `notebridge status`

Display sync state with interactive TUI.

```bash
notebridge status
```

**Features**:
- Live-updating table of pending changes
- Interactive conflict resolution
- Keyboard navigation (j/k or arrows)
- Auto-refresh every 2 seconds

### `notebridge browse`

Browse all tracked files with interactive TUI.

```bash
notebridge browse
```

**Features**:
- Table view showing all tracked files with status icons
- Diff preview mode (press enter or 'd')
- Interactive conflict resolution from diff view
- Keyboard navigation

### `notebridge dashboard`

View live status of running daemon.

```bash
notebridge dashboard
```

Connects to a running daemon (started with `start`) and displays real-time dashboard with:
- Daemon status, PID, and uptime
- Last sync time and files synced count
- Live log tail (scrollable with j/k)
- Auto-refresh every 2 seconds

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
- Org clock entries (`CLOCK:`)

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

## Development Status

This project is in early development. Core features are being actively built.

### Roadmap

**Phase 1: Core Sync**
- [x] Configuration management (`~/.config/notebridge/config.json`)
- [x] State tracking (mtime + SHA256 hash)
- [x] Org-mode parser (line-by-line with library support)
- [x] Markdown parser (line-by-line with library support)
- [x] Basic org-to-markdown conversion
- [x] Basic markdown-to-org conversion
- [x] ID-to-filename mapping
- [x] Conflict resolution (last-write-wins)

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
- [x] `daemon` command with configurable interval
- [x] A way to stop the sync daemon
- [x] `sync` command (one-shot)
- [x] `status` command with TUI
- [x] Structured logging
- [x] Error handling and recovery

**Phase 4: Enhanced TUI Experience**
- [x] Enhanced status command with live updates (Bubble Tea + Bubbles)
  - Live-updating display with auto-refresh
  - Interactive table showing files with pending changes
  - Spinner during directory scanning
  - Keyboard navigation through file lists
- [x] Sync progress indicator (Bubbles)
  - Spinner during sync operation
  - Real-time sync status updates
  - Clean formatted output
- [x] Interactive conflict resolution
  - Prompt user to choose conflict resolution strategy
  - Options: use org, use markdown, last-write-wins, skip file
  - Conflicts shown on single line in table
  - Auto-refresh after resolution
- [x] Live daemon status dashboard (Bubble Tea)
  - Real-time dashboard with daemon uptime
  - Last sync time and files synced count
  - Live log tail display
  - Press 'q' to quit view
  - Available via `dashboard` command (viewer) or `daemon` command (foreground mode)
- [x] Interactive file browser (Bubbles)
  - List of tracked files with sync status
  - Interactive table with all files
  - Diff preview mode (press enter/d)
  - Navigate with keyboard (j/k or arrows)

**Phase 5: Advanced Features**
- [ ] Configurable resolution strategy
  - Allow daemon/sync mode to use strategies other than last-write-wins
  - Config option for default resolution (use-org, use-markdown, prompt, last-write-wins)
  - Per-directory or per-file resolution rules
- [ ] `watch` - real-time file watcher mode
- [ ] `install` - generate launchd/systemd service
- [ ] `diff <file>` - show pending changes for a file
- [ ] Selective sync (include/exclude patterns)
- [ ] Dry-run mode
- [ ] SSH/Remote support for syncing across machines
- [ ] Move state from a JSON file to use [Skate](https://github.com/charmbracelet/skate)
- [ ] Troubleshoot daemon not always stopping gracefully


## Configuration

**Location**: `~/.config/notebridge/config.json`

```json
{
  "org_dir": "/path/to/org-roam",
  "obsidian_dir": "/path/to/obsidian/vault",
  "log_file": "/tmp/notebridge.log",
  "state_file": "~/.config/notebridge/state.json",
  "interval": "30s"
}
```

## State Tracking

**Location**: `~/.config/notebridge/state.json`

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

## Implementation

notebridge uses a **hybrid annotation pattern** for conversion: custom features (org-roam IDs, wikilinks) are extracted and marked with unique placeholders, standard conversion is performed, then markers are replaced with converted features. This provides clean separation between custom org-roam/Obsidian features and standard markdown/org-mode syntax.

**Libraries used**:
- **[go-org](https://github.com/niklasfasching/go-org)**: Org-mode parsing (future enhancement)
- **[goldmark](https://github.com/yuin/goldmark)**: Markdown parsing with frontmatter support
- **[gopkg.in/yaml.v3](https://gopkg.in/yaml.v3)**: YAML frontmatter handling
- **[google/uuid](https://github.com/google/uuid)**: Org-roam ID generation

Current implementation uses manual line-by-line conversion with proper library-based YAML and UUID handling. The hybrid marker system is in place for bidirectional link conversion, with full test coverage validating roundtrip conversion integrity.

**See**: `doc/conversion-options.md` for architectural decisions and `doc/hybrid-implementation.md` for implementation details.

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Charm](https://charm.land/) tools
- Inspired by the org-mode and Obsidian communities
