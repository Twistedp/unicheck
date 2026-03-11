package model

import "time"

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
