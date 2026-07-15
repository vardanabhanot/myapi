package ui

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

// makeCollectionContent builds the Collections sidebar tab: a tree whose
// branches are collections ("c:<i>") and leaves are request snapshots
// ("r:<i>:<j>"). Indexes are rebuilt on every Refresh.
func (g *gui) makeCollectionContent() *fyne.Container {
	g.collections = core.LoadCollections()

	g.collectionTree = widget.NewTree(
		func(uid widget.TreeNodeID) []widget.TreeNodeID {
			if uid == "" {
				ids := make([]widget.TreeNodeID, len(g.collections))
				for i := range g.collections {
					ids[i] = fmt.Sprintf("c:%d", i)
				}
				return ids
			}

			var ci int
			if _, err := fmt.Sscanf(uid, "c:%d", &ci); err != nil || ci >= len(g.collections) {
				return nil
			}

			ids := make([]widget.TreeNodeID, len(g.collections[ci].Requests))
			for i := range g.collections[ci].Requests {
				ids[i] = fmt.Sprintf("r:%d:%d", ci, i)
			}
			return ids
		},
		func(uid widget.TreeNodeID) bool {
			return uid == "" || strings.HasPrefix(uid, "c:")
		},
		func(branch bool) fyne.CanvasObject {
			label := widget.NewLabel("Collection")
			label.Truncation = fyne.TextTruncateEllipsis
			options := newTappableIcon(theme.MoreHorizontalIcon(), func() {})

			return container.NewBorder(nil, nil, nil, container.NewPadded(options), label)
		},
		func(uid widget.TreeNodeID, branch bool, o fyne.CanvasObject) {
			row := o.(*fyne.Container)
			label := row.Objects[0].(*widget.Label)
			options := row.Objects[1].(*fyne.Container).Objects[0].(*tappableIcon)

			if branch {
				var ci int
				if _, err := fmt.Sscanf(uid, "c:%d", &ci); err != nil || ci >= len(g.collections) {
					return
				}

				col := g.collections[ci]
				label.SetText(col.Name)
				options.onTapped = func() {
					// Re-locate by pointer at tap time: the captured index can
					// be stale after deletes/moves, the pointer cannot.
					i := g.collectionIndex(col)
					if i < 0 {
						return
					}

					rename := fyne.NewMenuItem("Rename", func() {
						g.renameDialog("Rename Collection", col.Name, func(name string) {
							if name == "" {
								return
							}
							col.Name = name
							g.saveCollections()
							g.collectionTree.Refresh()
						})
					})

					// ponytail: moves reorder index-based UIDs, so branch
					// open/closed state stays with the position, not the
					// collection. Cosmetic; stable UIDs if it ever grates.
					up := fyne.NewMenuItem("Move Up", func() {
						g.collections[i], g.collections[i-1] = g.collections[i-1], g.collections[i]
						g.saveCollections()
						g.collectionTree.Refresh()
					})
					up.Disabled = i == 0

					down := fyne.NewMenuItem("Move Down", func() {
						g.collections[i], g.collections[i+1] = g.collections[i+1], g.collections[i]
						g.saveCollections()
						g.collectionTree.Refresh()
					})
					down.Disabled = i == len(g.collections)-1

					del := fyne.NewMenuItem("Delete", func() {
						dialog.NewConfirm("Delete Collection", "Delete \""+col.Name+"\" and its requests?", func(confirmed bool) {
							if !confirmed {
								return
							}

							if i := g.collectionIndex(col); i >= 0 {
								g.collections = append(g.collections[:i], g.collections[i+1:]...)
							}

							g.saveCollections()
							g.collectionTree.Refresh()
						}, *g.Window).Show()
					})

					showIconMenu(options, rename, up, down, del)
				}
				return
			}

			var ci, ri int
			if _, err := fmt.Sscanf(uid, "r:%d:%d", &ci, &ri); err != nil ||
				ci >= len(g.collections) || ri >= len(g.collections[ci].Requests) {
				return
			}

			col := g.collections[ci]
			request := col.Requests[ri]
			label.SetText(entryTitle(request))
			options.onTapped = func() {
				i := requestIndex(col, request)
				if i < 0 {
					return
				}

				rename := fyne.NewMenuItem("Rename", func() {
					g.renameDialog("Rename Request", request.Name, func(name string) {
						request.Name = name

						// Push the name into open linked tabs too, or their
						// next send-sync snapshots the old name back over it.
						for _, t := range g.tabs {
							if t.colEntry != request || t.request == nil {
								continue
							}

							t.request.Name = name
							if t.item != nil {
								title := entryTitle(t.request)
								if t.request.IsDirty {
									title += " *"
								}
								t.item.Text = title
							}
						}

						g.saveCollections()
						g.collectionTree.Refresh()
						g.doctabs.Refresh()
					})
				})

				up := fyne.NewMenuItem("Move Up", func() {
					col.Requests[i], col.Requests[i-1] = col.Requests[i-1], col.Requests[i]
					g.saveCollections()
					g.collectionTree.Refresh()
				})
				up.Disabled = i == 0

				down := fyne.NewMenuItem("Move Down", func() {
					col.Requests[i], col.Requests[i+1] = col.Requests[i+1], col.Requests[i]
					g.saveCollections()
					g.collectionTree.Refresh()
				})
				down.Disabled = i == len(col.Requests)-1

				remove := fyne.NewMenuItem("Remove", func() {
					dialog.NewConfirm("Remove Request", "Remove this request from \""+col.Name+"\"?", func(confirmed bool) {
						if !confirmed {
							return
						}

						if i := requestIndex(col, request); i >= 0 {
							col.Requests = append(col.Requests[:i], col.Requests[i+1:]...)
						}

						g.saveCollections()
						g.collectionTree.Refresh()
					}, *g.Window).Show()
				})

				showIconMenu(options, rename, up, down, remove)
			}
		},
	)

	g.collectionTree.OnSelected = func(uid widget.TreeNodeID) {
		var ci, ri int
		if _, err := fmt.Sscanf(uid, "r:%d:%d", &ci, &ri); err != nil {
			// Branch tap: this collection becomes the creation context for
			// the new-request button; the highlight stays as the focus cue.
			if _, err := fmt.Sscanf(uid, "c:%d", &ci); err == nil && ci < len(g.collections) {
				g.focusedCollection = g.collections[ci]
			}
			return
		}

		defer g.collectionTree.UnselectAll()

		if ci >= len(g.collections) || ri >= len(g.collections[ci].Requests) {
			return
		}

		col := g.collections[ci]
		g.focusedCollection = col

		// Open as a linked tab: the copy gets its own identity (history,
		// tab map), but sends sync back into the collection entry.
		request := col.Requests[ri].Clone()
		request.ID = core.NewRequestID()
		request.IsDirty = true

		tabItem := g.makeTab(request)
		t := g.tabs[request.ID]
		t.collection = col
		t.colEntry = col.Requests[ri]
		g.doctabs.Append(tabItem)
		g.doctabs.Select(tabItem)
	}

	// New request lands in the focused collection, VS Code style; with no
	// collection focused it is just a plain detached tab.
	newReqBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		col := g.validFocusedCollection()

		if col == nil {
			tabItem := g.makeTab(nil)
			g.doctabs.Append(tabItem)
			g.doctabs.Select(tabItem)
			return
		}

		entry := &core.Request{Method: "GET"}
		col.Requests = append(col.Requests, entry)
		g.saveCollections()

		for i, c := range g.collections {
			if c == col {
				g.collectionTree.OpenBranch(fmt.Sprintf("c:%d", i))
				break
			}
		}
		g.collectionTree.Refresh()

		request := &core.Request{ID: core.NewRequestID(), Method: "GET", IsDirty: true}
		tabItem := g.makeTab(request)
		t := g.tabs[request.ID]
		t.collection = col
		t.colEntry = entry
		g.doctabs.Append(tabItem)
		g.doctabs.Select(tabItem)
	})
	newReqBtn.Importance = widget.HighImportance

	newColBtn := widget.NewButtonWithIcon("", theme.FolderNewIcon(), func() {
		nameEntry := widget.NewEntry()
		nameEntry.SetPlaceHolder("Collection name")

		dialog.NewForm("New Collection", "Create", "Cancel", []*widget.FormItem{
			widget.NewFormItem("Name", nameEntry),
		}, func(confirmed bool) {
			if !confirmed || nameEntry.Text == "" {
				return
			}

			g.collections = append(g.collections, &core.Collection{Name: nameEntry.Text})
			g.saveCollections()
			g.collectionTree.Refresh()
		}, *g.Window).Show()
	})
	newColBtn.Importance = widget.LowImportance

	header := container.NewBorder(nil, nil,
		container.NewPadded(sectionHeader("Collections")),
		container.NewPadded(container.NewHBox(newColBtn, newReqBtn)),
		nil,
	)

	return container.NewBorder(header, nil, nil, nil, g.collectionTree)
}

