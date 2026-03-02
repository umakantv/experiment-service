-- Migration: experiments
-- Generated: 20260228100000 UTC

-- Create experiments table
CREATE TABLE IF NOT EXISTS experiments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    start_date DATETIME NOT NULL,
    end_date DATETIME NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create variants table
CREATE TABLE IF NOT EXISTS variants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    experiment_id INTEGER NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    traffic_ratio REAL NOT NULL DEFAULT 0.5,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (experiment_id) REFERENCES experiments(id) ON DELETE CASCADE,
    UNIQUE(experiment_id, name)
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_experiments_name ON experiments(name);
CREATE INDEX IF NOT EXISTS idx_experiments_dates ON experiments(start_date, end_date);
CREATE INDEX IF NOT EXISTS idx_variants_experiment_id ON variants(experiment_id);
