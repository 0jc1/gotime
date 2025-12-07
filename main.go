package main

import (
	"fmt"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type TaskTimer struct {
	taskName        string
	elapsedTime     time.Duration
	isRunning       bool
	ticker          *time.Ticker
	taskList        map[string]time.Duration
	taskListMutex   sync.Mutex
	timeLabel       *widget.Label
	pauseResumeBtn  *widget.Button
	taskSelector    *widget.Select
	statsUpdateFunc func()
	stopTicker      chan bool
	currentView     string
	contentBox      *fyne.Container
}

const (
	TickInterval = 100 * time.Millisecond
)

func main() {
	myApp := app.New()
	w := myApp.NewWindow("Task Timer")

	// Set window size to be tall and narrow
	w.Resize(fyne.NewSize(400, 900))

	// Create task timer instance
	timer := &TaskTimer{
		taskName:      "Select a task",
		elapsedTime:   0,
		isRunning:     false,
		taskList:      make(map[string]time.Duration),
		stopTicker:    make(chan bool, 1),
		currentView:   "timer",
	}

	// Create the three main containers
	timerContainer := createTimerContainer(timer)
	dailyStatsContainer := createDailyStatsContainer(timer)
	addTaskContainer := createAddTaskContainer(timer)

	// Create content box that will hold the current view
	timer.contentBox = container.NewVBox()
	updateContentView(timer, timerContainer, dailyStatsContainer, addTaskContainer)

	// Create sidebar with navigation buttons
	sidebarContainer := container.NewVBox(
		widget.NewCard("Navigation", "", nil),
		widget.NewButton("⏱ Timer", func() {
			timer.currentView = "timer"
			updateContentView(timer, timerContainer, dailyStatsContainer, addTaskContainer)
		}),
		widget.NewButton("📊 Daily Stats", func() {
			timer.currentView = "stats"
			updateContentView(timer, timerContainer, dailyStatsContainer, addTaskContainer)
		}),
		widget.NewButton("➕ Add Task", func() {
			timer.currentView = "addtask"
			updateContentView(timer, timerContainer, dailyStatsContainer, addTaskContainer)
		}),
	)

	// Create main layout with sidebar and content
	mainLayout := container.NewHBox(
		container.NewVBox(
			widget.NewLabel("Menu"),
			widget.NewSeparator(),
			sidebarContainer,
		),
		timer.contentBox,
	)

	w.SetContent(mainLayout)
	w.ShowAndRun()
}

func updateContentView(timer *TaskTimer, timerContainer, dailyStatsContainer, addTaskContainer fyne.CanvasObject) {
	fyne.Do(func() {
		timer.contentBox.RemoveAll()

		switch timer.currentView {
		case "timer":
			timer.contentBox.Add(timerContainer)
		case "stats":
			timer.contentBox.Add(container.NewVBox(
				widget.NewLabel("📊 Daily Stats"),
				dailyStatsContainer,
			))
		case "addtask":
			timer.contentBox.Add(container.NewVBox(
				widget.NewLabel("➕ Add New Task"),
				addTaskContainer,
			))
		}
	})
}

func createTimerContainer(timer *TaskTimer) *fyne.Container {
	// Task name display
	taskNameLabel := widget.NewLabel(timer.taskName)
	taskNameLabel.Alignment = fyne.TextAlignCenter

	// Elapsed time display (HH:MM:SS format)
	timer.timeLabel = widget.NewLabel("00:00:00")
	timer.timeLabel.Alignment = fyne.TextAlignCenter

	// Task selector dropdown
	timer.taskSelector = widget.NewSelect([]string{"Select a task"}, func(value string) {
		timer.taskName = value
		taskNameLabel.SetText(value)
	})
	timer.taskSelector.PlaceHolder = "Select a task"
	timer.taskSelector.SetSelected("Select a task")

	// Pause/Resume button
	timer.pauseResumeBtn = widget.NewButton("▶ Start", func() {
		if timer.isRunning {
			// Pause
			timer.isRunning = false
			timer.pauseResumeBtn.SetText("▶ Start")
			timer.stopTicker <- true
		} else {
			// Resume/Start
			timer.isRunning = true
			timer.pauseResumeBtn.SetText("⏸ Pause")
			go startTimer(timer)
		}
	})

	// Reset button
	resetBtn := widget.NewButton("↻ Reset", func() {
		if timer.isRunning {
			timer.isRunning = false
			timer.pauseResumeBtn.SetText("▶ Start")
			timer.stopTicker <- true
		}

		// Add elapsed time to task list before resetting
		if timer.taskName != "Select a task" && timer.elapsedTime > 0 {
			timer.taskListMutex.Lock()
			timer.taskList[timer.taskName] += timer.elapsedTime
			timer.taskListMutex.Unlock()
			
			if timer.statsUpdateFunc != nil {
				timer.statsUpdateFunc()
			}
		}

		timer.elapsedTime = 0
		timer.timeLabel.SetText("00:00:00")
	})

	buttonContainer := container.NewHBox(
		timer.pauseResumeBtn,
		resetBtn,
	)

	return container.NewVBox(
		taskNameLabel,
		timer.timeLabel,
		timer.taskSelector,
		buttonContainer,
	)
}

func startTimer(timer *TaskTimer) {
	timer.ticker = time.NewTicker(TickInterval)
	defer timer.ticker.Stop()

	for {
		select {
		case <-timer.stopTicker:
			return
		case <-timer.ticker.C:
			if timer.isRunning {
				timer.elapsedTime += TickInterval
				hours := timer.elapsedTime / time.Hour
				minutes := (timer.elapsedTime % time.Hour) / time.Minute
				seconds := (timer.elapsedTime % time.Minute) / time.Second
				timeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
				
				fyne.Do(func() {
					timer.timeLabel.SetText(timeStr)
				})
			}
		}
	}
}

func createDailyStatsContainer(timer *TaskTimer) fyne.CanvasObject {
	// Container to display daily stats
	statsBox := container.NewVBox()

	// Update function
	timer.statsUpdateFunc = func() {
		fyne.Do(func() {
			statsBox.RemoveAll()
			
			timer.taskListMutex.Lock()
			defer timer.taskListMutex.Unlock()
			
			if len(timer.taskList) == 0 {
				statsBox.Add(widget.NewLabel("No tasks completed yet"))
			} else {
				for taskName, duration := range timer.taskList {
					hours := duration / time.Hour
					minutes := (duration % time.Hour) / time.Minute
					seconds := (duration % time.Minute) / time.Second
					timeStr := fmt.Sprintf("%02d:%02d:%02d", hours, minutes, seconds)
					statLabel := widget.NewLabel(fmt.Sprintf("%s: %s", taskName, timeStr))
					statsBox.Add(statLabel)
				}
			}
		})
	}

	// Initialize with empty state
	statsBox.Add(widget.NewLabel("No tasks completed yet"))

	return container.NewScroll(statsBox)
}

func createAddTaskContainer(timer *TaskTimer) *fyne.Container {
	taskNameInput := widget.NewEntry()
	taskNameInput.PlaceHolder = "Enter task name (e.g., 'Write code')"

	addBtn := widget.NewButton("Add Task", func() {
		taskName := taskNameInput.Text
		if taskName != "" && taskName != "Select a task" {
			// Update task selector
			options := timer.taskSelector.Options
			if !contains(options, taskName) {
				options = append(options, taskName)
				timer.taskSelector.Options = options
			}
			taskNameInput.SetText("")
		}
	})

	return container.NewVBox(
		taskNameInput,
		addBtn,
	)
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}