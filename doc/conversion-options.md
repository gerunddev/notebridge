# Conversion Architecture Options

This document explores different architectural patterns for implementing bidirectional conversion between org-roam and Obsidian markdown formats using parsing libraries.

## Context

We initially implemented a custom line-by-line parser for both org-mode and markdown. While this works well for our specific use case, we want to evaluate using established libraries like `go-org` (org-mode parser) and `goldmark` (markdown parser) to handle the parsing complexity.

The challenge: These libraries are designed for **parsing → rendering to HTML**, not for **bidirectional format conversion**.

## Option 1: Library-as-Parser + Custom AST Walker

### Architecture

```
┌─────────────┐
│  Org File   │
└──────┬──────┘
       │ parse
       ▼
┌─────────────────┐
│  go-org AST     │  ← Library handles parsing complexity
└──────┬──────────┘
       │ walk
       ▼
┌──────────────────────┐
│  Custom Converter    │  ← We control conversion logic
│  - Detect org-roam   │
│  - Handle tasks      │
│  - Map to Markdown   │
└──────┬───────────────┘
       │ build
       ▼
┌─────────────────┐
│  goldmark AST   │
└──────┬──────────┘
       │ render
       ▼
┌─────────────┐
│ Markdown    │
└─────────────┘
```

### Implementation Sketch

```go
type ASTConverter struct {
    idMap map[string]string
}

func (c *ASTConverter) ConvertOrgToMarkdown(orgContent string) (string, error) {
    // Parse with go-org
    orgDoc := org.New().Parse(strings.NewReader(orgContent))

    // Walk AST and convert nodes
    mdDoc := c.walkOrgAST(orgDoc.Nodes)

    // Render with goldmark
    var buf bytes.Buffer
    renderer := markdown.NewRenderer()
    renderer.Render(&buf, mdDoc)

    return buf.String(), nil
}

func (c *ASTConverter) walkOrgAST(nodes []org.Node) goldmark.Node {
    // Map each org node type to markdown node type
    // Handle custom org-roam features during traversal
}
```

### Pros
- Libraries handle parsing edge cases (escaping, nesting, etc.)
- We control conversion semantics
- Single pass through AST
- Type-safe node handling

### Cons
- Need to learn both AST structures deeply
- Complex mapping between different node types
- May need to handle library-specific quirks
- Breaking changes in libraries affect us

### Use Case
Best when you need to support the **full** feature set of both formats and want robust parsing.

---

## Option 2: Hybrid Annotation Pattern ⭐ (RECOMMENDED)

### Architecture

```
┌─────────────────┐
│  Original File  │
└────────┬────────┘
         │
         ▼
┌─────────────────────────┐
│  Step 1: Pre-scan       │
│  Extract Custom Features│
│  - org-roam IDs         │
│  - Task metadata        │
│  - Scheduling           │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Replace with Markers   │
│  [[id:abc]] → [MARK_1]  │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Step 2: Library Convert│
│  go-org or goldmark     │
└────────┬────────────────┘
         │
         ▼
┌─────────────────────────┐
│  Step 3: Post-process   │
│  [MARK_1] → [[wikilink]]│
└────────┬────────────────┘
         │
         ▼
┌─────────────────┐
│  Converted File │
└─────────────────┘
```

### Implementation Sketch

