package service

import (
	"testing"

	"github.com/venkxycodes/ab-experiments/internal/store"
	"github.com/venkxycodes/ab-experiments/model"
)

func TestResolvePersistsStableAssignment(t *testing.T) {
	svc := NewExperimentService(store.NewInMemoryExperimentStore())
	err := svc.CreateExperiment(model.Experiment{
		Key: "onboarding_sku_discovery", Status: model.StatusRunning,
		Variants: []model.Variant{{Key: "control", Weight: 50}, {Key: "treatment", Weight: 50}},
	})
	if err != nil {
		t.Fatalf("create experiment: %v", err)
	}

	first, err := svc.Resolve("onboarding_sku_discovery", "123", true)
	if err != nil {
		t.Fatalf("first resolve: %v", err)
	}
	second, err := svc.Resolve("onboarding_sku_discovery", "123", true)
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
	svc := NewExperimentService(store.NewInMemoryExperimentStore())
	err := svc.CreateExperiment(model.Experiment{
		Key: "onboarding_sku_discovery", Status: model.StatusRunning,
		Variants: []model.Variant{{Key: "control", Weight: 50}, {Key: "treatment", Weight: 50}},
	})
	if err != nil {
		t.Fatalf("create experiment: %v", err)
	}

	result, err := svc.Resolve("onboarding_sku_discovery", "123", false)
	if err != nil {
		t.Fatalf("resolve ineligible subject: %v", err)
	}
	if result.Eligible || result.Assignment != nil {
		t.Fatal("ineligible subject must not receive an assignment")
	}
}
