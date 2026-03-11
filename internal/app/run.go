package app

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"unicheck/internal/model"
	"unicheck/internal/storage"
	"unicheck/internal/ui/pages"
)

func Run() error {
	dataPath, err := storage.DataFilePath()
	if err != nil {
		return err
	}

	data, err := storage.LoadData(dataPath)
	if err != nil {
		return err
	}

	courses := data.Courses
	tasks := data.Tasks

	var saveTimer *time.Timer
	saveNow := func() {
		if saveTimer != nil {
			saveTimer.Stop()
		}

		saveTimer = time.AfterFunc(300*time.Millisecond, func() {
			_ = storage.SaveDataAtomic(dataPath, model.AppData{
				Courses: courses,
				Tasks:   tasks,
			})
		})
	}

	fyneApp := app.NewWithID("dev.twistedp.unicheck")
	window := fyneApp.NewWindow("Uni Organizer")
	window.Resize(fyne.NewSize(980, 620))

	refreshOverview := func() {}

	coursesPage := pages.BuildCoursesPage(window, &courses, saveNow, func() {
		refreshOverview()
	})

	tasksPage, refreshTasksPage := pages.BuildTasksPage(window, &tasks, &courses, saveNow, func() {
		refreshOverview()
	})

	overviewPage, refreshOverviewPage := pages.BuildOverviewPage(&courses, &tasks)
	refreshOverview = refreshOverviewPage

	tabs := container.NewAppTabs(
		container.NewTabItem("Overview", overviewPage),
		container.NewTabItem("Courses", coursesPage),
		container.NewTabItem("Tasks", tasksPage),
	)
	tabs.SetTabLocation(container.TabLocationTop)
	tabs.OnSelected = func(tab *container.TabItem) {
		switch tab.Text {
		case "Overview":
			refreshOverview()
		case "Tasks":
			refreshTasksPage()
		}
	}

	header := container.NewHBox(
		widget.NewLabelWithStyle("University Organizer", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		layout.NewSpacer(),
	)

	window.SetContent(container.NewBorder(header, nil, nil, nil, tabs))
	window.CenterOnScreen()
	window.Show()
	fyneApp.Run()

	return nil
}
