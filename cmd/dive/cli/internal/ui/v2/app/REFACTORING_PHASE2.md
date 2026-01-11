# Phase 2 Refactoring Status

## Completed

### 1. Custom Messages (`messages.go`)
Created semantic messages:
- `LayerChangedMsg` - sent when active layer changes
- `NodeToggledMsg` - sent when tree node is collapsed/expanded
- `PaneChangedMsg` - sent when active pane changes
- `LayerSelectionChangedMsg` - sent when layer is selected
- `TreeSelectionChangedMsg` - sent when tree node is selected
- `PaneFocusRequestMsg` - requests focus to a specific pane
- `RefreshTreeContentMsg` - requests tree content refresh

### 2. Pane Interface (`panes/base.go`)
Created `BasePane` interface and implementation:
- `SetSize(width, height int)`
- `Focus() / Blur()`
- `IsFocused() bool`
- `GetDimensions() (int, int)`

### 3. LayersPane Component (`panes/layers.go`)
Created independent tea.Model component:
- Handles its own viewport and content generation
- Responds to keyboard (up/down, j/k)
- Responds to mouse wheel and clicks
- Sends `LayerChangedMsg` when selection changes
- Manages layer index internally

### 4. TreePane Component (`panes/tree.go`)
Created independent tea.Model component:
- Handles its own viewport and content generation
- Responds to keyboard navigation
- Responds to mouse events
- Sends `NodeToggledMsg` when nodes are collapsed
- Manages tree index and scroll synchronization

## In Progress

### 5. Main Model Update
Need to update `Model` struct to use new pane components:

**Changes needed:**
```go
type Model struct {
    // ... existing fields ...

    // NEW: Use pane components instead of raw viewports
    layersPane panes.LayersPane
    treePane   panes.TreePane

    // REMOVE: These will be managed by panes
    // layersViewport viewport.Model  <- REMOVE
    // treeViewport   viewport.Model  <- REMOVE
    // treeIndex      int              <- REMOVE (managed by TreePane)

    // KEEP: imageViewport (still needed for static image details)
    imageViewport viewport.Model
}
```

### 6. Update Function
Route messages to appropriate panes:

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Route to focused pane
        switch m.activePane {
        case PaneLayer:
            newModel, cmd := m.layersPane.Update(msg)
            m.layersPane = newModel.(panes.LayersPane)
            cmds = append(cmds, cmd)
        case PaneTree:
            newModel, cmd := m.treePane.Update(msg)
            m.treePane = newModel.(panes.TreePane)
            cmds = append(cmds, cmd)
        case PaneImage:
            // Handle image scrolling directly
            if msg.String() == "up" || msg.String() == "k" {
                m.imageViewport.LineUp(1)
            } else if msg.String() == "down" || msg.String() == "j" {
                m.imageViewport.LineDown(1)
            }
        }

    case LayerChangedMsg:
        // React to layer changes
        m.updateTreeForCurrentLayer()

    case NodeToggledMsg:
        // Tree was toggled, content already updated by TreePane
        // Nothing to do here

    case tea.WindowSizeMsg:
        m.recalculateLayout()
        // Update pane sizes
        m.layersPane.SetSize(m.layout.LeftWidth, m.layout.LayersHeight)
        m.treePane.SetSize(m.layout.RightWidth, m.layout.TreeHeight)
    }

    return m, tea.Batch(cmds...)
}
```

### 7. Remove Flags
No longer need `updateLayers/updateTree/updateImage` flags because:
- Panes only receive messages when they are focused
- Each pane manages its own state
- No more double-update problem

## Benefits of Phase 2

1. **Separation of Concerns**: Each pane manages its own state
2. **No Double-Scroll**: Panes only update when focused
3. **Reactive**: Changes flow through custom messages
4. **Testable**: Each pane can be tested independently
5. **Extensible**: Easy to add new panes

## Next Steps to Complete Phase 2

1. ✅ Create messages.go
2. ✅ Create panes/base.go
3. ✅ Create panes/layers.go
4. ✅ Create panes/tree.go
5. ⏳ Update Model struct
6. ⏳ Update NewModel function
7. ⏳ Update Init function
8. ⏳ Update View function
9. ⏳ Update Update function with message routing
10. ⏳ Remove old viewport code
11. ⏳ Test everything works

## Architecture After Phase 2

```
┌─────────────────────────────────────────────┐
│              Main Model (Coordinator)        │
│  - Routes messages to active pane           │
│  - Responds to LayerChangedMsg              │
│  - Manages layout calculation               │
└─────────────────────────────────────────────┘
         │                    │               │
    ┌────▼────┐         ┌────▼────┐    ┌────▼────┐
    │LayersPane│         │ TreePane │    │Details  │
    │(tea.Model)│        │(tea.Model)│    │  Static  │
    └──────────┘         └──────────┘    └─────────┘
         │                    │
         └─────┬──────────────┘
               │
         LayerChangedMsg
```
