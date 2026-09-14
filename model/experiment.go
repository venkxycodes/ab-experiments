package model

import "time"

type ExperimentStatus string

const (
	StatusDraft     ExperimentStatus = "draft"
	StatusRunning   ExperimentStatus = "running"
	StatusPaused    ExperimentStatus = "paused"
	StatusCompleted ExperimentStatus = "completed"
)

type Variant struct {
	Key    string `json:"key"`
	Weight int    `json:"weight"`
}

type Experiment struct {
	Key      string          `json:"key"`
	Status   ExperimentStatus `json:"status"`
	Variants []Variant       `json:"variants"`
}

type Assignment struct {
	ID            string    `json:"assignment_id"`
	ExperimentKey string    `json:"experiment_key"`
	SubjectID     string    `json:"subject_id"`
	Cohort        string    `json:"cohort"`
	Bucket        int       `json:"bucket"`
	AssignedAt    time.Time `json:"assigned_at"`
}

type ResolveResult struct {
	Eligible   bool
	Assignment *Assignment
}
