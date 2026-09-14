package store

import (
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
	CreateExperiment(experiment model.Experiment) error
	GetExperiment(key string) (model.Experiment, error)
	ListExperiments() []model.Experiment
	GetAssignment(experimentKey, subjectID string) (model.Assignment, error)
	CreateAssignment(assignment model.Assignment) error
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

func (s *InMemoryExperimentStore) CreateExperiment(experiment model.Experiment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.experiments[experiment.Key]; exists {
		return ErrConflict
	}
	s.experiments[experiment.Key] = cloneExperiment(experiment)
	return nil
}

func (s *InMemoryExperimentStore) GetExperiment(key string) (model.Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	experiment, exists := s.experiments[key]
	if !exists {
		return model.Experiment{}, ErrNotFound
	}
	return cloneExperiment(experiment), nil
}

func (s *InMemoryExperimentStore) ListExperiments() []model.Experiment {
	s.mu.RLock()
	defer s.mu.RUnlock()
	experiments := make([]model.Experiment, 0, len(s.experiments))
	for _, experiment := range s.experiments {
		experiments = append(experiments, cloneExperiment(experiment))
	}
	sort.Slice(experiments, func(i, j int) bool {
		return experiments[i].Key < experiments[j].Key
	})
	return experiments
}

func (s *InMemoryExperimentStore) GetAssignment(experimentKey, subjectID string) (model.Assignment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	assignment, exists := s.assignments[assignmentKey(experimentKey, subjectID)]
	if !exists {
		return model.Assignment{}, ErrNotFound
	}
	return assignment, nil
}

func (s *InMemoryExperimentStore) CreateAssignment(assignment model.Assignment) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := assignmentKey(assignment.ExperimentKey, assignment.SubjectID)
	if _, exists := s.assignments[key]; exists {
		return ErrConflict
	}
	s.assignments[key] = assignment
	return nil
}

func assignmentKey(experimentKey, subjectID string) string {
	return experimentKey + ":" + subjectID
}

func cloneExperiment(experiment model.Experiment) model.Experiment {
	experiment.Variants = append([]model.Variant(nil), experiment.Variants...)
	return experiment
}
