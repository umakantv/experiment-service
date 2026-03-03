package handlers

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"oauth-service/models"

	"github.com/gorilla/mux"
	"github.com/jmoiron/sqlx"
	"github.com/umakantv/go-utils/cache"
	"github.com/umakantv/go-utils/errs"
	"github.com/umakantv/go-utils/httpserver"
	logger "github.com/umakantv/go-utils/logger"
	"go.uber.org/zap"
)

// Handler handles experiment-related operations
type Handler struct {
	db    *sqlx.DB
	cache cache.Cache
}

// NewHandler creates a new handler
func NewHandler(db *sqlx.DB, cache cache.Cache) *Handler {
	return &Handler{
		db:    db,
		cache: cache,
	}
}

// logRequest logs the request with the specified format
func (h *Handler) logRequest(ctx context.Context, level string, message string, fields ...zap.Field) {
	routeName := httpserver.GetRouteName(ctx)
	method := httpserver.GetRouteMethod(ctx)
	path := httpserver.GetRoutePath(ctx)
	auth := httpserver.GetRequestAuth(ctx)

	// Build log message
	logMsg := time.Now().Format("2006-01-02 15:04:05") + " - " + routeName + " - " + method + " - " + path
	if auth != nil {
		logMsg += " - client:" + auth.Client
	}

	// Add custom fields
	allFields := append([]zap.Field{
		zap.String("route", routeName),
		zap.String("method", method),
		zap.String("path", path),
	}, fields...)

	switch level {
	case "info":
		logger.Info(logMsg, allFields...)
	case "error":
		logger.Error(logMsg, allFields...)
	case "debug":
		logger.Debug(logMsg, allFields...)
	}
}

