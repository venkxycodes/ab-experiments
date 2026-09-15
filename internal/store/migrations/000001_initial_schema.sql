CREATE TABLE experiments (
    key VARCHAR(128) PRIMARY KEY,
    status VARCHAR(32) NOT NULL,
    variants JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE experiment_assignments (
    id VARCHAR(64) PRIMARY KEY,
    experiment_key VARCHAR(128) NOT NULL REFERENCES experiments(key),
    subject_id VARCHAR(256) NOT NULL,
    cohort VARCHAR(64) NOT NULL,
    bucket SMALLINT NOT NULL CHECK (bucket >= 0 AND bucket < 10),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT experiment_assignments_experiment_subject_key
        UNIQUE (experiment_key, subject_id)
);

CREATE INDEX experiment_assignments_experiment_key_idx
    ON experiment_assignments (experiment_key);
