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
					showIconMenu(options, fyne.NewMenuItem("Delete", func() {
						dialog.NewConfirm("Delete Collection", "Delete \""+col.Name+"\" and its requests?", func(confirmed bool) {
							if !confirmed {
								return
							}

							for i, c := range g.collections {
								if c == col {
									g.collections = append(g.collections[:i], g.collections[i+1:]...)
									break
								}
							}

							g.saveCollections()
							g.collectionTree.Refresh()
						}, *g.Window).Show()
					}))
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
			label.SetText(tabTitle(request.Method, request.URL))
			options.onTapped = func() {
				showIconMenu(options, fyne.NewMenuItem("Remove", func() {
					dialog.NewConfirm("Remove Request", "Remove this request from \""+col.Name+"\"?", func(confirmed bool) {
						if !confirmed {
							return
						}

						for i, r := range col.Requests {
							if r == request {
								col.Requests = append(col.Requests[:i], col.Requests[i+1:]...)
								break
							}
						}

						g.saveCollections()
						g.collectionTree.Refresh()
					}, *g.Window).Show()
				}))
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
		t := g.tabs[request.ID+".json"]
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
		t := g.tabs[request.ID+".json"]
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

// syncCollectionEntry mirrors a sent request back into its linked collection
// entry. Called from the send goroutine after a successful request.
func (g *gui) syncCollectionEntry(request *core.Request) {
	t := g.tabs[request.ID+".json"]
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
		if t := g.tabs[request.ID+".json"]; t != nil {
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