// validFocusedCollection re-checks the focused collection still exists —
// it may have been deleted since it was focused.
func (g *gui) validFocusedCollection() *core.Collection {
	for _, c := range g.collections {
		if c == g.focusedCollection {
			return c
		}
	}

	return nil
}

// collectionIndex finds a collection by pointer identity; -1 when deleted.
func (g *gui) collectionIndex(col *core.Collection) int {
	for i, c := range g.collections {
		if c == col {
			return i
		}
	}

	return -1
}

// requestIndex finds an entry inside a collection by pointer identity.
func requestIndex(col *core.Collection, r *core.Request) int {
	for i, req := range col.Requests {
		if req == r {
			return i
		}
	}

	return -1
}

// entryTitle prefers the user-given name over the method+path fallback.
func entryTitle(r *core.Request) string {
	if r.Name == "" {
		return tabTitle(r.Method, r.URL)
	}

	name := r.Name
	if ru := []rune(name); len(ru) > 22 {
		name = string(ru[:22]) + "…"
	}

	return r.Method + " " + name
}

// renameDialog is the one prefilled single-field form both rename menus use.
func (g *gui) renameDialog(title, current string, apply func(string)) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(current)

	dialog.NewForm(title, "Save", "Cancel", []*widget.FormItem{
		widget.NewFormItem("Name", nameEntry),
	}, func(confirmed bool) {
		if !confirmed {
			return
		}

		apply(strings.TrimSpace(nameEntry.Text))
	}, *g.Window).Show()
}