```go
type FeatureMarker struct {
    MarkerID   string              // Unique placeholder: "NOTEBR_MARKER_abc123"
    Feature    string              // Feature type: "org-roam-id", "task-scheduled"
    Original   string              // Original syntax
    Convert    func() string       // Conversion function
}

type HybridConverter struct {
    markers []FeatureMarker
    idMap   map[string]string
}

func (c *HybridConverter) OrgToMarkdown(input string) (string, error) {
    // Step 1: Extract and replace custom features with markers
    marked, markers := c.extractAndMark(input)

    // Step 2: Use library for standard conversion
    converted := c.libraryConvert(marked)

    // Step 3: Replace markers with converted features
    final := c.applyMarkers(converted, markers)

    return final, nil
}

func (c *HybridConverter) extractAndMark(input string) (string, []FeatureMarker) {
    markers := []FeatureMarker{}
    result := input

    // Replace org-roam links with markers
    re := regexp.MustCompile(`\[\[id:([^\]]+)\](?:\[([^\]]+)\])?\]`)
    result = re.ReplaceAllStringFunc(result, func(match string) string {
        marker := c.createMarker("org-roam-id", match)
        markers = append(markers, marker)
        return marker.MarkerID
    })

    // Replace SCHEDULED/DEADLINE with markers
    // Replace TODO/DONE with markers
    // etc.

    return result, markers
}

func (c *HybridConverter) createMarker(featureType, original string) FeatureMarker {
    markerID := fmt.Sprintf("NOTEBR_MARKER_%s", uuid.New().String()[:8])

    return FeatureMarker{
        MarkerID: markerID,
        Feature:  featureType,
        Original: original,
        Convert:  func() string {
            // Conversion logic based on feature type
            switch featureType {
            case "org-roam-id":
                return c.convertOrgLink(original)
            case "task-scheduled":
                return c.convertScheduled(original)
            // ... etc
            }
        },
    }
}

func (c *HybridConverter) applyMarkers(converted string, markers []FeatureMarker) string {
    result := converted
    for _, marker := range markers {
        replacement := marker.Convert()
        result = strings.Replace(result, marker.MarkerID, replacement, 1)
    }
    return result
}
```

### Key Insight: Content-Based Markers

Instead of tracking line numbers (which change during conversion), we use **unique placeholder strings**:

```
Before:  "Here's a link: [[id:abc123][My Note]] to a file"
Marked:  "Here's a link: NOTEBR_MARKER_xyz789 to a file"
Library: "Here's a link: NOTEBR_MARKER_xyz789 to a file"  (preserved!)
After:   "Here's a link: [[My Note]] to a file"
```

The library treats markers as plain text, preserving them through conversion.

### Pros
- Simpler - libraries do the heavy lifting
- Only handle "delta" from standard formats
- No location tracking needed
- Easier to maintain and extend
- More resilient to library changes

### Cons
- Two-pass processing (small overhead)
- Need unique marker format that won't collide
- Must ensure markers survive library processing

### Use Case
**Best for our use case** - we have well-defined custom features (org-roam IDs, task metadata) that need special handling, but want robust standard parsing.

---

## Option 3: Decorator/Middleware Pattern

### Architecture

```go
type Converter interface {
    Convert(input string) (string, error)
}

// Base converter using libraries
type StandardOrgToMarkdown struct {
    parser   *org.Parser
    renderer *markdown.Renderer
}

func (s *StandardOrgToMarkdown) Convert(input string) (string, error) {
    // Use go-org and goldmark for standard conversion
}

// Decorator for org-roam features
type OrgRoamDecorator struct {
    base  Converter
    idMap map[string]string
}

func (d *OrgRoamDecorator) Convert(input string) (string, error) {
    // Pre-process: extract org-roam IDs
    ids := d.extractIDs(input)

    // Delegate to base converter
    output, err := d.base.Convert(input)
    if err != nil {
        return "", err
    }

    // Post-process: convert IDs to wikilinks
    output = d.convertLinks(output, ids)

    return output, nil
}

// Decorator for task metadata
type TaskMetadataDecorator struct {
    base Converter
}

func (t *TaskMetadataDecorator) Convert(input string) (string, error) {
    // Pre-process: extract task metadata
    tasks := t.extractTasks(input)

    // Delegate to base
    output, err := t.base.Convert(input)

    // Post-process: add task metadata in target format
    output = t.addTaskMetadata(output, tasks)

    return output, nil
}

// Chain decorators
converter := &TaskMetadataDecorator{
    base: &OrgRoamDecorator{
        base: &StandardOrgToMarkdown{},
        idMap: idMap,
    },
}

result, err := converter.Convert(orgContent)
```

### Pros
- Clean separation of concerns
- Each decorator handles one feature
- Composable and testable
- Easy to add/remove features
- Follows well-known design pattern

### Cons
- Multiple passes through content
- Order of decorators matters
- Harder to share state between decorators
- Each decorator needs pre/post processing logic

### Use Case
Good when you want to **add features incrementally** or support **multiple conversion targets**.

---

## Option 4: Custom AST Extension Pattern

### Architecture