// HealthCheck handles GET /health - health check endpoint
func (h *Handler) HealthCheck(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "service": "experiment-service"}`))
}

// ListExperiments handles GET /experiments - list all experiments
func (h *Handler) ListExperiments(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	h.logRequest(ctx, "info", "Listing experiments")

	// Try cache first
	cacheKey := "experiments:list"
	if cached, err := h.cache.Get(cacheKey); err == nil {
		h.logRequest(ctx, "debug", "Serving from cache")
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached.([]byte))
		return
	}

	// Query experiments
	rows, err := h.db.Query(`
		SELECT id, name, description, experiment_type, start_date, end_date, created_at, updated_at 
		FROM experiments 
		ORDER BY created_at DESC`)
	if err != nil {
		h.logRequest(ctx, "error", "Failed to query experiments", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Database error"))
		return
	}
	defer rows.Close()

	experiments := []models.Experiment{}
	for rows.Next() {
		var exp models.Experiment
		err := rows.Scan(&exp.ID, &exp.Name, &exp.Description, &exp.ExperimentType, &exp.StartDate, &exp.EndDate, &exp.CreatedAt, &exp.UpdatedAt)
		if err != nil {
			h.logRequest(ctx, "error", "Failed to scan experiment", zap.Error(err))
			continue
		}
		experiments = append(experiments, exp)
	}

	// Load variants for each experiment
	for i := range experiments {
		variants, err := h.getVariantsByExperimentID(experiments[i].ID)
		if err != nil {
			h.logRequest(ctx, "error", "Failed to load variants", zap.Error(err), zap.Int("experiment_id", experiments[i].ID))
			continue
		}
		experiments[i].Variants = variants

		// Load rules for each experiment
		rules, err := h.getRulesByExperimentID(experiments[i].ID)
		if err != nil {
			h.logRequest(ctx, "error", "Failed to load rules", zap.Error(err), zap.Int("experiment_id", experiments[i].ID))
			continue
		}
		experiments[i].Rules = rules
	}

	// Cache the result
	response, _ := json.Marshal(experiments)
	h.cache.Set(cacheKey, response, 5*time.Minute)

	h.logRequest(ctx, "info", "Experiments retrieved successfully", zap.Int("count", len(experiments)))

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

// GetExperiment handles GET /experiments/{id} - get experiment by ID
func (h *Handler) GetExperiment(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logRequest(ctx, "error", "Invalid experiment ID", zap.String("id", idStr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Invalid experiment ID"))
		return
	}

	h.logRequest(ctx, "info", "Getting experiment", zap.Int("experiment_id", id))

	// Try cache first
	cacheKey := "experiment:" + idStr
	if cached, err := h.cache.Get(cacheKey); err == nil {
		h.logRequest(ctx, "debug", "Serving experiment from cache", zap.Int("experiment_id", id))
		w.Header().Set("Content-Type", "application/json")
		w.Write(cached.([]byte))
		return
	}

	// Query experiment
	var exp models.Experiment
	err = h.db.QueryRow(`
		SELECT id, name, description, experiment_type, start_date, end_date, created_at, updated_at 
		FROM experiments 
		WHERE id = ?`, id).
		Scan(&exp.ID, &exp.Name, &exp.Description, &exp.ExperimentType, &exp.StartDate, &exp.EndDate, &exp.CreatedAt, &exp.UpdatedAt)

	if err == sql.ErrNoRows {
		h.logRequest(ctx, "info", "Experiment not found", zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errs.NewNotFoundError("Experiment not found"))
		return
	}
	if err != nil {
		h.logRequest(ctx, "error", "Failed to query experiment", zap.Error(err), zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Database error"))
		return
	}

	// Load variants
	exp.Variants, err = h.getVariantsByExperimentID(exp.ID)
	if err != nil {
		h.logRequest(ctx, "error", "Failed to load variants", zap.Error(err), zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to load variants"))
		return
	}

	// Load rules
	exp.Rules, err = h.getRulesByExperimentID(exp.ID)
	if err != nil {
		h.logRequest(ctx, "error", "Failed to load rules", zap.Error(err), zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to load rules"))
		return
	}

	// Cache the result
	response, _ := json.Marshal(exp)
	h.cache.Set(cacheKey, response, 10*time.Minute)

	h.logRequest(ctx, "info", "Experiment retrieved successfully", zap.Int("experiment_id", id))

	w.Header().Set("Content-Type", "application/json")
	w.Write(response)
}

// CreateExperiment handles POST /experiments - create a new experiment
func (h *Handler) CreateExperiment(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req models.CreateExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logRequest(ctx, "error", "Invalid request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Invalid JSON"))
		return
	}

	// Validate input
	if req.Name == "" {
		h.logRequest(ctx, "error", "Missing required field: name")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Name is required"))
		return
	}

	if req.StartDate.IsZero() || req.EndDate.IsZero() {
		h.logRequest(ctx, "error", "Missing required fields: start_date or end_date")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Start date and end date are required"))
		return
	}

	if req.EndDate.Before(req.StartDate) {
		h.logRequest(ctx, "error", "End date before start date")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("End date must be after start date"))
		return
	}

	if req.ExperimentType == "" {
		req.ExperimentType = "ramp-up-percentage"
	}

	// Validate based on experiment type
	if req.ExperimentType == "ramp-up-percentage" {
		if len(req.Variants) < 2 {
			h.logRequest(ctx, "error", "Not enough variants")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errs.NewValidationError("At least 2 variants are required (e.g., control and treatment)"))
			return
		}

		// Validate traffic percentage sums to 100.0
		var totalTraffic float64
		for _, v := range req.Variants {
			if v.Name == "" {
				h.logRequest(ctx, "error", "Variant missing name")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("All variants must have a name"))
				return
			}
			if v.TrafficPercentage < 0 {
				h.logRequest(ctx, "error", "Variant traffic percentage negative", zap.Float64("traffic_percentage", v.TrafficPercentage))
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("Traffic percentage must be non-negative"))
				return
			}
			totalTraffic += v.TrafficPercentage
		}
		if totalTraffic < 99.9 || totalTraffic > 100.1 {
			h.logRequest(ctx, "error", "Invalid traffic percentage sum", zap.Float64("total", totalTraffic))
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errs.NewValidationError("Traffic percentages must sum to 100"))
			return
		}
	} else if req.ExperimentType == "rule-based-assignment" {
		if len(req.Rules) == 0 {
			h.logRequest(ctx, "error", "No rules provided for rule-based experiment")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errs.NewValidationError("At least one rule is required for rule-based experiments"))
			return
		}
		// Validate rules have unique priorities
		priorities := make(map[int]bool)
		for _, rule := range req.Rules {
			if rule.Priority <= 0 {
				h.logRequest(ctx, "error", "Invalid rule priority", zap.Int("priority", rule.Priority))
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("Rule priority must be a positive integer"))
				return
			}
			if priorities[rule.Priority] {
				h.logRequest(ctx, "error", "Duplicate rule priority", zap.Int("priority", rule.Priority))
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("Rule priorities must be unique"))
				return
			}
			priorities[rule.Priority] = true
			if rule.Condition == "" {
				h.logRequest(ctx, "error", "Rule missing condition")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("All rules must have a condition"))
				return
			}
			if rule.Action == "" {
				h.logRequest(ctx, "error", "Rule missing action")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("All rules must have an action"))
				return
			}
		}
		// Variants are optional for rule-based, but if provided, validate names
		for _, v := range req.Variants {
			if v.Name == "" {
				h.logRequest(ctx, "error", "Variant missing name")
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("All variants must have a name"))
				return
			}
		}
	}

	h.logRequest(ctx, "info", "Creating experiment", zap.String("name", req.Name))

	// Start transaction
	tx, err := h.db.Begin()
	if err != nil {
		h.logRequest(ctx, "error", "Failed to begin transaction", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Database error"))
		return
	}
	defer tx.Rollback()

	// Insert experiment
	result, err := tx.Exec(`
		INSERT INTO experiments (name, description, experiment_type, start_date, end_date, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		req.Name, req.Description, req.ExperimentType, req.StartDate, req.EndDate, time.Now(), time.Now())
	if err != nil {
		// Check for unique constraint violation (duplicate name)
		if strings.Contains(err.Error(), "UNIQUE constraint failed") {
			h.logRequest(ctx, "info", "Experiment name already exists", zap.String("name", req.Name))
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(errs.NewValidationError("Experiment with this name already exists"))
			return
		}
		h.logRequest(ctx, "error", "Failed to create experiment", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create experiment"))
		return
	}

	experimentID, _ := result.LastInsertId()

	// Insert variants
	createdAt := time.Now()
	updatedAt := createdAt
	exp := models.Experiment{
		ID:             int(experimentID),
		Name:           req.Name,
		Description:    req.Description,
		ExperimentType: req.ExperimentType,
		StartDate:      req.StartDate,
		EndDate:        req.EndDate,
		Variants:       make([]models.Variant, len(req.Variants)),
		Rules:          make([]models.Rule, len(req.Rules)),
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
	for index, v := range req.Variants {
		result, err := tx.Exec(`
			INSERT INTO variants (experiment_id, name, description, traffic_percentage, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			experimentID, v.Name, v.Description, v.TrafficPercentage, createdAt, updatedAt)
		if err != nil {
			h.logRequest(ctx, "error", "Failed to create variant", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create variant"))
			return
		}
		variantID, err := result.LastInsertId()
		if err != nil {
			h.logRequest(ctx, "error", "Failed to fetch variant ID", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create variant"))
			return
		}
		exp.Variants[index] = models.Variant{
			ID:                int(variantID),
			ExperimentID:      int(experimentID),
			Name:              v.Name,
			Description:       v.Description,
			TrafficPercentage: v.TrafficPercentage,
			CreatedAt:         createdAt,
			UpdatedAt:         updatedAt,
		}
	}

	// Insert rules
	for index, r := range req.Rules {
		result, err := tx.Exec(`
			INSERT INTO experiment_rules (experiment_id, priority, condition, action, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			experimentID, r.Priority, r.Condition, r.Action, createdAt, updatedAt)
		if err != nil {
			h.logRequest(ctx, "error", "Failed to create rule", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create rule"))
			return
		}
		ruleID, err := result.LastInsertId()
		if err != nil {
			h.logRequest(ctx, "error", "Failed to fetch rule ID", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create rule"))
			return
		}
		exp.Rules[index] = models.Rule{
			ID:           int(ruleID),
			ExperimentID: int(experimentID),
			Priority:     r.Priority,
			Condition:    r.Condition,
			Action:       r.Action,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		}
	}

	if err := tx.Commit(); err != nil {
		h.logRequest(ctx, "error", "Failed to commit transaction", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create experiment"))
		return
	}

	// Clear experiments list cache
	h.cache.Delete("experiments:list")

	h.logRequest(ctx, "info", "Experiment created successfully", zap.Int("experiment_id", int(experimentID)))

	// Return created experiment

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(exp)
}

// UpdateExperiment handles PUT /experiments/{id} - update experiment
func (h *Handler) UpdateExperiment(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logRequest(ctx, "error", "Invalid experiment ID", zap.String("id", idStr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Invalid experiment ID"))
		return
	}

	var req models.UpdateExperimentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logRequest(ctx, "error", "Invalid request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Invalid JSON"))
		return
	}

	h.logRequest(ctx, "info", "Updating experiment", zap.Int("experiment_id", id))

	// Build update query dynamically
	setParts := []string{}
	args := []interface{}{}

	if req.Name != nil {
		setParts = append(setParts, "name = ?")
		args = append(args, *req.Name)
	}
	if req.Description != nil {
		setParts = append(setParts, "description = ?")
		args = append(args, *req.Description)
	}
	if req.ExperimentType != nil {
		setParts = append(setParts, "experiment_type = ?")
		args = append(args, *req.ExperimentType)
	}
	if req.StartDate != nil {
		setParts = append(setParts, "start_date = ?")
		args = append(args, *req.StartDate)
	}
	if req.EndDate != nil {
		setParts = append(setParts, "end_date = ?")
		args = append(args, *req.EndDate)
	}

	hasExperimentUpdates := len(setParts) > 0
	hasVariantUpdates := len(req.Variants) > 0
	hasRuleUpdates := len(req.Rules) > 0

	if !hasExperimentUpdates && !hasVariantUpdates && !hasRuleUpdates {
		h.logRequest(ctx, "error", "No fields to update", zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("No fields to update"))
		return
	}

	tx, err := h.db.Begin()
	if err != nil {
		h.logRequest(ctx, "error", "Failed to begin transaction", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Database error"))
		return
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow("SELECT EXISTS(SELECT 1 FROM experiments WHERE id = ?)", id).Scan(&exists)
	if err != nil {
		h.logRequest(ctx, "error", "Failed to check experiment existence", zap.Error(err), zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Database error"))
		return
	}
	if !exists {
		h.logRequest(ctx, "info", "Experiment not found for update", zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errs.NewNotFoundError("Experiment not found"))
		return
	}

	if hasExperimentUpdates {
		setParts = append(setParts, "updated_at = ?")
		args = append(args, time.Now())
		args = append(args, id)

		query := "UPDATE experiments SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
		if _, err := tx.Exec(query, args...); err != nil {
			h.logRequest(ctx, "error", "Failed to update experiment", zap.Error(err), zap.Int("experiment_id", id))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to update experiment"))
			return
		}
	}

	if hasVariantUpdates {
		rows, err := tx.Query(`
			SELECT id, traffic_percentage
			FROM variants
			WHERE experiment_id = ?`, id)
		if err != nil {
			h.logRequest(ctx, "error", "Failed to load variants", zap.Error(err), zap.Int("experiment_id", id))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to load variants"))
			return
		}
		defer rows.Close()

		currentTraffic := map[int]float64{}
		for rows.Next() {
			var variantID int
			var trafficPercentage float64
			if err := rows.Scan(&variantID, &trafficPercentage); err != nil {
				h.logRequest(ctx, "error", "Failed to scan variants", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to load variants"))
				return
			}
			currentTraffic[variantID] = trafficPercentage
		}

		if len(currentTraffic) == 0 {
			h.logRequest(ctx, "error", "No variants found for experiment", zap.Int("experiment_id", id))
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errs.NewValidationError("No variants found for experiment"))
			return
		}

		updatedTraffic := map[int]float64{}
		for variantID, trafficPercentage := range currentTraffic {
			updatedTraffic[variantID] = trafficPercentage
		}

		for _, variantUpdate := range req.Variants {
			if variantUpdate.ID == 0 {
				h.logRequest(ctx, "error", "Variant update missing ID", zap.Int("experiment_id", id))
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("Variant ID is required"))
				return
			}
			if _, ok := currentTraffic[variantUpdate.ID]; !ok {
				h.logRequest(ctx, "error", "Variant does not belong to experiment", zap.Int("experiment_id", id), zap.Int("variant_id", variantUpdate.ID))
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("Variant does not belong to this experiment"))
				return
			}
			if variantUpdate.TrafficPercentage != nil {
				if *variantUpdate.TrafficPercentage < 0 {
					h.logRequest(ctx, "error", "Variant traffic percentage negative", zap.Float64("traffic_percentage", *variantUpdate.TrafficPercentage))
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(errs.NewValidationError("Traffic percentage must be non-negative"))
					return
				}
				updatedTraffic[variantUpdate.ID] = *variantUpdate.TrafficPercentage
			}
		}

		var totalTraffic float64
		for _, trafficPercentage := range updatedTraffic {
			totalTraffic += trafficPercentage
		}
		if totalTraffic < 99.9 || totalTraffic > 100.1 {
			h.logRequest(ctx, "error", "Invalid traffic percentage sum", zap.Float64("total", totalTraffic))
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errs.NewValidationError("Traffic percentages must sum to 100"))
			return
		}

		for _, variantUpdate := range req.Variants {
			variantSetParts := []string{}
			variantArgs := []interface{}{}
			if variantUpdate.Name != nil {
				variantSetParts = append(variantSetParts, "name = ?")
				variantArgs = append(variantArgs, *variantUpdate.Name)
			}
			if variantUpdate.Description != nil {
				variantSetParts = append(variantSetParts, "description = ?")
				variantArgs = append(variantArgs, *variantUpdate.Description)
			}
			if variantUpdate.TrafficPercentage != nil {
				variantSetParts = append(variantSetParts, "traffic_percentage = ?")
				variantArgs = append(variantArgs, *variantUpdate.TrafficPercentage)
			}
			if len(variantSetParts) == 0 {
				h.logRequest(ctx, "error", "Variant update missing fields", zap.Int("variant_id", variantUpdate.ID))
				w.WriteHeader(http.StatusBadRequest)
				json.NewEncoder(w).Encode(errs.NewValidationError("Variant update must include at least one field"))
				return
			}

			variantSetParts = append(variantSetParts, "updated_at = ?")
			variantArgs = append(variantArgs, time.Now(), variantUpdate.ID, id)

			variantQuery := "UPDATE variants SET " + strings.Join(variantSetParts, ", ") + " WHERE id = ? AND experiment_id = ?"
			if _, err := tx.Exec(variantQuery, variantArgs...); err != nil {
				h.logRequest(ctx, "error", "Failed to update variant", zap.Error(err), zap.Int("variant_id", variantUpdate.ID))
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to update variant"))
				return
			}
		}
	}

	// Handle rule updates
	if hasRuleUpdates {
		// Load existing rules
		rows, err := tx.Query(`
			SELECT id, priority
			FROM experiment_rules
			WHERE experiment_id = ?`, id)
		if err != nil {
			h.logRequest(ctx, "error", "Failed to load rules", zap.Error(err), zap.Int("experiment_id", id))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to load rules"))
			return
		}
		defer rows.Close()

		currentRules := map[int]int{} // ruleID -> priority
		for rows.Next() {
			var ruleID, priority int
			if err := rows.Scan(&ruleID, &priority); err != nil {
				h.logRequest(ctx, "error", "Failed to scan rules", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to load rules"))
				return
			}
			currentRules[ruleID] = priority
		}

		for _, ruleUpdate := range req.Rules {
			// If ID is provided, update existing rule
			if ruleUpdate.ID != 0 {
				if _, ok := currentRules[ruleUpdate.ID]; !ok {
					h.logRequest(ctx, "error", "Rule does not belong to experiment", zap.Int("experiment_id", id), zap.Int("rule_id", ruleUpdate.ID))
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(errs.NewValidationError("Rule does not belong to this experiment"))
					return
				}

				ruleSetParts := []string{}
				ruleArgs := []interface{}{}
				if ruleUpdate.Priority != nil {
					ruleSetParts = append(ruleSetParts, "priority = ?")
					ruleArgs = append(ruleArgs, *ruleUpdate.Priority)
				}
				if ruleUpdate.Condition != nil {
					ruleSetParts = append(ruleSetParts, "condition = ?")
					ruleArgs = append(ruleArgs, *ruleUpdate.Condition)
				}
				if ruleUpdate.Action != nil {
					ruleSetParts = append(ruleSetParts, "action = ?")
					ruleArgs = append(ruleArgs, *ruleUpdate.Action)
				}
				if len(ruleSetParts) == 0 {
					h.logRequest(ctx, "error", "Rule update missing fields", zap.Int("rule_id", ruleUpdate.ID))
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(errs.NewValidationError("Rule update must include at least one field"))
					return
				}

				ruleSetParts = append(ruleSetParts, "updated_at = ?")
				ruleArgs = append(ruleArgs, time.Now(), ruleUpdate.ID, id)

				ruleQuery := "UPDATE experiment_rules SET " + strings.Join(ruleSetParts, ", ") + " WHERE id = ? AND experiment_id = ?"
				if _, err := tx.Exec(ruleQuery, ruleArgs...); err != nil {
					h.logRequest(ctx, "error", "Failed to update rule", zap.Error(err), zap.Int("rule_id", ruleUpdate.ID))
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to update rule"))
					return
				}
			} else {
				// Create new rule (ID not provided)
				if ruleUpdate.Priority == nil || ruleUpdate.Condition == nil || ruleUpdate.Action == nil {
					h.logRequest(ctx, "error", "New rule missing required fields")
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(errs.NewValidationError("New rules must have priority, condition, and action"))
					return
				}
				_, err := tx.Exec(`
					INSERT INTO experiment_rules (experiment_id, priority, condition, action, created_at, updated_at) 
					VALUES (?, ?, ?, ?, ?, ?)`,
					id, *ruleUpdate.Priority, *ruleUpdate.Condition, *ruleUpdate.Action, time.Now(), time.Now())
				if err != nil {
					h.logRequest(ctx, "error", "Failed to create rule", zap.Error(err))
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create rule"))
					return
				}
			}
		}
	}

	if err := tx.Commit(); err != nil {
		h.logRequest(ctx, "error", "Failed to commit transaction", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to update experiment"))
		return
	}

	// Clear caches
	h.cache.Delete("experiments:list")
	h.cache.Delete("experiment:" + idStr)

	h.logRequest(ctx, "info", "Experiment updated successfully", zap.Int("experiment_id", id))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Experiment updated successfully"})
}

// DeleteExperiment handles DELETE /experiments/{id} - delete experiment
func (h *Handler) DeleteExperiment(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logRequest(ctx, "error", "Invalid experiment ID", zap.String("id", idStr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Invalid experiment ID"))
		return
	}

	h.logRequest(ctx, "info", "Deleting experiment", zap.Int("experiment_id", id))

	// Delete experiment (variants will be cascade deleted)
	result, err := h.db.Exec("DELETE FROM experiments WHERE id = ?", id)
	if err != nil {
		h.logRequest(ctx, "error", "Failed to delete experiment", zap.Error(err), zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to delete experiment"))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.logRequest(ctx, "info", "Experiment not found for deletion", zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errs.NewNotFoundError("Experiment not found"))
		return
	}

	// Clear caches
	h.cache.Delete("experiments:list")
	h.cache.Delete("experiment:" + idStr)

	h.logRequest(ctx, "info", "Experiment deleted successfully", zap.Int("experiment_id", id))

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"message": "Experiment deleted successfully"})
}

// getVariantsByExperimentID retrieves all variants for a given experiment ID
func (h *Handler) getVariantsByExperimentID(experimentID int) ([]models.Variant, error) {
	rows, err := h.db.Query(`
		SELECT id, experiment_id, name, description, traffic_percentage, created_at, updated_at 
		FROM variants 
		WHERE experiment_id = ?
		ORDER BY id`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var variants []models.Variant
	for rows.Next() {
		var v models.Variant
		err := rows.Scan(&v.ID, &v.ExperimentID, &v.Name, &v.Description, &v.TrafficPercentage, &v.CreatedAt, &v.UpdatedAt)
		if err != nil {
			return nil, err
		}
		variants = append(variants, v)
	}

	if variants == nil {
		variants = []models.Variant{}
	}

	return variants, nil
}

// getRulesByExperimentID retrieves all rules for a given experiment ID
func (h *Handler) getRulesByExperimentID(experimentID int) ([]models.Rule, error) {
	rows, err := h.db.Query(`
		SELECT id, experiment_id, priority, condition, action, created_at, updated_at 
		FROM experiment_rules 
		WHERE experiment_id = ?
		ORDER BY priority ASC`, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []models.Rule
	for rows.Next() {
		var r models.Rule
		err := rows.Scan(&r.ID, &r.ExperimentID, &r.Priority, &r.Condition, &r.Action, &r.CreatedAt, &r.UpdatedAt)
		if err != nil {
			return nil, err
		}
		rules = append(rules, r)
	}

	if rules == nil {
		rules = []models.Rule{}
	}

	return rules, nil
}

// EvaluateExperiment handles POST /experiments/{id}/evaluate - evaluate an experiment for an entity
func (h *Handler) EvaluateExperiment(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]

	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.logRequest(ctx, "error", "Invalid experiment ID", zap.String("id", idStr))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Invalid experiment ID"))
		return
	}

	var req models.EvaluateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logRequest(ctx, "error", "Invalid request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Invalid JSON"))
		return
	}

	if req.EntityType == "" || req.EntityID == "" {
		h.logRequest(ctx, "error", "Missing required fields: entity_type or entity_id")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("entity_type and entity_id are required"))
		return
	}

	h.logRequest(ctx, "info", "Evaluating experiment", zap.Int("experiment_id", id), zap.String("entity_type", req.EntityType), zap.String("entity_id", req.EntityID))

	// 1. Check if evaluation already exists for this entity and experiment
	var existingVariant string
	err = h.db.QueryRow(`
		SELECT variant_name 
		FROM evaluations 
		WHERE experiment_id = ? AND entity_type = ? AND entity_id = ?`,
		id, req.EntityType, req.EntityID).Scan(&existingVariant)

	if err == nil {
		h.logRequest(ctx, "info", "Returning existing evaluation", zap.Int("experiment_id", id), zap.String("variant", existingVariant))
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(models.EvaluateResponse{
			ExperimentID: id,
			VariantName:  existingVariant,
			EntityType:   req.EntityType,
			EntityID:     req.EntityID,
		})
		return
	} else if err != sql.ErrNoRows {
		h.logRequest(ctx, "error", "Failed to query existing evaluation", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Database error"))
		return
	}

	// 2. Fetch experiment and variants
	var exp models.Experiment
	err = h.db.QueryRow(`
		SELECT id, name, experiment_type, start_date, end_date 
		FROM experiments 
		WHERE id = ?`, id).Scan(&exp.ID, &exp.Name, &exp.ExperimentType, &exp.StartDate, &exp.EndDate)

	if err == sql.ErrNoRows {
		h.logRequest(ctx, "info", "Experiment not found", zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errs.NewNotFoundError("Experiment not found"))
		return
	} else if err != nil {
		h.logRequest(ctx, "error", "Failed to query experiment", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Database error"))
		return
	}

	// 3. Check if experiment is active
	now := time.Now()
	if now.Before(exp.StartDate) || now.After(exp.EndDate) {
		h.logRequest(ctx, "info", "Experiment is not active", zap.Int("experiment_id", id), zap.Time("now", now))
		// If not active, we might return a default or error. Let's return error for now.
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Experiment is not active"))
		return
	}

	variants, err := h.getVariantsByExperimentID(id)
	if err != nil || len(variants) == 0 {
		h.logRequest(ctx, "error", "Failed to load variants or no variants found", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to load variants"))
		return
	}

	// 4. Hash-based bucketing for assignment
	// We combine experiment ID and entity ID to ensure deterministic but unique assignment per experiment
	hasher := sha256.New()
	hasher.Write([]byte(fmt.Sprintf("%d:%s:%s", id, req.EntityType, req.EntityID)))
	hashBytes := hasher.Sum(nil)
	
	// Use the first 8 bytes of the hash to get a uint64
	hashValue := binary.BigEndian.Uint64(hashBytes[:8])
	
	// Bucketing (0-9999 for 0.01% precision)
	bucket := hashValue % 10000

	// Normalize to 0-100 percentage space (0.01% increments)
	percentageBucket := float64(bucket) / 100
	
	var assignedVariant string
	var cumulativePercentage float64
	for _, v := range variants {
		cumulativePercentage += v.TrafficPercentage
		if percentageBucket < cumulativePercentage {
			assignedVariant = v.Name
			break
		}
	}

	// Fallback to last variant if rounding issues occur (shouldn't happen with 1.0 sum validation)
	if assignedVariant == "" {
		assignedVariant = variants[len(variants)-1].Name
	}

	// 5. Return the result (no persistence for hash-based evaluation)
	h.logRequest(ctx, "info", "Experiment evaluated successfully", zap.Int("experiment_id", id), zap.String("variant", assignedVariant))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models.EvaluateResponse{
		ExperimentID: id,
		VariantName:  assignedVariant,
		EntityType:   req.EntityType,
		EntityID:     req.EntityID,
	})
}

// CreateManualEvaluation handles POST /evaluations - manually whitelist an entity for a variant
func (h *Handler) CreateManualEvaluation(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	var req struct {
		ExperimentID int    `json:"experiment_id"`
		EntityType   string `json:"entity_type"`
		EntityID     string `json:"entity_id"`
		VariantName  string `json:"variant_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logRequest(ctx, "error", "Invalid request body", zap.Error(err))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Invalid JSON"))
		return
	}

	if req.ExperimentID == 0 || req.EntityType == "" || req.EntityID == "" || req.VariantName == "" {
		h.logRequest(ctx, "error", "Missing required fields")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("experiment_id, entity_type, entity_id, and variant_name are required"))
		return
	}

	h.logRequest(ctx, "info", "Creating manual evaluation", zap.Int("experiment_id", req.ExperimentID), zap.String("entity_id", req.EntityID), zap.String("variant", req.VariantName))

	// Verify variant exists for the experiment
	var exists bool
	err := h.db.QueryRow(`
		SELECT EXISTS(SELECT 1 FROM variants WHERE experiment_id = ? AND name = ?)`,
		req.ExperimentID, req.VariantName).Scan(&exists)
	if err != nil || !exists {
		h.logRequest(ctx, "error", "Variant does not exist for experiment", zap.Int("experiment_id", req.ExperimentID), zap.String("variant", req.VariantName))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Variant does not exist for this experiment"))
		return
	}

	// Insert or update manual evaluation
	_, err = h.db.Exec(`
		INSERT INTO evaluations (experiment_id, entity_type, entity_id, variant_name, created_at) 
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(experiment_id, entity_type, entity_id) DO UPDATE SET 
			variant_name = excluded.variant_name,
			created_at = excluded.created_at`,
		req.ExperimentID, req.EntityType, req.EntityID, req.VariantName, time.Now())

	if err != nil {
		h.logRequest(ctx, "error", "Failed to create manual evaluation", zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create manual evaluation"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Manual evaluation created/updated successfully"})
}
