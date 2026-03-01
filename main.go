package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/sdassow/fyne-datepicker"
)

type Course struct {
	Name       string
	Module     string
	ECTS       float64
	Difficulty string
	Completed  bool
}

type Task struct {
	Name      string
	TaskType  string
	Effort    string
	CourseRef string
	Completed bool
	Deadline  time.Time
}

type AppData struct {
	Courses []Course `json:"courses"`
	Tasks   []Task   `json:"tasks"`
}

func main() {
	a := app.New()
	w := a.NewWindow("Uni Organizer")
	w.Resize(fyne.NewSize(980, 620))

	// ---- In-memory data (for UI demo only) ----
	// courses := []Course{
	// 	{Name: "Analytical Chemistry", Module: "Chemistry", ECTS: 6, Difficulty: "Hard", Completed: false},
	// 	{Name: "Molecular Biology II", Module: "Biology", ECTS: 5, Difficulty: "Medium", Completed: false},
	// }
	// tasks := []Task{
	// 	{Name: "Prepare lab report #1", TaskType: "Exam", Effort: "Low", CourseRef: "Analytical Chemistry", Completed: false, Deadline: time.Now()},
	// 	{Name: "Learn lecture 2 slides", TaskType: "Assignment", Effort: "Medium", CourseRef: "Molecular Biology II", Completed: true, Deadline: time.Now()},
	// }

	// ---- Setup data structs and load json data ----
	
	path, err := dataFilePath(); if err != nil {panic(err)}
	
	data, err := loadData(path); if err != nil {panic(err)}
	
	// courses := make([]Course, len(data.Courses))
	// tasks := make([]Task, len(data.Tasks))

	courses := data.Courses
	tasks := data.Tasks

	var saveTimer *time.Timer
	saveNow := func() {
		if saveTimer != nil {
			saveTimer.Stop()
		}

		// debounce to not overload FS (called in goroutine)
		saveTimer = time.AfterFunc(300*time.Millisecond, func() {
			_ = saveDataAtomic(path, AppData{Courses: courses, Tasks: tasks})
		})
	}


	// ---- Build Pages ----

	refreshOverview := func() {}

	coursesPage := buildCoursesPage(w, &courses, saveNow, func() {
		refreshOverview()
	})

	//func to repopulate Courses dropdown
	var refreshCourses func()
	tasksPage, refreshCourses := buildTasksPage(w, &tasks, &courses, saveNow, func() {
		refreshOverview()
	})
	overviewPage, refreshOverviewFn := buildOverviewPage(&courses, &tasks)
	refreshOverview = refreshOverviewFn


	tabs := container.NewAppTabs(
		container.NewTabItem("Overview", overviewPage),
		container.NewTabItem("Courses", coursesPage),
		container.NewTabItem("Tasks", tasksPage),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	
	tabs.OnSelected = func(ti * container.TabItem) {
		if ti.Text == "Overview" {
			refreshOverview()
		}
		if ti.Text == "Tasks" {
			refreshCourses()
		}
	}

	header := container.NewHBox(
		widget.NewLabelWithStyle("University Organizer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
	)

	w.SetContent(container.NewBorder(header, nil, nil, nil, tabs))
	w.ShowAndRun()
}

func buildCoursesPage(win fyne.Window, courses *[]Course, saveNow func(), onChange func()) fyne.CanvasObject {

	selected := -1

	// ---- Form fields ----
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

	loadCourseIntoForm := func(c Course) {
		name.SetText(c.Name)
		module.SetText(c.Module)
		ects.SetText(strconv.FormatFloat(c.ECTS, 'f', -1, 64))
		difficulty.SetSelected(c.Difficulty)
		completed.SetChecked(c.Completed)
	}

	var list *widget.List
	list = widget.NewList(
		func() int { return len(*courses) },
		func() fyne.CanvasObject {
			// row template
			title := widget.NewLabel("Course Name")
			sub := widget.NewLabel("Module • ECTS • Difficulty")
			done := widget.NewCheck("Done", func(bool) {})

			left := container.NewVBox(title, sub)
			row := container.NewBorder(nil, nil, nil, done, left)
			return row
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			c := (*courses)[i]

			// obj is the row container we created above:
			border := obj.(*fyne.Container)             // container.NewBorder => *fyne.Container
			left := border.Objects[0].(*fyne.Container) // left VBox is first in Objects for this border layout
			title := left.Objects[0].(*widget.Label)
			sub := left.Objects[1].(*widget.Label)
			done := border.Objects[1].(*widget.Check)

			title.SetText(c.Name)
			sub.SetText(fmt.Sprintf("%s • %.1f ECTS • %s", c.Module, c.ECTS, c.Difficulty))

			done.OnChanged = nil
			done.SetChecked(c.Completed)
			idx := int(i)
			done.OnChanged = func(v bool) {
				(*courses)[idx].Completed = v
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

	list.OnUnselected = func(id widget.ListItemID) {
		selected = -1
		clearForm()
	}

	// ---- Buttons ----

	saveBtn := widget.NewButton("Add Course", nil)
	cancelBtn := widget.NewButton("Cancel Edit", func() {
		selected = -1
		list.UnselectAll()
		clearForm()
	})
	cancelBtn.Hide()

	saveBtn.OnTapped = func() {
		ectsInt := float64(0)
		if ects.Text != "" {
			if v, err := strconv.ParseFloat(ects.Text, 64); err == nil {
				ectsInt = v
			}
		}

		diff := difficulty.Selected
		if diff == "" {
			diff = "Medium"
		}

		if selected >= 0 {
			// ---- EDIT: update existing item ----
			(*courses)[selected] = Course{
				Name:       name.Text,
				Module:     module.Text,
				ECTS:       ectsInt,
				Difficulty: diff,
				Completed:  completed.Checked,
			}
			list.Refresh()
			list.Select(selected)
		} else {
			// ---- Add append new Course ----
			*courses = append(*courses, Course{
				Name:       name.Text,
				Module:     module.Text,
				ECTS:       ectsInt,
				Difficulty: diff,
				Completed:  completed.Checked,
			})
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

	//Toggle Ui mode based on selection
	updateModeUI := func() {
		if selected >= 0 {
			saveBtn.SetText("Save Changes")
			cancelBtn.Show()
		} else {
			saveBtn.SetText("Add Course")
			cancelBtn.Hide()
		}
	}

	// override onselected and onUnselectedto update the ui mode

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

	// ---- Sorting ----
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
				// false before true => completed last
				return (*courses)[i].Completed && !(*courses)[j].Completed == false
			})
		}
		list.Refresh()
		saveNow()
	})
	sortSelect.SetSelected("Name (A→Z)")

	// ---- Layout: left form, right list ----
	form := container.NewVBox(
		widget.NewLabelWithStyle("Add / Edit Course", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Name", name),
			widget.NewFormItem("Module", module),
			widget.NewFormItem("ECTS", ects),
			widget.NewFormItem("Difficulty", difficulty),
			widget.NewFormItem("", completed),
		),
		container.NewHBox(saveBtn, cancelBtn),
	)

	right := container.NewBorder(
		container.NewHBox(widget.NewLabelWithStyle("Courses S26", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), sortSelect),
		nil, nil, nil,
		list,
	)

	// Split view (feels like a desktop app)
	split := container.NewHSplit(form, right)
	split.Offset = 0.36

	return split

}

func buildTasksPage(win fyne.Window, tasks *[]Task, courses *[]Course, saveNow func(), onChange func()) (fyne.CanvasObject, func()) {
	const deadlineLayout = "02.01.2006"

	selected := -1

	// ---- Form fields ----
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

		dialog.ShowCustomConfirm(
			"Choose deadline",
			"OK",
			"Cancel",
			picker,
			picker.OnActioned,
			win,
		)
	})

	effort := widget.NewSelect([]string{"Easy", "Medium", "Hard"}, func(string) {})
	effort.PlaceHolder = "Effort"


	courseRef := widget.NewSelect([]string{}, func(string) {})
	courseRef.PlaceHolder = "Course"

	refreshCourseOptions := func() {
		opts := make([]string, 0, len(*courses))
		for _, c := range *courses {
			opts = append(opts, fmt.Sprintf("%s", c.Name))
		}
		courseRef.Options = opts
		courseRef.Refresh()

		if courseRef.Selected != "" {
			found := false
			for _, o := range opts {
				if o == courseRef.Selected {
					found = true
					break 
				}

			}
			if !found {
				courseRef.ClearSelected()
			}
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

	loadTaskIntoForm := func(t Task) {
		taskName.SetText(t.Name)
		taskType.SetSelected(t.TaskType)
		effort.SetSelected(t.Effort)
		courseRef.SetSelected(t.CourseRef)
		if t.Deadline.IsZero() {
			selectedDeadline = time.Now()
		} else {
			selectedDeadline = t.Deadline
		}
		deadline.SetText(selectedDeadline.Format(deadlineLayout))
		completed.SetChecked(t.Completed)
	}

	// ---- List ----
	var list *widget.List
	list = widget.NewList(
		func() int { return len(*tasks) },
		func() fyne.CanvasObject {
			title := widget.NewLabel("Task name")
			sub := widget.NewLabel("Effort • Course")
			done := widget.NewCheck("Done", func(bool) {})

			left := container.NewVBox(title, sub)
			row := container.NewBorder(nil, nil, nil, done, left)
			return row
		},
		func(i widget.ListItemID, obj fyne.CanvasObject) {
			t := (*tasks)[i]

			border := obj.(*fyne.Container)
			left := border.Objects[0].(*fyne.Container)
			title := left.Objects[0].(*widget.Label)
			sub := left.Objects[1].(*widget.Label)
			done := border.Objects[1].(*widget.Check)

			title.SetText(t.Name)
			parts := []string{t.TaskType, t.Effort}
			if t.CourseRef != "" {
				parts = append(parts, t.CourseRef)
			}
			if t.Deadline.IsZero() {
				parts = append(parts, "No deadline")
			} else {
				parts = append(parts, t.Deadline.Format(deadlineLayout))
			}
			sub.SetText(strings.Join(parts, "  •  "))

			done.OnChanged = nil
			done.SetChecked(t.Completed)
			idx := int(i)
			done.OnChanged = func(v bool) {
				(*tasks)[idx].Completed = v
				list.Refresh()
				saveNow()
				onChange()
			}
		},
	)

	list.OnSelected = func(id widget.ListItemID) {
		selected = int(id)
		loadTaskIntoForm((*tasks)[selected])
	}

	list.OnUnselected = func(id widget.ListItemID) {
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

	saveBtn.OnTapped = func() {

		eff := effort.Selected
		if eff == "" {
			eff = "Medium"
		}
		tt := taskType.Selected
		if tt == "" {
			tt = "Assignment"
		}

		if selected >= 0 {
			// ---- EDIT: update existing item ----
			(*tasks)[selected] = Task{
				Name:      taskName.Text,
				TaskType:  tt,
				Effort:    eff,
				CourseRef: courseRef.Selected,
				Completed: completed.Checked,
				Deadline:  selectedDeadline,
			}
			list.Refresh()
			list.Select(selected)
		} else {
			// ---- Add append new Task ----
			*tasks = append(*tasks, Task{
				Name:      taskName.Text,
				TaskType:  tt,
				Effort:    eff,
				CourseRef: courseRef.Selected,
				Completed: completed.Checked,
				Deadline:  selectedDeadline,
			})
			list.Refresh()
		}

		selected = -1
		list.UnselectAll()
		clearForm()
		cancelBtn.Hide()
		saveBtn.SetText("Add Task")
		saveNow()
		onChange()

	}

	//Toggle Ui mode based on selection
	updateModeUI := func() {
		if selected >= 0 {
			saveBtn.SetText("Save Changes")
			cancelBtn.Show()
		} else {
			saveBtn.SetText("Add Task")
			cancelBtn.Hide()
		}
	}

	// override onselected and onUnselectedto update the ui mode

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

	// ---- Layout ----
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
		container.NewHBox(saveBtn, cancelBtn),
	)

	right := container.NewBorder(
		widget.NewLabelWithStyle("Tasks", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		nil, nil, nil,
		list,
	)

	split := container.NewHSplit(form, right)
	split.Offset = 0.36

	return split, refreshCourseOptions
}

func buildOverviewPage(courses *[]Course, tasks *[]Task) (fyne.CanvasObject, func()) {
	title := widget.NewLabelWithStyle("Overview", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	subtitle := widget.NewLabel("Your semester at a glance")

	totalECTSValue := widget.NewLabelWithStyle("0.0", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	courseCountValue := widget.NewLabel("0 courses")
	ectsBar := widget.NewProgressBar()

	doneTasksValue := widget.NewLabelWithStyle("0", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	doneTasksSub := widget.NewLabel("Tasks completed")
	doneBar := widget.NewProgressBar()

	openTasksValue := widget.NewLabelWithStyle("0", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	openTasksSub := widget.NewLabel("Tasks open")
	openBar := widget.NewProgressBar()

	card := func(headline string, value *widget.Label, detail *widget.Label, bar *widget.ProgressBar) fyne.CanvasObject {
		head := widget.NewLabelWithStyle(headline, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		body := container.NewVBox(
			head,
			value,
			detail,
			bar,
		)
		return widget.NewCard("", "", body)
	}

	topRow := container.NewGridWithColumns(
		2,
		card("Total ECTS", totalECTSValue, courseCountValue, ectsBar),
		card("Completed Tasks", doneTasksValue, doneTasksSub, doneBar),
	)

	bottomRow := container.NewGridWithColumns(
		1,
		card("Open Tasks", openTasksValue, openTasksSub, openBar),
	)

	content := container.NewVBox(
		title,
		subtitle,
		widget.NewSeparator(),
		topRow,
		bottomRow,
	)

	refresh := func() {
		totalECTS := 0.0
		completedECTS := 0.0
		for _, c := range *courses {
			totalECTS += c.ECTS
			if c.Completed {
				completedECTS += c.ECTS
			}
		}
		totalTasks := len(*tasks)
		doneTasks := 0
		for _, t := range *tasks {
			if t.Completed {
				doneTasks++
			}
		}
		openTasks := totalTasks - doneTasks

		taskRatio := 0.0
		if totalTasks > 0 {
			taskRatio = float64(doneTasks) / float64(totalTasks)
		}

		ectsBar.Max = totalECTS
		ectsBar.SetValue(completedECTS)
		if totalECTS == 0 {
			ectsBar.Max = 1
			ectsBar.SetValue(0)
		}
		totalECTSValue.SetText(fmt.Sprintf("%.1f", totalECTS))
		courseCountValue.SetText(fmt.Sprintf("%.1f / %.1f ECTS completed", completedECTS, totalECTS))

		doneTasksValue.SetText(strconv.Itoa(doneTasks))
		doneTasksSub.SetText(fmt.Sprintf("%d of %d tasks done", doneTasks, totalTasks))
		doneBar.SetValue(taskRatio)

		openTasksValue.SetText(strconv.Itoa(openTasks))
		openTasksSub.SetText(fmt.Sprintf("%d of %d tasks open", openTasks, totalTasks))
		openBar.SetValue(1 - taskRatio)
	}

	refresh()
	return container.NewPadded(content), refresh
}


func dataFilePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil { return "", err}
	dir = filepath.Join(dir, "uni-organizer")
	if err := os.MkdirAll(dir, 0o755); err != nil {return "", err}
	return filepath.Join(dir, "data.json"), nil
}

func loadData(path string) (AppData, error) {
	var d AppData

	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return d, nil // first run
		}
		return d, err
	}

	if len(b) == 0 {
		return d, nil
	}
	err = json.Unmarshal(b, &d)
	return d, err
}

func saveDataAtomic(path string, d AppData) error {
	b, err := json.MarshalIndent(d, "", "  ")
	if err != nil {
		return err
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
