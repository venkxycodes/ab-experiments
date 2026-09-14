package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash/fnv"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/venkxycodes/ab-experiments/internal/store"
	"github.com/venkxycodes/ab-experiments/model"
)

var (
	ErrInvalidExperiment    = errors.New("invalid experiment")
	ErrExperimentNotRunning = errors.New("experiment is not running")
	ErrInvalidSubject       = errors.New("subject_id is required")
)

type ExperimentService interface {
	CreateExperiment(experiment model.Experiment) error
	GetExperiment(key string) (model.Experiment, error)
	ListExperiments() []model.Experiment
	Resolve(key, subjectID string, eligible bool) (model.ResolveResult, error)
}

type experimentService struct {
	repository store.ExperimentStore
	mu         sync.Mutex
}

func NewExperimentService(repository store.ExperimentStore) ExperimentService {
	return &experimentService{repository: repository}
}

func (s *experimentService) CreateExperiment(experiment model.Experiment) error {
	if err := validateExperiment(&experiment); err != nil {
		return err
	}
	return s.repository.CreateExperiment(experiment)
}

func (s *experimentService) GetExperiment(key string) (model.Experiment, error) {
	return s.repository.GetExperiment(key)
}

func (s *experimentService) ListExperiments() []model.Experiment {
	return s.repository.ListExperiments()
}

func (s *experimentService) Resolve(key, subjectID string, eligible bool) (model.ResolveResult, error) {
	if strings.TrimSpace(subjectID) == "" {
		return model.ResolveResult{}, ErrInvalidSubject
	}
	experiment, err := s.repository.GetExperiment(key)
	if err != nil {
		return model.ResolveResult{}, err
	}
	if experiment.Status != model.StatusRunning {
		return model.ResolveResult{}, ErrExperimentNotRunning
	}
	if !eligible {
		return model.ResolveResult{Eligible: false}, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if assignment, err := s.repository.GetAssignment(key, subjectID); err == nil {
		return model.ResolveResult{Eligible: true, Assignment: &assignment}, nil
	}

	bucket := stableBucket(key, subjectID)
	assignment := model.Assignment{
		ID:            assignmentID(key, subjectID),
		ExperimentKey: key,
		SubjectID:     subjectID,
		Cohort:        chooseCohort(experiment.Variants, bucket),
		Bucket:        bucket,
		AssignedAt:    time.Now().UTC(),
	}

	if err := s.repository.CreateAssignment(assignment); err != nil {
		if errors.Is(err, store.ErrConflict) {
			existing, getErr := s.repository.GetAssignment(key, subjectID)
			if getErr == nil {
				return model.ResolveResult{Eligible: true, Assignment: &existing}, nil
			}
		}
		return model.ResolveResult{}, err
	}
	return model.ResolveResult{Eligible: true, Assignment: &assignment}, nil
}

func validateExperiment(experiment *model.Experiment) error {
	experiment.Key = strings.TrimSpace(experiment.Key)
	if experiment.Key == "" || len(experiment.Variants) < 2 {
		return ErrInvalidExperiment
	}
	if experiment.Status == "" {
		experiment.Status = model.StatusRunning
	}
	switch experiment.Status {
	case model.StatusDraft, model.StatusRunning, model.StatusPaused, model.StatusCompleted:
	default:
		return ErrInvalidExperiment
	}

	seen := make(map[string]struct{}, len(experiment.Variants))
	total := 0
	for _, variant := range experiment.Variants {
		if strings.TrimSpace(variant.Key) == "" || variant.Weight < 0 || variant.Weight%10 != 0 {
			return ErrInvalidExperiment
		}
		if _, exists := seen[variant.Key]; exists {
			return ErrInvalidExperiment
		}
		seen[variant.Key] = struct{}{}
		total += variant.Weight
	}
	if total != 100 {
		return ErrInvalidExperiment
	}
	return nil
}

func stableBucket(experimentKey, subjectID string) int {
	if numericID, err := strconv.ParseUint(subjectID, 10, 64); err == nil {
		return int(numericID % 10)
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(experimentKey + ":" + subjectID))
	return int(hasher.Sum32() % 10)
}

func chooseCohort(variants []model.Variant, bucket int) string {
	cursor := 0
	for _, variant := range variants {
		cursor += variant.Weight
		if bucket < cursor/10 {
			return variant.Key
		}
	}
	return variants[len(variants)-1].Key
}

func assignmentID(experimentKey, subjectID string) string {
	sum := sha256.Sum256([]byte(experimentKey + ":" + subjectID))
	return "assignment_" + hex.EncodeToString(sum[:8])
}
