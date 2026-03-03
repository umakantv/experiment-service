-- Migration: experiment_rules
-- Generated: 20260304100000 UTC

-- Create experiment_rules table
CREATE TABLE IF NOT EXISTS experiment_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    experiment_id INTEGER NOT NULL,
    priority INTEGER NOT NULL,
    condition TEXT NOT NULL,
    action TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (experiment_id) REFERENCES experiments(id) ON DELETE CASCADE,
    UNIQUE(experiment_id, priority)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_experiment_rules_experiment_id ON experiment_rules(experiment_id);
CREATE INDEX IF NOT EXISTS idx_experiment_rules_priority ON experiment_rules(experiment_id, priority);