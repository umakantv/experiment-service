-- Migration: add experiment_type to experiments
-- Generated: 20260303100000 UTC

ALTER TABLE experiments
ADD COLUMN experiment_type TEXT NOT NULL DEFAULT 'ramp-up-percentage';
