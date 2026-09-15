package service

import (
	"context"
	"testing"

	"github.com/venkxycodes/ab-experiments/internal/store"
	"github.com/venkxycodes/ab-experiments/model"
)

func TestResolvePersistsStableAssignment(t *testing.T) {
	svc := NewExperimentService(newStubStore())
	ctx := context.Background()
	err := svc.CreateExperiment(ctx, model.Experiment{
		Key: "onboarding_sku_discovery", Status: model.StatusRunning,
		Variants: []model.Variant{{Key: "control", Weight: 50}, {Key: "treatment", Weight: 50}},
	})
	if err != nil {
		t.Fatalf("create experiment: %v", err)
	}

	first, err := svc.Resolve(ctx, "onboarding_sku_discovery", "123", true)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	second, err := svc.Resolve(ctx, "onboarding_sku_discovery", "123", true)
	if err != nil {
		t.Fatalf("second resolve: %v", err)
	}
	if first.Assignment == nil || second.Assignment == nil {
		t.Fatal("expected assignments")
	}
	if first.Assignment.Cohort != second.Assignment.Cohort || first.Assignment.ID != second.Assignment.ID {
		t.Fatal("assignment was not stable")
	}
	if first.Assignment.Bucket != 3 {
		t.Fatalf("expected bucket 3, got %d", first.Assignment.Bucket)
	}
}

func TestIneligibleSubjectIsNotAssigned(t *testing.T) {
	svc := NewExperimentService(newStubStore())
	ctx := context.Background()
	err := svc.CreateExperiment(ctx, model.Experiment{
		Key: "onboarding_sku_discovery", Status: model.StatusRunning,
		Variants: []model.Variant{{Key: "control", Weight: 50}, {Key: "treatment", Weight: 50}},
	})
	if err != nil {
		t.Fatalf("create experiment: %v", err)
	}

	result, err := svc.Resolve(ctx, "onboarding_sku_discovery", "123", false)
	if err != nil {
		t.Fatalf("resolve ineligible subject: %v", err)
	}
	if result.Eligible || result.Assignment != nil {
		t.Fatal("ineligible subject must not receive an assignment")
	}
}

type stubStore struct {
	experiments map[string]model.Experiment
	assignments map[string]model.Assignment
}

func newStubStore() *stubStore {
	return &stubStore{
		experiments: make(map[string]model.Experiment),
		assignments: make(map[string]model.Assignment),
	}
}

func (s *stubStore) CreateExperiment(_ context.Context, experiment model.Experiment) error {
	if _, exists := s.experiments[experiment.Key]; exists {
		return store.ErrConflict
	}
	s.experiments[experiment.Key] = experiment
	return nil
}

func (s *stubStore) GetExperiment(_ context.Context, key string) (model.Experiment, error) {
	experiment, exists := s.experiments[key]
	if !exists {
		return model.Experiment{}, store.ErrNotFound
	}
	return experiment, nil
}

func (s *stubStore) ListExperiments(_ context.Context) ([]model.Experiment, error) {
	experiments := make([]model.Experiment, 0, len(s.experiments))
	for _, experiment := range s.experiments {
		experiments = append(experiments, experiment)
	}
	return experiments, nil
}

func (s *stubStore) GetOrCreateAssignment(_ context.Context, assignment model.Assignment) (model.Assignment, error) {
	key := assignment.ExperimentKey + ":" + assignment.SubjectID
	if existing, exists := s.assignments[key]; exists {
		return existing, nil
	}
	s.assignments[key] = assignment
	return assignment, nil
}
