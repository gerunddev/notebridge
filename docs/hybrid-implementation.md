# Hybrid Annotation Pattern Implementation

This document describes the implementation of Option 2 (Hybrid Annotation Pattern) from the conversion architecture options.

## Overview

The hybrid annotation pattern extracts custom features from the source content, replaces them with unique markers, performs standard conversion, then replaces the markers with converted features.

## Implementation Status

✅ **Completed** - Core marker system and link conversion

### What's Implemented

1. **Marker System** (`internal/convert/hybrid.go`)
   - `FeatureMarker` struct for tracking custom features
   - `HybridConverter` for managing markers and conversions
   - Unique marker generation using UUID-based IDs (e.g., `NOTEBR_MARKER_abc12345`)

2. **Org-roam Link Extraction** (`extractOrgRoamLinks`)
   - Extracts `[[id:uuid][description]]` and `[[id:uuid]]` links
   - Replaces with markers during conversion
   - Converts to wikilinks: `[[filename|description]]` or `[[filename]]`

3. **Wikilink Extraction** (`extractWikilinks`)
   - Extracts `[[filename|description]]` and `[[filename]]` links
   - Replaces with markers during conversion
   - Converts to org-roam links: `[[id:uuid][description]]` or `[[id:uuid]]`

4. **Hybrid Converters**
   - `HybridOrgToMarkdown` - Org → Markdown with marker-based link conversion
   - `HybridMarkdownToOrg` - Markdown → Org with marker-based link conversion

5. **Test Coverage**
   - `TestHybridOrgToMarkdown` - Full org-to-markdown conversion ✅
   - `TestHybridMarkdownToOrg` - Full markdown-to-org conversion ✅
   - `TestHybridRoundtrip` - Bidirectional conversion integrity ✅
   - `TestMarkerExtraction` - Marker creation and application ✅

## Architecture

```
┌─────────────────┐
│  Original File  │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────┐
│  Step 1: Extract Custom Features│
│  - Org-roam IDs                 │
│  - Wikilinks                    │
│  Replace with NOTEBR_MARKER_xxx │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│  Step 2: Standard Conversion    │
│  Using existing converters      │
│  (Markers preserved as text)    │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────────────────────┐
│  Step 3: Apply Markers          │
│  NOTEBR_MARKER_xxx → [[link]]   │
└────────┬────────────────────────┘
         │
         ▼
┌─────────────────┐
│  Converted File │
└─────────────────┘
```

## Key Design Decisions

### 1. Using Existing Converters (Not go-org Library)

**Decision**: Use existing manual converters instead of go-org library for Step 2

**Rationale**:
- `go-org` library doesn't provide a markdown writer - it only has HTML output
- Would require implementing a custom markdown renderer for go-org AST (Option 1)
- Our existing converters already work and pass all tests
- Marker system provides the main architectural benefit

**Future**: Could implement custom go-org markdown renderer if needed for better edge case handling

### 2. Content-Based Markers

**Decision**: Use unique string markers (e.g., `NOTEBR_MARKER_abc12345`) instead of position tracking

**Benefits**:
- Survives content transformations (reordering, line breaks, etc.)
- No need to track line numbers or positions
- Simple string replacement for marker application
- Libraries treat markers as plain text, preserving them

**Implementation**:
```go
markerID := fmt.Sprintf("NOTEBR_MARKER_%s", uuid.New().String()[:8])
```

### 3. Closure-Based Conversion Functions

**Decision**: Store conversion logic as closures in `FeatureMarker.Convert`

**Benefits**:
- Captures context (UUID, description, idMap) at extraction time
- Deferred execution - conversion happens after library processing
- Clean separation between extraction and conversion

**Example**:
```go
marker := c.createMarker("org-roam-id", match, context, func() string {
    filename, ok := c.idMap[uuid]
    if !ok {
        filename = uuid
    }
    if description != "" {
        return fmt.Sprintf("[[%s|%s]]", filename, description)
    }
    return fmt.Sprintf("[[%s]]", filename)
})
```

## What's Not Yet Implemented

The following features from the original manual converters are **not yet** using the marker system:

1. **Task Metadata** (TODO/DONE, priorities)
   - Currently handled by manual converters
   - Could extract with markers in future

2. **Scheduling Info** (SCHEDULED, DEADLINE, CLOSED)
   - Currently handled by manual converters
   - Could extract with markers in future

3. **Code Blocks**
   - Currently handled by manual converters
   - Standard feature, likely doesn't need markers

4. **Blockquotes**
   - Currently handled by manual converters
   - Standard feature, likely doesn't need markers

5. **Headers**
   - Currently handled by manual converters
   - Standard feature, likely doesn't need markers

## Usage Example

```go
import "github.com/gerunddev/notebridge/internal/convert"

// Create ID mapping
idMap := map[string]string{
    "123e4567-e89b-12d3-a456-426614174000": "My Note",
}

// Convert org to markdown
markdown, err := convert.HybridOrgToMarkdown(orgContent, idMap)
if err != nil {
    log.Fatal(err)
}

// Convert markdown to org
org, err := convert.HybridMarkdownToOrg(markdownContent, idMap)
if err != nil {
    log.Fatal(err)
}
```

## Test Results

All 23 test suites passing:

- ✅ Original converters (org_to_md, md_to_org)
- ✅ Hybrid converters (org→md, md→org)
- ✅ Roundtrip tests (preserves content through bidirectional conversion)
- ✅ Marker extraction tests
- ✅ Individual feature tests (headers, tasks, links, properties)

## Performance Considerations

The hybrid approach adds minimal overhead:

1. **Two passes** through content (extract markers, apply markers)
   - Marker extraction: O(n) regex replacements
   - Marker application: O(m) string replacements where m = number of markers

2. **Memory**: Stores markers in slice (typically small - only custom features)

3. **Overall**: Still O(n) complexity, with small constant factor overhead

## Future Enhancements

1. **Add Task Marker Support**
   - Extract TODO/DONE with scheduling info
   - Convert entire task blocks atomically

2. **Custom go-org Markdown Writer**
   - Implement `org.Writer` interface for markdown output
   - Use go-org for parsing, custom writer for rendering
   - Better edge case handling for org-mode features

3. **Goldmark Org-mode Renderer**
   - Implement goldmark renderer that outputs org-mode
   - Use goldmark for markdown parsing
   - Better edge case handling for markdown features

4. **Additional Custom Features**
   - Org drawers (LOGBOOK, etc.)
   - Obsidian callouts → Org special blocks
   - Org tables ↔ Markdown tables (if needed)

## Conclusion

The hybrid annotation pattern successfully provides:

- ✅ Clean separation of concerns (custom vs standard features)
- ✅ Resilient conversion (markers survive transformations)
- ✅ Extensible architecture (easy to add new features)
- ✅ Test coverage (all conversions verified)
- ✅ Backward compatibility (existing tests still pass)

The implementation validates Option 2 as a practical approach for bidirectional format conversion with custom features.
