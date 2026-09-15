package store

import (
	"context"
	"errors"
	"sort"
	"sync"

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

type InMemoryExperimentStore struct {
	mu          sync.RWMutex
	experiments map[string]model.Experiment
	assignments map[string]model.Assignment
}

func NewInMemoryExperimentStore() ExperimentStore {
	return &InMemoryExperimentStore{
		experiments: make(map[string]model.Experiment),
		assignments: make(map[string]model.Assignment),
	}
}

func (s *InMemoryExperimentStore) CreateExperiment(_ context.Context, experiment model.Experiment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.experiments[experiment.Key]; exists {
		return ErrConflict
	}
	s.experiments[experiment.Key] = cloneExperiment(experiment)
	return nil
}

func (s *InMemoryExperimentStore) GetExperiment(_ context.Context, key string) (model.Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	experiment, exists := s.experiments[key]
	if !exists {
		return model.Experiment{}, ErrNotFound
	}
	return cloneExperiment(experiment), nil
}

func (s *InMemoryExperimentStore) ListExperiments(_ context.Context) ([]model.Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	experiments := make([]model.Experiment, 0, len(s.experiments))
	for _, experiment := range s.experiments {
		experiments = append(experiments, cloneExperiment(experiment))
	}
	sort.Slice(experiments, func(i, j int) bool {
		return experiments[i].Key < experiments[j].Key
	})
	return experiments, nil
}

func (s *InMemoryExperimentStore) GetOrCreateAssignment(_ context.Context, assignment model.Assignment) (model.Assignment, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := assignmentKey(assignment.ExperimentKey, assignment.SubjectID)
	if existing, exists := s.assignments[key]; exists {
		return existing, nil
	}
	s.assignments[key] = assignment
	return assignment, nil
}

func assignmentKey(experimentKey, subjectID string) string {
	return experimentKey + ":" + subjectID
}

func cloneExperiment(experiment model.Experiment) model.Experiment {
	experiment.Variants = append([]model.Variant(nil), experiment.Variants...)
	return experiment
}
