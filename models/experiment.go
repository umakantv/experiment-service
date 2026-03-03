package models

import (
	"encoding/json"
	"time"
)

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
	ID           int       `json:"id" db:"id"`
	ExperimentID int       `json:"experiment_id" db:"experiment_id"`
	Priority     int       `json:"priority" db:"priority"`
	Condition    string    `json:"condition" db:"condition"`
	Action       string    `json:"action" db:"action"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// ActionType defines the type of action a rule can perform
type ActionType string

const (
	ActionAssignVariant   ActionType = "assign_variant"
	ActionEnableExperiment ActionType = "enable_experiment"
	ActionSetPayload      ActionType = "set_payload"
)

// RuleAction represents the parsed action from a rule
type RuleAction struct {
	Type     ActionType              `json:"type"`
	Variant  string                  `json:"variant,omitempty"`
	Payload  map[string]interface{}  `json:"payload,omitempty"`
}

// ParseAction parses the action string into a structured RuleAction
func (r *Rule) ParseAction() (*RuleAction, error) {
	var rawAction map[string]interface{}
	if err := json.Unmarshal([]byte(r.Action), &rawAction); err != nil {
		return nil, err
	}
	
	action := &RuleAction{}
	
	if actionType, ok := rawAction["action"].(string); ok {
		action.Type = ActionType(actionType)
	} else {
		return nil, nil // Legacy format: plain string action
	}
	
	switch action.Type {
	case ActionAssignVariant:
		if variant, ok := rawAction["variant"].(string); ok {
			action.Variant = variant
		}
	case ActionSetPayload:
		if payload, ok := rawAction["payload"].(map[string]interface{}); ok {
			action.Payload = payload
		} else if value, ok := rawAction["value"].(map[string]interface{}); ok {
			action.Payload = value
		}
	}
	
	return action, nil
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
	EntityType string                 `json:"entity_type"`
	EntityID   string                 `json:"entity_id"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// EvaluateResponse represents the response from evaluating an experiment
type EvaluateResponse struct {
	ExperimentID   int                    `json:"experiment_id"`
	VariantName    string                 `json:"variant_name,omitempty"`
	EntityType     string                 `json:"entity_type"`
	EntityID       string                 `json:"entity_id"`
	Payload        map[string]interface{} `json:"payload,omitempty"`
	MatchedRule    *MatchedRuleInfo       `json:"matched_rule,omitempty"`
}

// MatchedRuleInfo contains information about the matched rule
type MatchedRuleInfo struct {
	Priority int    `json:"priority"`
	Action   string `json:"action"`
}
