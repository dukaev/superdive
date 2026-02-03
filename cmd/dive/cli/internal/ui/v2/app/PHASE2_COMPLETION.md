# Phase 2 Completion Plan

## Current Status

✅ Created components:
- `messages.go` - Custom semantic messages
- `layers_pane.go` - LayersPane component (independent)
- `tree_pane.go` - TreePane component (independent)

✅ Updated Model struct:
- Replaced `layersViewport` with `layersPane LayersPane`
- Replaced `treeViewport` with `treePane TreePane`
- Removed `treeIndex int` (managed by TreePane)

✅ Updated NewModel:
- Creates LayersPane and TreePane components
- Sets initial focus on LayersPane

⏳ IN PROGRESS: Update function needs complete rewrite

## What Update Function Should Do (Phase 2)

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    switch msg := msg.(type) {
    case tea.KeyMsg:
        // Route to focused pane
        switch m.activePane {
        case PaneLayer:
            newPane, cmd := m.layersPane.Update(msg)
            m.layersPane = newPane.(LayersPane)
            cmds = append(cmds, cmd)

        case PaneTree:
            newPane, cmd := m.treePane.Update(msg)
            m.treePane = newPane.(TreePane)
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
        // Layer changed - update tree
        m.updateTreeForCurrentLayer()

    case tea.WindowSizeMsg:
        m.recalculateLayout()
        // Update pane sizes
        m.layersPane.SetSize(m.layout.LeftWidth, m.layout.LayersHeight)
        m.treePane.SetSize(m.layout.RightWidth, m.layout.TreeHeight)

    case tea.MouseMsg:
        // Handle mouse - route to panes
        // Panes now handle their own mouse events!
        x, y := msg.X, msg.Y
        inLeftCol := x >= 0 && x < m.layout.LeftWidth
        inRightCol := x >= l.LeftWidth && x < m.width

        if inLeftCol {
            newPane, cmd := m.layersPane.Update(msg)
            m.layersPane = newPane.(LayersPane)
            cmds = append(cmds, cmd)
        } else if inRightCol {
            newPane, cmd := m.treePane.Update(msg)
            m.treePane = newPane.(TreePane)
            cmds = append(cmds, cmd)
        }
    }

    // Update help and filter
    // ... (existing code)

    return m, tea.Batch(cmds...)
}
```

## Key Changes from Phase 1

### BEFORE (Phase 1 - flags approach):
```go
updateLayers := false  // Flag
if inLeftCol {
    updateLayers = false
}
if updateLayers {
    m.layersViewport.Update(msg)  // Direct update
}
```

### AFTER (Phase 2 - component approach):
```go
if inLeftCol {
    newPane, cmd := m.layersPane.Update(msg)  // Component manages itself
    m.layersPane = newPane.(LayersPane)
}
// NO FLAGS NEEDED!
```

## Benefits of Phase 2

1. ✅ **No double-scroll**: Panes only update when focused
2. ✅ **No flags complexity**: Components manage their state
3. ✅ **Reactive**: LayerChangedMsg triggers tree update
4. ✅ **Testable**: Each pane can be tested independently
5. ✅ **Clean separation**: Main Model only routes messages

## Files to Modify

1. ✅ `model.go` - Update Model struct, NewModel
2. ⏳ `model.go` - Rewrite Update function
3. ⏳ `model.go` - Simplify/remove handleKeyPress, navigateUp/Down
4. ⏳ `model.go` - Update View to use panes
5. ⏳ `model_view.go` - Update renderLayersPane, renderTreePane

## Next Steps

Given the complexity and token limits, recommended approach:

**Option A**: I can create a complete replacement `model.go` file with all Phase 2 changes
**Option B**: Incremental updates of specific functions
**Option C**: Create a summary and let you complete the remaining updates

Which approach do you prefer?
