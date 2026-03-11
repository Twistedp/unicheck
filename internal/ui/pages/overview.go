package pages

import (
	"fmt"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"unicheck/internal/model"
)

func BuildOverviewPage(courses *[]model.Course, tasks *[]model.Task) (fyne.CanvasObject, func()) {
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
		body := container.NewVBox(head, value, detail, bar)
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
		for _, course := range *courses {
			totalECTS += course.ECTS
			if course.Completed {
				completedECTS += course.ECTS
			}
		}

		totalTasks := len(*tasks)
		doneTasks := 0
		for _, task := range *tasks {
			if task.Completed {
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
