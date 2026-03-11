package pages

import (
	"fmt"
	"sort"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"unicheck/internal/model"
)

func BuildCoursesPage(_ fyne.Window, courses *[]model.Course, saveNow func(), onChange func()) fyne.CanvasObject {
	selected := -1

	name := widget.NewEntry()
	name.SetPlaceHolder("Course name")

	module := widget.NewEntry()
	module.SetPlaceHolder("Module")

	ects := widget.NewEntry()
	ects.SetPlaceHolder("ECTS")

	difficulty := widget.NewSelect([]string{"Easy", "Medium", "Hard"}, func(string) {})
	difficulty.PlaceHolder = "Difficulty"

	completed := widget.NewCheck("Completed", func(bool) {})

	clearForm := func() {
		name.SetText("")
		module.SetText("")
		ects.SetText("")
		difficulty.ClearSelected()
		completed.SetChecked(false)
	}

	loadCourseIntoForm := func(course model.Course) {
		name.SetText(course.Name)
		module.SetText(course.Module)
		ects.SetText(strconv.FormatFloat(course.ECTS, 'f', -1, 64))
		difficulty.SetSelected(course.Difficulty)
		completed.SetChecked(course.Completed)
	}

	var list *widget.List
	list = widget.NewList(
		func() int { return len(*courses) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("Course Name")
			sub := widget.NewLabel("Module • ECTS • Difficulty")
			done := widget.NewCheck("Done", func(bool) {})

			left := container.NewVBox(title, sub)
			return container.NewBorder(nil, nil, nil, done, left)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			course := (*courses)[i]

			border := obj.(*fyne.Container)
			left := border.Objects[0].(*fyne.Container)
			title := left.Objects[0].(*widget.Label)
			sub := left.Objects[1].(*widget.Label)
			done := border.Objects[1].(*widget.Check)

			title.SetText(course.Name)
			sub.SetText(fmt.Sprintf("%s • %.1f ECTS • %s", course.Module, course.ECTS, course.Difficulty))

			done.OnChanged = nil
			done.SetChecked(course.Completed)

			idx := int(i)
			done.OnChanged = func(value bool) {
				(*courses)[idx].Completed = value
				list.Refresh()
				saveNow()
				onChange()
			}
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		selected = int(id)
		loadCourseIntoForm((*courses)[selected])
	}

	list.OnUnselected = func(widget.ListItemID) {
		selected = -1
		clearForm()
	}

	saveBtn := widget.NewButton("Add Course", nil)
	cancelBtn := widget.NewButton("Cancel Edit", func() {
		selected = -1
		list.UnselectAll()
		clearForm()
	})
	cancelBtn.Hide()

	deleteBtn := widget.NewButton("Delete Course", func() {
		if selected < 0 || selected >= len(*courses) {
			return
		}

		*courses = append((*courses)[:selected], (*courses)[selected+1:]...)
		selected = -1
		list.UnselectAll()
		clearForm()
		list.Refresh()
		saveNow()
		onChange()
	})
	deleteBtn.Importance = widget.DangerImportance
	deleteBtn.Hide()

	saveBtn.OnTapped = func() {
		ectsValue := 0.0
		if ects.Text != "" {
			if value, err := strconv.ParseFloat(ects.Text, 64); err == nil {
				ectsValue = value
			}
		}

		diff := difficulty.Selected
		if diff == "" {
			diff = "Medium"
		}

		course := model.Course{
			Name:       name.Text,
			Module:     module.Text,
			ECTS:       ectsValue,
			Difficulty: diff,
			Completed:  completed.Checked,
		}

		if selected >= 0 {
			(*courses)[selected] = course
			list.Refresh()
			list.Select(selected)
		} else {
			*courses = append(*courses, course)
			list.Refresh()
		}

		selected = -1
		list.UnselectAll()
		clearForm()
		cancelBtn.Hide()
		saveBtn.SetText("Add Course")
		saveNow()
		onChange()
	}

	updateModeUI := func() {
		if selected >= 0 {
			saveBtn.SetText("Save Changes")
			cancelBtn.Show()
			deleteBtn.Show()
			return
		}

		saveBtn.SetText("Add Course")
		cancelBtn.Hide()
		deleteBtn.Hide()
	}

	prevOnSelected := list.OnSelected
	list.OnSelected = func(id widget.ListItemID) {
		prevOnSelected(id)
		updateModeUI()
	}

	prevOnUnselected := list.OnUnselected
	list.OnUnselected = func(id widget.ListItemID) {
		prevOnUnselected(id)
		updateModeUI()
	}

	sortSelect := widget.NewSelect([]string{
		"Name (A→Z)",
		"Module (A→Z)",
		"ECTS (high→low)",
		"Difficulty (easy→hard)",
		"Completed (done last)",
	}, func(choice string) {
		switch choice {
		case "Name (A→Z)":
			sort.SliceStable(*courses, func(i, j int) bool {
				return (*courses)[i].Name < (*courses)[j].Name
			})
		case "Module (A→Z)":
			sort.SliceStable(*courses, func(i, j int) bool {
				a, b := (*courses)[i], (*courses)[j]
				if a.Module != b.Module {
					return a.Module < b.Module
				}
				return a.Name < b.Name
			})
		case "ECTS (high→low)":
			sort.SliceStable(*courses, func(i, j int) bool {
				return (*courses)[i].ECTS > (*courses)[j].ECTS
			})
		case "Difficulty (easy→hard)":
			rank := map[string]int{"Easy": 0, "Medium": 1, "Hard": 2}
			sort.SliceStable(*courses, func(i, j int) bool {
				return rank[(*courses)[i].Difficulty] < rank[(*courses)[j].Difficulty]
			})
		case "Completed (done last)":
			sort.SliceStable(*courses, func(i, j int) bool {
				return !(*courses)[i].Completed && (*courses)[j].Completed
			})
		}

		list.Refresh()
		saveNow()
	})
	sortSelect.SetSelected("Name (A→Z)")

	form := container.NewVBox(
		widget.NewLabelWithStyle("Add / Edit Course", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Name", name),
			widget.NewFormItem("Module", module),
			widget.NewFormItem("ECTS", ects),
			widget.NewFormItem("Difficulty", difficulty),
			widget.NewFormItem("", completed),
		),
		container.NewHBox(saveBtn, cancelBtn, deleteBtn),
	)

	right := container.NewBorder(
		container.NewHBox(widget.NewLabelWithStyle("Courses S26", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), sortSelect),
		nil, nil, nil,
		list,
	)

	split := container.NewHSplit(form, right)
	split.Offset = 0.36

	return split
}
