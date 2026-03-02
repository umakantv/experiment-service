package models

import "time"

// Experiment represents an A/B test experiment
type Experiment struct {
	ID          int       `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description" db:"description"`
	StartDate   time.Time `json:"start_date" db:"start_date"`
	EndDate     time.Time `json:"end_date" db:"end_date"`
	Variants    []Variant `json:"variants" db:"-"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// Variant represents a single variant within an experiment (e.g., control, treatment)
type Variant struct {
	ID           int     `json:"id" db:"id"`
	ExperimentID int     `json:"experiment_id" db:"experiment_id"`
	Name         string  `json:"name" db:"name"`
	Description  string  `json:"description" db:"description"`
	TrafficRatio float64 `json:"traffic_ratio" db:"traffic_ratio"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// CreateExperimentRequest represents the request to create an experiment
type CreateExperimentRequest struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Variants    []CreateVariantRequest `json:"variants"`
}

// CreateVariantRequest represents the request to create a variant
type CreateVariantRequest struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	TrafficRatio float64 `json:"traffic_ratio"`
}

// UpdateExperimentRequest represents the request to update an experiment
type UpdateExperimentRequest struct {
	Name        *string    `json:"name,omitempty"`
	Description *string    `json:"description,omitempty"`
	StartDate   *time.Time `json:"start_date,omitempty"`
	EndDate     *time.Time `json:"end_date,omitempty"`
}

// UpdateVariantRequest represents the request to update a variant
type UpdateVariantRequest struct {
	Name         *string  `json:"name,omitempty"`
	Description  *string  `json:"description,omitempty"`
	TrafficRatio *float64 `json:"traffic_ratio,omitempty"`
}

// EvaluateRequest represents the request to evaluate an experiment
type EvaluateRequest struct {
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
}

// EvaluateResponse represents the response from evaluating an experiment
type EvaluateResponse struct {
	ExperimentID int    `json:"experiment_id"`
	VariantName  string `json:"variant_name"`
	EntityType   string `json:"entity_type"`
	EntityID     string `json:"entity_id"`
}
