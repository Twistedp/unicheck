package pages

import (
	"fmt"
	"image/color"
	"sort"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	datepicker "github.com/sdassow/fyne-datepicker"

	"unicheck/internal/model"
)

const deadlineLayout = "02.01.2006"

func BuildTasksPage(win fyne.Window, tasks *[]model.Task, courses *[]model.Course, saveNow func(), onChange func()) (fyne.CanvasObject, func()) {
	selected := -1
	visibleTaskIndexes := make([]int, 0, len(*tasks))

	taskName := widget.NewEntry()
	taskName.SetPlaceHolder("Task")

	taskType := widget.NewSelect([]string{"Exam", "Assignment", "AG"}, func(string) {})
	taskType.PlaceHolder = "Type"

	selectedDeadline := time.Now()
	var deadline *widget.Button
	deadline = widget.NewButton(selectedDeadline.Format(deadlineLayout), func() {
		picker := datepicker.NewDatePicker(selectedDeadline, time.Monday, func(chosen time.Time, ok bool) {
			if ok {
				selectedDeadline = chosen
				deadline.SetText(chosen.Format(deadlineLayout))
			}
		})

		dialog.ShowCustomConfirm("Choose deadline", "OK", "Cancel", picker, picker.OnActioned, win)
	})

	effort := widget.NewSelect([]string{"Easy", "Medium", "Hard"}, func(string) {})
	effort.PlaceHolder = "Effort"

	showDone := widget.NewCheck("Show Done", nil)
	showDone.SetChecked(false)

	courseRef := widget.NewSelect([]string{}, func(string) {})
	courseRef.PlaceHolder = "Course"

	courseFilter := widget.NewSelect([]string{"All courses"}, nil)
	courseFilter.SetSelected("All courses")

	refreshCourseOptions := func() {
		options := make([]string, 0, len(*courses))
		for _, course := range *courses {
			options = append(options, course.Name)
		}

		courseRef.Options = options
		courseRef.Refresh()

		if courseRef.Selected != "" && !containsOption(options, courseRef.Selected) {
			courseRef.ClearSelected()
		}

		filterOptions := append([]string{"All courses"}, options...)
		courseFilter.Options = filterOptions
		courseFilter.Refresh()

		if courseFilter.Selected == "" || !containsOption(filterOptions, courseFilter.Selected) {
			courseFilter.SetSelected("All courses")
		}
	}
	refreshCourseOptions()

	completed := widget.NewCheck("Completed", func(bool) {})

	clearForm := func() {
		taskName.SetText("")
		taskType.ClearSelected()
		effort.ClearSelected()
		courseRef.ClearSelected()
		selectedDeadline = time.Now()
		deadline.SetText(selectedDeadline.Format(deadlineLayout))
		completed.SetChecked(false)
	}

	loadTaskIntoForm := func(task model.Task) {
		taskName.SetText(task.Name)
		taskType.SetSelected(task.TaskType)
		effort.SetSelected(task.Effort)
		courseRef.SetSelected(task.CourseRef)
		if task.Deadline.IsZero() {
			selectedDeadline = time.Now()
		} else {
			selectedDeadline = task.Deadline
		}
		deadline.SetText(selectedDeadline.Format(deadlineLayout))
		completed.SetChecked(task.Completed)
	}

	sortVisibleTasksByDeadline := func() {
		sort.SliceStable(visibleTaskIndexes, func(i, j int) bool {
			a := (*tasks)[visibleTaskIndexes[i]]
			b := (*tasks)[visibleTaskIndexes[j]]

			if a.Deadline.IsZero() && b.Deadline.IsZero() {
				return a.Name < b.Name
			}
			if a.Deadline.IsZero() {
				return false
			}
			if b.Deadline.IsZero() {
				return true
			}
			if a.Deadline.Equal(b.Deadline) {
				return a.Name < b.Name
			}
			return a.Deadline.Before(b.Deadline)
		})
	}

	rebuildVisibleTasks := func() {
		visibleTaskIndexes = visibleTaskIndexes[:0]
		for i, task := range *tasks {
			if !showDone.Checked && task.Completed {
				continue
			}
			if courseFilter.Selected != "" && courseFilter.Selected != "All courses" && task.CourseRef != courseFilter.Selected {
				continue
			}
			visibleTaskIndexes = append(visibleTaskIndexes, i)
		}
		sortVisibleTasksByDeadline()
	}

	taskTypeColor := func(kind string) color.Color {
		switch kind {
		case "Exam":
			return color.NRGBA{R: 198, G: 40, B: 40, A: 255}
		case "Assignment":
			return color.NRGBA{R: 37, G: 99, B: 235, A: 255}
		case "AG":
			return color.NRGBA{R: 34, G: 139, B: 34, A: 255}
		default:
			return color.NRGBA{R: 107, G: 114, B: 128, A: 255}
		}
	}

	deadlineColor := func(deadline time.Time) color.Color {
		if deadline.IsZero() {
			return color.NRGBA{R: 107, G: 114, B: 128, A: 255}
		}

		now := time.Now()
		if deadline.Before(now) {
			return color.NRGBA{R: 198, G: 40, B: 40, A: 255}
		}
		if deadline.Before(now.Add(72 * time.Hour)) {
			return color.NRGBA{R: 217, G: 119, B: 6, A: 255}
		}
		return color.NRGBA{R: 8, G: 145, B: 178, A: 255}
	}

	var list *widget.List
	var updateModeUI func()
	rebuildVisibleTasks()

	list = widget.NewList(
		func() int { return len(visibleTaskIndexes) },
		func() fyne.CanvasObject {
			title := canvas.NewText("Task name", theme.Color(theme.ColorNameForeground))
			title.TextStyle = fyne.TextStyle{Bold: true}
			title.TextSize = theme.TextSize() + 1

			typeBg := canvas.NewRectangle(taskTypeColor("Exam"))
			typeLabel := canvas.NewText("Exam", color.White)
			typeLabel.Alignment = fyne.TextAlignCenter
			typeLabel.TextStyle = fyne.TextStyle{Bold: true}
			typeTag := container.NewPadded(container.NewStack(typeBg, container.NewCenter(typeLabel)))

			deadlineText := canvas.NewText("01.01.2006", deadlineColor(time.Now()))
			deadlineText.TextStyle = fyne.TextStyle{Bold: true}

			meta := container.NewHBox(typeTag, deadlineText)
			sub := widget.NewLabel("Effort • Course")
			done := widget.NewCheck("Done", func(bool) {})

			left := container.NewVBox(title, meta, sub)
			return container.NewBorder(nil, nil, nil, done, left)
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			taskIdx := visibleTaskIndexes[int(i)]
			task := (*tasks)[taskIdx]

			border := obj.(*fyne.Container)
			left := border.Objects[0].(*fyne.Container)
			title := left.Objects[0].(*canvas.Text)
			meta := left.Objects[1].(*fyne.Container)
			typeTag := meta.Objects[0].(*fyne.Container)
			typeBg := typeTag.Objects[0].(*fyne.Container).Objects[0].(*canvas.Rectangle)
			typeLabel := typeTag.Objects[0].(*fyne.Container).Objects[1].(*fyne.Container).Objects[0].(*canvas.Text)
			deadlineText := meta.Objects[1].(*canvas.Text)
			sub := left.Objects[2].(*widget.Label)
			done := border.Objects[1].(*widget.Check)

			title.Text = task.Name
			title.Refresh()

			typeLabel.Text = task.TaskType
			typeBg.FillColor = taskTypeColor(task.TaskType)
			typeBg.Refresh()
			typeLabel.Refresh()

			if task.Deadline.IsZero() {
				deadlineText.Text = "No deadline"
			} else {
				deadlineText.Text = task.Deadline.Format(deadlineLayout)
			}
			deadlineText.Color = deadlineColor(task.Deadline)
			deadlineText.Refresh()

			if task.CourseRef != "" {
				sub.SetText(fmt.Sprintf("%s • %s", task.CourseRef, task.Effort))
			} else {
				sub.SetText(task.Effort)
			}

			done.OnChanged = nil
			done.SetChecked(task.Completed)

			idx := taskIdx
			done.OnChanged = func(value bool) {
				(*tasks)[idx].Completed = value
				rebuildVisibleTasks()
				if selected == idx && !showDone.Checked && value {
					selected = -1
					list.UnselectAll()
					clearForm()
					if updateModeUI != nil {
						updateModeUI()
					}
				}
				list.Refresh()
				saveNow()
				onChange()
			}
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		selected = visibleTaskIndexes[int(id)]
		loadTaskIntoForm((*tasks)[selected])
	}

	list.OnUnselected = func(widget.ListItemID) {
		selected = -1
		clearForm()
	}

	saveBtn := widget.NewButton("Add Task", nil)
	cancelBtn := widget.NewButton("Cancel Edit", func() {
		selected = -1
		list.UnselectAll()
		clearForm()
	})
	cancelBtn.Hide()

	deleteBtn := widget.NewButton("Delete Task", func() {
		if selected < 0 || selected >= len(*tasks) {
			return
		}

		*tasks = append((*tasks)[:selected], (*tasks)[selected+1:]...)
		rebuildVisibleTasks()
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
		eff := effort.Selected
		if eff == "" {
			eff = "Medium"
		}

		kind := taskType.Selected
		if kind == "" {
			kind = "Assignment"
		}

		task := model.Task{
			Name:      taskName.Text,
			TaskType:  kind,
			Effort:    eff,
			CourseRef: courseRef.Selected,
			Completed: completed.Checked,
			Deadline:  selectedDeadline,
		}

		if selected >= 0 {
			(*tasks)[selected] = task
		} else {
			*tasks = append(*tasks, task)
		}

		rebuildVisibleTasks()
		selected = -1
		list.UnselectAll()
		clearForm()
		cancelBtn.Hide()
		saveBtn.SetText("Add Task")
		updateModeUI()
		list.Refresh()
		saveNow()
		onChange()
	}

	updateModeUI = func() {
		if selected >= 0 {
			saveBtn.SetText("Save Changes")
			cancelBtn.Show()
			deleteBtn.Show()
			return
		}

		saveBtn.SetText("Add Task")
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

	showDone.OnChanged = func(bool) {
		rebuildVisibleTasks()
		selected = -1
		list.UnselectAll()
		clearForm()
		updateModeUI()
		list.Refresh()
	}

	courseFilter.OnChanged = func(string) {
		rebuildVisibleTasks()
		selected = -1
		list.UnselectAll()
		clearForm()
		updateModeUI()
		list.Refresh()
	}

	form := container.NewVBox(
		widget.NewLabelWithStyle("Add / Edit Task", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Task", taskName),
			widget.NewFormItem("Type", taskType),
			widget.NewFormItem("Deadline", deadline),
			widget.NewFormItem("Effort", effort),
			widget.NewFormItem("Course", courseRef),
			widget.NewFormItem("", completed),
		),
		container.NewHBox(saveBtn, cancelBtn, deleteBtn),
	)

	right := container.NewBorder(
		container.NewHBox(
			widget.NewLabelWithStyle("Tasks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			widget.NewLabel("Deadline (closest first)"),
			courseFilter,
			showDone,
		),
		nil, nil, nil,
		list,
	)

	split := container.NewHSplit(form, right)
	split.Offset = 0.36

	return split, func() {
		refreshCourseOptions()
		rebuildVisibleTasks()
		selected = -1
		list.UnselectAll()
		clearForm()
		updateModeUI()
		list.Refresh()
	}
}

func containsOption(options []string, selected string) bool {
	for _, option := range options {
		if option == selected {
			return true
		}
	}
	return false
}
