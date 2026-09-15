package store

import (
	"context"
	"errors"

	"github.com/venkxycodes/ab-experiments/model"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

type ExperimentStore interface {
	CreateExperiment(ctx context.Context, experiment model.Experiment) error
	GetExperiment(ctx context.Context, key string) (model.Experiment, error)
	ListExperiments(ctx context.Context) ([]model.Experiment, error)
	GetOrCreateAssignment(ctx context.Context, assignment model.Assignment) (model.Assignment, error)
}