// syncCollectionEntry mirrors a sent request back into its linked collection
// entry. Called from the send goroutine after a successful request.
func (g *gui) syncCollectionEntry(request *core.Request) {
	t := g.tabs[request.ID]
	if t == nil || t.collection == nil || t.colEntry == nil {
		return
	}

	if !t.collection.UpdateRequest(t.colEntry, request) {
		// Entry was removed from the collection while this tab was open
		t.collection, t.colEntry = nil, nil
		return
	}

	g.saveCollections()
	fyne.Do(func() {
		g.collectionTree.Refresh()
	})
}

// addToCollectionDialog snapshots the request into a picked (or newly
// created) collection.
func (g *gui) addToCollectionDialog(request *core.Request) {
	var d dialog.Dialog

	addTo := func(col *core.Collection) {
		entry := request.Clone()
		col.Requests = append(col.Requests, entry)

		// Like save-as: the tab now mirrors its new collection entry, so
		// later sends keep the snapshot fresh.
		if t := g.tabs[request.ID]; t != nil {
			t.collection = col
			t.colEntry = entry
		}

		g.saveCollections()
		g.collectionTree.Refresh()
		d.Hide()
	}

	nameEntry := widget.NewEntry()
	nameEntry.SetPlaceHolder("New collection name")
	createBtn := widget.NewButtonWithIcon("", theme.ContentAddIcon(), func() {
		if nameEntry.Text == "" {
			return
		}

		col := &core.Collection{Name: nameEntry.Text}
		g.collections = append(g.collections, col)
		addTo(col)
	})
	nameEntry.OnSubmitted = func(string) { createBtn.OnTapped() }

	list := widget.NewList(
		func() int {
			return len(g.collections)
		},
		func() fyne.CanvasObject {
			label := widget.NewLabel("Collection")
			label.Truncation = fyne.TextTruncateEllipsis

			return container.NewBorder(nil, nil, widget.NewIcon(theme.FolderIcon()), nil, label)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			o.(*fyne.Container).Objects[0].(*widget.Label).SetText(g.collections[i].Name)
		},
	)
	list.OnSelected = func(i widget.ListItemID) {
		list.UnselectAll()
		addTo(g.collections[i])
	}

	content := container.NewBorder(
		container.NewBorder(nil, nil, nil, createBtn, nameEntry),
		nil, nil, nil,
		list,
	)

	d = dialog.NewCustom("Add to Collection", "Cancel", content, *g.Window)
	d.Resize(fyne.NewSize(340, 400))
	d.Show()
}

func (g *gui) saveCollections() {
	if err := core.SaveCollections(g.collections); err != nil {
		dialog.NewError(err, *g.Window).Show()
	}
}