```go
// Extend go-org's AST with custom node types
type OrgRoamIDNode struct {
    org.Node
    ID          string
    Description string
}

type OrgTaskNode struct {
    org.Node
    State      string  // TODO, DONE
    Priority   string  // A, B, C
    Scheduled  time.Time
    Deadline   time.Time
}

// Extend goldmark's AST with custom nodes
type WikilinkNode struct {
    ast.BaseInline
    Target string
    Alias  string
}

type ObsidianTaskNode struct {
    ast.BaseBlock
    Checked   bool
    Scheduled time.Time
    Deadline  time.Time
    Priority  string
}

// Custom converter that understands both
type ExtendedConverter struct {
    orgParser  *org.Parser
    mdRenderer *goldmark.Markdown
}

func (c *ExtendedConverter) ConvertNode(orgNode org.Node) ast.Node {
    switch n := orgNode.(type) {
    case *OrgRoamIDNode:
        return &WikilinkNode{
            Target: c.idMap[n.ID],
            Alias:  n.Description,
        }
    case *OrgTaskNode:
        return &ObsidianTaskNode{
            Checked:   n.State == "DONE",
            Scheduled: n.Scheduled,
            Deadline:  n.Deadline,
            Priority:  convertPriority(n.Priority),
        }
    default:
        // Delegate standard nodes to library
        return c.standardConvert(orgNode)
    }
}
```

### Pros
- Best of both worlds - leverage libraries for standard features
- Type-safe custom features
- Single unified AST structure
- Full control over rendering

### Cons
- Most complex implementation
- Requires deep library integration
- Need to maintain custom node types
- May break on library updates
- Requires understanding library internals

### Use Case
Best when you need **extensive customization** and want to **contribute back** to the libraries.

---

## Comparison Matrix

| Criteria | Option 1: AST Walker | Option 2: Hybrid Annotation ⭐ | Option 3: Decorator | Option 4: AST Extension |
|----------|---------------------|-------------------------------|---------------------|------------------------|
| **Complexity** | High | Medium | Medium | Very High |
| **Library Integration** | Deep | Shallow | Medium | Very Deep |
| **Maintainability** | Medium | High | High | Low |
| **Performance** | Good (1 pass) | Good (2 pass) | Medium (N passes) | Best (1 pass) |
| **Extensibility** | Medium | High | Very High | Medium |
| **Library Update Risk** | High | Low | Medium | Very High |
| **Learning Curve** | Steep | Gentle | Moderate | Very Steep |
| **Best For** | Full format support | Targeted custom features | Multiple targets | Contributing to libraries |

---

## Recommendation: Option 2 (Hybrid Annotation)

For the notebridge use case, **Option 2** is recommended because:

1. **Well-defined custom features**: Org-roam IDs, task scheduling, priorities are discrete, identifiable features
2. **Libraries handle complexity**: go-org and goldmark deal with edge cases in parsing
3. **Simple to maintain**: Only need to update marker extraction/application for new features
4. **Resilient**: Less dependent on library internals
5. **Proven pattern**: Similar to how template engines handle placeholders

### Implementation Strategy

1. Start with org-roam ID conversion (highest impact)
2. Add task metadata handling
3. Add scheduling/deadline conversion
4. Add priority conversion
5. Handle remaining edge cases

Each feature is isolated and testable.

---

## Current Implementation

As of this writing, notebridge uses a **custom line-by-line parser** that:
- ✅ Passes all tests including roundtrip conversion
- ✅ Handles org-roam specific features correctly
- ✅ Is simple and maintainable
- ✅ Uses `gopkg.in/yaml.v3` for YAML front matter (replaced manual parsing)

The custom implementation works well but could benefit from library parsing for:
- Better handling of edge cases (nested structures, escaping)
- Support for more org-mode and markdown features
- Reduced maintenance burden

---

## Future Work

If we implement Option 2, we could:
1. Use `github.com/niklasfasching/go-org` for org-mode parsing
2. Use `github.com/yuin/goldmark` with `go.abhg.dev/goldmark/frontmatter` for markdown
3. Maintain custom marker-based conversion for:
   - Org-roam IDs ↔ Wikilinks
   - Task metadata (SCHEDULED, DEADLINE, CLOSED)
   - Priority levels
   - Any other org-roam or Obsidian specific features

This hybrid approach gives us the best of both worlds.
