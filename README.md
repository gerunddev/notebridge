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
notebridge sync --dry-run  # Preview changes without modifying files
```

**Flags**:
- `--dry-run` - Preview mode that shows what would be synced without actually modifying files

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

### `notebridge install`

Generate system service files for automatic daemon startup.

```bash
notebridge install
```

Generates platform-specific service files:
- **macOS**: Creates launchd plist at `~/Library/LaunchAgents/com.notebridge.plist`
- **Linux**: Creates systemd user service at `~/.config/systemd/user/notebridge.service`

The command provides instructions for enabling and disabling the service after installation.

### `notebridge uninstall`

Remove system service files.

```bash
notebridge uninstall
```

Automatically handles cleanup:
- Stops and unloads/disables the service if running
- Removes the service file
- Reloads system service manager (systemd only)

## Configuration

**Location**: `~/.config/notebridge/config.json`

```json
{
  "org_dir": "/path/to/org-roam",
  "obsidian_dir": "/path/to/obsidian/vault",
  "log_file": "/tmp/notebridge.log",
  "state_file": "~/.config/notebridge/state.json",
  "interval": "30s",
  "resolution_strategy": "last-write-wins",
  "exclude_patterns": ["*.tmp", "drafts/*"]
}
```

**Configuration Options**:
- `org_dir`: Path to org-roam directory
- `obsidian_dir`: Path to Obsidian vault directory
- `log_file`: Path to log file (default: `/tmp/notebridge.log`)
- `state_file`: Path to state file (default: `~/.config/notebridge/state.json`)
- `interval`: Sync interval for daemon mode (e.g., "30s", "1m", "5m")
- `resolution_strategy`: Conflict resolution strategy (optional, default: "last-write-wins")
  - `last-write-wins`: Use the file with newer modification time
  - `use-org`: Always prefer org-roam version
  - `use-markdown`: Always prefer Obsidian version
- `exclude_patterns`: Glob patterns for files to exclude from sync (optional, default: [])

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

## Development

For information about the project structure, implementation details, and development roadmap, see [docs/project-management.md](docs/project-management.md).

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

MIT License - see LICENSE file for details

## Acknowledgments

- Built with [Charm](https://charm.land/) tools
- Inspired by the org-mode and Obsidian communities
