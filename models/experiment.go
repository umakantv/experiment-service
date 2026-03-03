package models

import "time"

// Experiment represents an A/B test experiment
type Experiment struct {
	ID             int       `json:"id" db:"id"`
	Name           string    `json:"name" db:"name"`
	Description    string    `json:"description" db:"description"`
	ExperimentType string    `json:"experiment_type" db:"experiment_type"`
	StartDate      time.Time `json:"start_date" db:"start_date"`
	EndDate        time.Time `json:"end_date" db:"end_date"`
	Variants       []Variant `json:"variants" db:"-"`
	Rules          []Rule    `json:"rules" db:"-"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time `json:"updated_at" db:"updated_at"`
}

// Variant represents a single variant within an experiment (e.g., control, treatment)
type Variant struct {
	ID                int       `json:"id" db:"id"`
	ExperimentID      int       `json:"experiment_id" db:"experiment_id"`
	Name              string    `json:"name" db:"name"`
	Description       string    `json:"description" db:"description"`
	TrafficPercentage float64   `json:"traffic_percentage" db:"traffic_percentage"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// Rule represents a prioritized condition-action pair for rule-based experiments
type Rule struct {
	ID           int    `json:"id" db:"id"`
	ExperimentID int    `json:"experiment_id" db:"experiment_id"`
	Priority     int    `json:"priority" db:"priority"`
	Condition    string `json:"condition" db:"condition"`
	Action       string `json:"action" db:"action"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// CreateExperimentRequest represents the request to create an experiment
type CreateExperimentRequest struct {
	Name           string                  `json:"name"`
	Description    string                  `json:"description"`
	ExperimentType string                  `json:"experiment_type"`
	StartDate      time.Time                `json:"start_date"`
	EndDate        time.Time                `json:"end_date"`
	Variants       []CreateVariantRequest  `json:"variants"`
	Rules          []CreateRuleRequest     `json:"rules"`
}

// CreateVariantRequest represents the request to create a variant
type CreateVariantRequest struct {
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	TrafficPercentage float64 `json:"traffic_percentage"`
}

// CreateRuleRequest represents the request to create a rule
type CreateRuleRequest struct {
	Priority  int    `json:"priority"`
	Condition string `json:"condition"`
	Action    string `json:"action"`
}

// UpdateExperimentRequest represents the request to update an experiment
type UpdateExperimentRequest struct {
	Name           *string                `json:"name,omitempty"`
	Description    *string                `json:"description,omitempty"`
	ExperimentType *string                `json:"experiment_type,omitempty"`
	StartDate      *time.Time             `json:"start_date,omitempty"`
	EndDate        *time.Time             `json:"end_date,omitempty"`
	Variants       []UpdateVariantRequest `json:"variants,omitempty"`
	Rules          []UpdateRuleRequest    `json:"rules,omitempty"`
}

// UpdateVariantRequest represents the request to update a variant
type UpdateVariantRequest struct {
	ID                int      `json:"id"`
	Name              *string  `json:"name,omitempty"`
	Description       *string  `json:"description,omitempty"`
	TrafficPercentage *float64 `json:"traffic_percentage,omitempty"`
}

// UpdateRuleRequest represents the request to update a rule
type UpdateRuleRequest struct {
	ID        int     `json:"id,omitempty"`
	Priority  *int    `json:"priority,omitempty"`
	Condition *string `json:"condition,omitempty"`
	Action    *string `json:"action,omitempty"`
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
