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
		defer g.collectionTree.UnselectAll()

		var ci, ri int
		if _, err := fmt.Sscanf(uid, "r:%d:%d", &ci, &ri); err != nil {
			return // branch taps just expand
		}
		if ci >= len(g.collections) || ri >= len(g.collections[ci].Requests) {
			return
		}

		// Open a copy: collection entries are snapshots, so edits and sends
		// belong to the new tab (and history), not the collection.
		request := g.collections[ci].Requests[ri].Clone()
		request.ID = core.NewRequestID()
		request.IsDirty = true

		tabItem := g.makeTab(request)
		g.doctabs.Append(tabItem)
		g.doctabs.Select(tabItem)
	}

	newBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), func() {
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
	newBtn.Importance = widget.HighImportance

	header := container.NewBorder(nil, nil, container.NewPadded(sectionHeader("Collections")), container.NewPadded(newBtn), nil)

	return container.NewBorder(header, nil, nil, nil, g.collectionTree)
}

// addToCollectionDialog snapshots the request into a picked (or newly
// created) collection.
func (g *gui) addToCollectionDialog(request *core.Request) {
	var d dialog.Dialog

	addTo := func(col *core.Collection) {
		col.Requests = append(col.Requests, request.Clone())
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
