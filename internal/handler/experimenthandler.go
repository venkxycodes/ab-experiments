package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/venkxycodes/ab-experiments/internal/service"
	"github.com/venkxycodes/ab-experiments/internal/store"
	"github.com/venkxycodes/ab-experiments/model"
)

type ExperimentHandler struct {
	service service.ExperimentService
}

func NewExperimentHandler(experimentService service.ExperimentService) *ExperimentHandler {
	return &ExperimentHandler{service: experimentService}
}

func (h *ExperimentHandler) CreateExperiment(c *gin.Context) {
	var experiment model.Experiment
	if err := c.ShouldBindJSON(&experiment); err != nil {
		writeError(c, service.ErrInvalidExperiment)
		return
	}
	if err := h.service.CreateExperiment(c.Request.Context(), experiment); err != nil {
		writeError(c, err)
		return
	}
	experiment.Key = strings.TrimSpace(experiment.Key)
	created, err := h.service.GetExperiment(c.Request.Context(), experiment.Key)
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusCreated, created)
}

func (h *ExperimentHandler) ListExperiments(c *gin.Context) {
	experiments, err := h.service.ListExperiments(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, experiments)
}

func (h *ExperimentHandler) GetExperiment(c *gin.Context) {
	experiment, err := h.service.GetExperiment(c.Request.Context(), c.Param("key"))
	if err != nil {
		writeError(c, err)
		return
	}
	c.JSON(http.StatusOK, experiment)
}

type resolveRequest struct {
	SubjectID string `json:"subject_id"`
	Eligible  *bool  `json:"eligible"`
}

type resolveResponse struct {
	ExperimentKey string `json:"experiment_key"`
	Eligible      bool   `json:"eligible"`
	Cohort        string `json:"cohort,omitempty"`
	Bucket        *int   `json:"bucket,omitempty"`
	AssignmentID  string `json:"assignment_id,omitempty"`
}

func (h *ExperimentHandler) Resolve(c *gin.Context) {
	var request resolveRequest
	if err := c.ShouldBindJSON(&request); err != nil || request.Eligible == nil {
		writeError(c, service.ErrInvalidSubject)
		return
	}
	result, err := h.service.Resolve(c.Request.Context(), c.Param("key"), request.SubjectID, *request.Eligible)
	if err != nil {
		writeError(c, err)
		return
	}

	response := resolveResponse{
		ExperimentKey: c.Param("key"),
		Eligible:      result.Eligible,
	}
	if result.Assignment != nil {
		response.Cohort = result.Assignment.Cohort
		response.Bucket = &result.Assignment.Bucket
		response.AssignmentID = result.Assignment.ID
	}
	c.JSON(http.StatusOK, response)
}

func writeError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	switch {
	case errors.Is(err, service.ErrInvalidExperiment), errors.Is(err, service.ErrInvalidSubject):
		status = http.StatusBadRequest
	case errors.Is(err, store.ErrNotFound):
		status = http.StatusNotFound
	case errors.Is(err, store.ErrConflict):
		status = http.StatusConflict
	case errors.Is(err, service.ErrExperimentNotRunning):
		status = http.StatusConflict
	}
	c.JSON(status, gin.H{"error": err.Error()})
}
