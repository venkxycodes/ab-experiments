package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/venkxycodes/ab-experiments/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

type PostgresConfig struct {
	MaxOpenConns int
	MaxIdleConns int
}

type PostgresExperimentStore struct {
	db    *gorm.DB
	sqlDB *sql.DB
}

type experimentRecord struct {
	Key       string    `gorm:"column:key;primaryKey;size:128"`
	Status    string    `gorm:"column:status;size:32;not null"`
	Variants  string    `gorm:"column:variants;type:jsonb;not null"`
	CreatedAt time.Time `gorm:"column:created_at;not null"`
	UpdatedAt time.Time `gorm:"column:updated_at;not null"`
}

type assignmentRecord struct {
	ID            string    `gorm:"column:id;primaryKey;size:64"`
	ExperimentKey string    `gorm:"column:experiment_key;size:128;not null;uniqueIndex:idx_assignment_subject"`
	SubjectID     string    `gorm:"column:subject_id;size:256;not null;uniqueIndex:idx_assignment_subject"`
	Cohort        string    `gorm:"column:cohort;size:64;not null"`
	Bucket        int       `gorm:"column:bucket;not null"`
	AssignedAt    time.Time `gorm:"column:assigned_at;not null"`
}

func (experimentRecord) TableName() string {
	return "experiments"
}

func (assignmentRecord) TableName() string {
	return "experiment_assignments"
}

func OpenPostgresExperimentStore(dsn string, cfg PostgresConfig) (*PostgresExperimentStore, error) {
	if dsn == "" {
		return nil, errors.New("DATABASE_URL is required for postgres storage")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt:            true,
		SkipDefaultTransaction: true,
		Logger:                  logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}

	store := &PostgresExperimentStore{db: db, sqlDB: sqlDB}
	if err := db.AutoMigrate(&experimentRecord{}, &assignmentRecord{}); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return store, nil
}

func (s *PostgresExperimentStore) Close() error {
	return s.sqlDB.Close()
}

func (s *PostgresExperimentStore) CreateExperiment(ctx context.Context, experiment model.Experiment) error {
	variants, err := json.Marshal(experiment.Variants)
	if err != nil {
		return err
	}

	record := experimentRecord{
		Key:      experiment.Key,
		Status:   string(experiment.Status),
		Variants: string(variants),
	}
	result := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&record)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrConflict
	}
	return nil
}

func (s *PostgresExperimentStore) GetExperiment(ctx context.Context, key string) (model.Experiment, error) {
	var record experimentRecord
	result := s.db.WithContext(ctx).
		Select("key", "status", "variants").
		Where("key = ?", key).
		First(&record)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return model.Experiment{}, ErrNotFound
	}
	if result.Error != nil {
		return model.Experiment{}, result.Error
	}
	return record.toModel()
}

func (s *PostgresExperimentStore) ListExperiments(ctx context.Context) ([]model.Experiment, error) {
	var records []experimentRecord
	result := s.db.WithContext(ctx).
		Select("key", "status", "variants").
		Order("key ASC").
		Find(&records)
	if result.Error != nil {
		return nil, result.Error
	}

	experiments := make([]model.Experiment, 0, len(records))
	for _, record := range records {
		experiment, err := record.toModel()
		if err != nil {
			return nil, err
		}
		experiments = append(experiments, experiment)
	}
	return experiments, nil
}

func (s *PostgresExperimentStore) GetOrCreateAssignment(ctx context.Context, assignment model.Assignment) (model.Assignment, error) {
	record := assignmentRecord{
		ID:            assignment.ID,
		ExperimentKey: assignment.ExperimentKey,
		SubjectID:     assignment.SubjectID,
		Cohort:        assignment.Cohort,
		Bucket:        assignment.Bucket,
		AssignedAt:    assignment.AssignedAt,
	}

	result := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{DoNothing: true}).
		Create(&record)
	if result.Error != nil {
		return model.Assignment{}, result.Error
	}
	if result.RowsAffected == 0 {
		var existing assignmentRecord
		query := s.db.WithContext(ctx).
			Select("id", "experiment_key", "subject_id", "cohort", "bucket", "assigned_at").
			Where("experiment_key = ? AND subject_id = ?", assignment.ExperimentKey, assignment.SubjectID).
			First(&existing)
		if errors.Is(query.Error, gorm.ErrRecordNotFound) {
			return model.Assignment{}, ErrNotFound
		}
		if query.Error != nil {
			return model.Assignment{}, query.Error
		}
		return existing.toModel(), nil
	}
	return record.toModel(), nil
}

func (r experimentRecord) toModel() (model.Experiment, error) {
	var variants []model.Variant
	if err := json.Unmarshal([]byte(r.Variants), &variants); err != nil {
		return model.Experiment{}, err
	}
	return model.Experiment{
		Key:      r.Key,
		Status:   model.ExperimentStatus(r.Status),
		Variants: variants,
	}, nil
}

func (r assignmentRecord) toModel() (model.Assignment, error) {
	return model.Assignment{
		ID:            r.ID,
		ExperimentKey: r.ExperimentKey,
		SubjectID:     r.SubjectID,
		Cohort:        r.Cohort,
		Bucket:        r.Bucket,
		AssignedAt:    r.AssignedAt,
	}, nil
}
