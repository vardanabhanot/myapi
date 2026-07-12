package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/vardanabhanot/myapi/core"
)

// makeEnvContent builds the Environments sidebar tab. Row 0 is the fixed
// "No Environment" option; list selection IS the active environment.
func (g *gui) makeEnvContent() *fyne.Container {
	g.envStore = core.LoadEnvStore()
	core.SetActiveVars(g.envStore.ActiveEnv().VarMap())

	g.envList = widget.NewList(
		func() int {
			return len(g.envStore.Envs) + 1
		},
		func() fyne.CanvasObject {
			name := widget.NewLabel("Environment")
			name.Truncation = fyne.TextTruncateEllipsis
			edit := newTappableIcon(theme.DocumentCreateIcon(), func() {})

			return container.NewBorder(nil, nil, nil, container.NewPadded(edit), name)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			row := o.(*fyne.Container)
			name := row.Objects[0].(*widget.Label)
			edit := row.Objects[1].(*fyne.Container).Objects[0].(*tappableIcon)

			if i == 0 {
				name.SetText("No Environment")
				edit.Hide()
				return
			}

			env := g.envStore.Envs[i-1]
			name.SetText(env.Name)
			edit.Show()
			edit.onTapped = func() {
				g.editEnvDialog(env)
			}
		},
	)

	g.envList.OnSelected = func(i widget.ListItemID) {
		if i == 0 {
			g.envStore.Active = ""
		} else {
			g.envStore.Active = g.envStore.Envs[i-1].Name
		}

		core.SetActiveVars(g.envStore.ActiveEnv().VarMap())

		if err := core.SaveEnvStore(g.envStore); err != nil {
			dialog.NewError(err, *g.Window).Show()
		}
	}

	g.selectActiveEnv()

	addBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), func() {
		env := &core.Environment{Name: "New Environment", Variables: &[]core.FormType{{Checked: true}}}
		g.envStore.Envs = append(g.envStore.Envs, env)

		// A first environment is what the user came here to use — activate
		// it instead of leaving "No Environment" selected.
		if g.envStore.Active == "" {
			g.envStore.Active = env.Name
		}

		g.envList.Refresh()
		g.editEnvDialog(env)
	})
	addBtn.Importance = widget.HighImportance

	header := container.NewBorder(nil, nil, container.NewPadded(sectionHeader("Environments")), container.NewPadded(addBtn), nil)

	return container.NewBorder(header, nil, nil, nil, g.envList)
}

// selectActiveEnv syncs the list selection with the persisted active env.
func (g *gui) selectActiveEnv() {
	index := 0

	if g.envStore.Active != "" {
		for i, env := range g.envStore.Envs {
			if env.Name == g.envStore.Active {
				index = i + 1
				break
			}
		}
	}

	g.envList.Select(index)
}

// editEnvDialog edits an environment in place: the name entry and formBlock
// write straight into the struct, so closing the dialog just persists.
func (g *gui) editEnvDialog(env *core.Environment) {
	nameEntry := widget.NewEntry()
	nameEntry.SetText(env.Name)
	nameEntry.OnChanged = func(s string) {
		// Active is tracked by name, keep it following a rename
		if g.envStore.Active == env.Name {
			g.envStore.Active = s
		}
		env.Name = s
	}

	var d *dialog.CustomDialog

	deleteBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		dialog.NewConfirm("Delete Environment", "Delete \""+env.Name+"\"? This cannot be undone.", func(confirmed bool) {
			if !confirmed {
				return
			}

			for i, e := range g.envStore.Envs {
				if e == env {
					g.envStore.Envs = append(g.envStore.Envs[:i], g.envStore.Envs[i+1:]...)
					break
				}
			}

			if g.envStore.Active == env.Name {
				g.envStore.Active = ""
			}

			d.Hide() // OnClosed persists and re-syncs the list
		}, *g.Window).Show()
	})
	deleteBtn.Importance = widget.DangerImportance

	hint := widget.NewLabel("Use {{name}} in URL, headers, body or auth fields.")
	hint.Importance = widget.LowImportance

	content := container.NewBorder(
		container.NewVBox(nameEntry, hint),
		container.NewBorder(nil, nil, deleteBtn, nil),
		nil, nil,
		g.formBlock(env.Variables),
	)

	d = dialog.NewCustom("Edit Environment", "Done", content, *g.Window)
	d.SetOnClosed(func() {
		core.SetActiveVars(g.envStore.ActiveEnv().VarMap())

		if err := core.SaveEnvStore(g.envStore); err != nil {
			dialog.NewError(err, *g.Window).Show()
		}

		g.envList.Refresh()
		g.selectActiveEnv()
	})
	d.Resize(fyne.NewSize(560, 460))
	d.Show()
}
