# NoteBridge - Project Management

This document tracks the development status, implementation details, and project structure of NoteBridge.

## Development Status

All planned phases have been completed! The project includes full bidirectional sync between org-roam and Obsidian with comprehensive format conversion.

### Roadmap

**Phase 1: Core Sync** ✓
- [x] Configuration management (`~/.config/notebridge/config.json`)
- [x] State tracking (mtime + SHA256 hash)
- [x] Org-mode parser (line-by-line with library support)
- [x] Markdown parser (line-by-line with library support)
- [x] Basic org-to-markdown conversion
- [x] Basic markdown-to-org conversion
- [x] ID-to-filename mapping
- [x] Conflict resolution (last-write-wins)

**Phase 2: Format Conversion** ✓
- [x] Link conversion (org-roam IDs ↔ Obsidian wikilinks)
- [x] Task conversion (TODO/DONE ↔ checkboxes)
- [x] Metadata handling (properties ↔ front matter)
- [x] ROAM_REFS property (URLs, citation keys)
- [x] Scheduled/Deadline dates
- [x] Priority levels
- [x] Tags and aliases
- [x] Code blocks and quotes
- [x] Callouts (Obsidian) ↔ Special blocks (Org)
- [x] Embeds handling

**Phase 3: Daemon & CLI** ✓
- [x] `daemon` command with configurable interval
- [x] A way to stop the sync daemon
- [x] `sync` command (one-shot)
- [x] `status` command with TUI
- [x] Structured logging
- [x] Error handling and recovery

**Phase 4: Enhanced TUI Experience** ✓
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

**Phase 5: Advanced Features** ✓
- [x] Configurable resolution strategy
  - Config option for default resolution (use-org, use-markdown, last-write-wins)
  - Allows daemon/sync mode to use strategies other than last-write-wins
- [x] `install` - generate launchd/systemd service files
- [x] `uninstall` - remove system service files
- [x] Selective sync (exclude patterns)
- [x] Dry-run mode (`--dry-run` flag)

## Future Considerations

Potential enhancements for future development:

- Per-directory or per-file resolution rules
- Include patterns (complement to exclude patterns)
- Remote sync over SSH/SFTP
- Webhook integration for triggering syncs
- Web UI for monitoring and configuration
- Plugin system for custom converters
- Support for other note-taking formats (Notion, Evernote, etc.)

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

The state file tracks:
- **File metadata**: Modification time and content hash for change detection
- **File pairing**: Which org files correspond to which markdown files
- **ID mapping**: org-roam UUIDs mapped to filenames for link conversion

## Implementation

notebridge uses a **hybrid annotation pattern** for conversion: custom features (org-roam IDs, wikilinks) are extracted and marked with unique placeholders, standard conversion is performed, then markers are replaced with converted features. This provides clean separation between custom org-roam/Obsidian features and standard markdown/org-mode syntax.

### Conversion Libraries

- **[go-org](https://github.com/niklasfasching/go-org)**: Org-mode parsing (future enhancement)
- **[goldmark](https://github.com/yuin/goldmark)**: Markdown parsing with frontmatter support
- **[gopkg.in/yaml.v3](https://gopkg.in/yaml.v3)**: YAML frontmatter handling
- **[google/uuid](https://github.com/google/uuid)**: Org-roam ID generation

### Conversion Strategy

Current implementation uses manual line-by-line conversion with proper library-based YAML and UUID handling. The hybrid marker system is in place for bidirectional link conversion, with full test coverage validating roundtrip conversion integrity.

**See**: `doc/conversion-options.md` for architectural decisions and `doc/hybrid-implementation.md` for implementation details.

### Testing

The project maintains comprehensive test coverage:
- Unit tests for all conversion functions
- Roundtrip tests (org → md → org and md → org → md)
- Idempotence tests (repeated conversions produce same result)
- Integration tests for sync operations
- Test fixtures covering all supported formats

## Project Structure

```
notebridge/
├── cmd/
│   └── notebridge/
│       └── main.go              # CLI entry point
├── internal/
│   ├── config/                  # Configuration management
│   │   ├── config.go
│   │   └── config_test.go
│   ├── state/                   # State tracking (mtime, hash, id_map)
│   │   ├── state.go
│   │   └── state_test.go
│   ├── sync/                    # Sync logic and conflict resolution
│   │   ├── sync.go
│   │   └── sync_test.go
│   ├── convert/                 # Format conversion
│   │   ├── org_to_md.go
│   │   ├── md_to_org.go
│   │   ├── *_test.go
│   │   └── testdata/
│   ├── parser/                  # Format parsers
│   │   ├── org.go
│   │   └── markdown.go
│   ├── daemon/                  # Background daemon functionality
│   │   └── daemon.go
│   ├── logger/                  # Structured logging
│   │   └── logger.go
│   └── tui/                     # Terminal UI components
│       ├── status.go
│       ├── browse.go
│       └── dashboard.go
├── docs/                        # Documentation
│   ├── project-management.md
│   ├── conversion-options.md
│   └── hybrid-implementation.md
├── go.mod
├── go.sum
└── README.md
```

### Key Components

- **cmd/notebridge**: CLI application entry point
- **internal/config**: Configuration loading and validation
- **internal/state**: File state tracking and change detection
- **internal/sync**: Synchronization engine and conflict resolution
- **internal/convert**: Bidirectional format conversion (org ↔ markdown)
- **internal/daemon**: Background daemon for continuous sync
- **internal/logger**: Structured logging with configurable output
- **internal/tui**: Terminal UI components using Bubble Tea

## Architecture Decisions

### Why Line-by-Line Conversion?

Instead of using a full AST-based parser, notebridge uses line-by-line conversion for several reasons:

1. **Simplicity**: Easier to understand and maintain
2. **Robustness**: Handles malformed documents gracefully
3. **Performance**: Fast processing for large files
4. **Flexibility**: Easy to add new conversion rules

### Why Hybrid Marker Pattern?

The hybrid annotation pattern allows:

1. Clean separation of concerns (custom vs. standard features)
2. Reliable roundtrip conversion
3. Extensibility for new features
4. Testability with clear input/output expectations

### Why mtime + Hash?

The hybrid state tracking approach provides:

1. **Performance**: Fast mtime checks catch most cases
2. **Accuracy**: Hash comparison eliminates false positives
3. **Reliability**: Detects actual content changes, not just metadata updates

## Contributing

See the main [README](../README.md#contributing) for contribution guidelines.

When contributing to NoteBridge:

1. **Add tests**: All new features must include comprehensive tests
2. **Update docs**: Keep documentation in sync with code changes
3. **Follow patterns**: Match existing code style and architecture
4. **Test conversions**: Verify roundtrip conversion integrity
5. **Handle edge cases**: Consider malformed input and error conditions
