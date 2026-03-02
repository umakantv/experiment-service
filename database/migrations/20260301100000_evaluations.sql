-- Migration: evaluations
-- Generated: 20260301100000 UTC

-- Create evaluations table
CREATE TABLE IF NOT EXISTS evaluations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    experiment_id INTEGER NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id TEXT NOT NULL,
    variant_name TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (experiment_id) REFERENCES experiments(id) ON DELETE CASCADE,
    UNIQUE(experiment_id, entity_type, entity_id)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_evaluations_entity ON evaluations(entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_evaluations_experiment ON evaluations(experiment_id);
