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
		SELECT id, name, description, start_date, end_date, created_at, updated_at 
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
		err := rows.Scan(&exp.ID, &exp.Name, &exp.Description, &exp.StartDate, &exp.EndDate, &exp.CreatedAt, &exp.UpdatedAt)
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
		SELECT id, name, description, start_date, end_date, created_at, updated_at 
		FROM experiments 
		WHERE id = ?`, id).
		Scan(&exp.ID, &exp.Name, &exp.Description, &exp.StartDate, &exp.EndDate, &exp.CreatedAt, &exp.UpdatedAt)

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

	if len(req.Variants) < 2 {
		h.logRequest(ctx, "error", "Not enough variants")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("At least 2 variants are required (e.g., control and treatment)"))
		return
	}

	// Validate traffic ratio sums to 1.0
	var totalTraffic float64
	for _, v := range req.Variants {
		if v.Name == "" {
			h.logRequest(ctx, "error", "Variant missing name")
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(errs.NewValidationError("All variants must have a name"))
			return
		}
		totalTraffic += v.TrafficRatio
	}
	if totalTraffic < 0.99 || totalTraffic > 1.01 {
		h.logRequest(ctx, "error", "Invalid traffic ratio sum", zap.Float64("total", totalTraffic))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("Traffic ratios must sum to 1.0"))
		return
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
		INSERT INTO experiments (name, description, start_date, end_date, created_at, updated_at) 
		VALUES (?, ?, ?, ?, ?, ?)`,
		req.Name, req.Description, req.StartDate, req.EndDate, time.Now(), time.Now())
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
	for _, v := range req.Variants {
		_, err := tx.Exec(`
			INSERT INTO variants (experiment_id, name, description, traffic_ratio, created_at, updated_at) 
			VALUES (?, ?, ?, ?, ?, ?)`,
			experimentID, v.Name, v.Description, v.TrafficRatio, time.Now(), time.Now())
		if err != nil {
			h.logRequest(ctx, "error", "Failed to create variant", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to create variant"))
			return
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
	exp := models.Experiment{
		ID:          int(experimentID),
		Name:        req.Name,
		Description: req.Description,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
		Variants:    make([]models.Variant, len(req.Variants)),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	for i, v := range req.Variants {
		exp.Variants[i] = models.Variant{
			ExperimentID: int(experimentID),
			Name:         v.Name,
			Description:  v.Description,
			TrafficRatio: v.TrafficRatio,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

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
	if req.StartDate != nil {
		setParts = append(setParts, "start_date = ?")
		args = append(args, *req.StartDate)
	}
	if req.EndDate != nil {
		setParts = append(setParts, "end_date = ?")
		args = append(args, *req.EndDate)
	}

	if len(setParts) == 0 {
		h.logRequest(ctx, "error", "No fields to update", zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(errs.NewValidationError("No fields to update"))
		return
	}

	setParts = append(setParts, "updated_at = ?")
	args = append(args, time.Now())
	args = append(args, id)

	query := "UPDATE experiments SET " + strings.Join(setParts, ", ") + " WHERE id = ?"
	result, err := h.db.Exec(query, args...)
	if err != nil {
		h.logRequest(ctx, "error", "Failed to update experiment", zap.Error(err), zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(errs.NewInternalServerError("Failed to update experiment"))
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		h.logRequest(ctx, "info", "Experiment not found for update", zap.Int("experiment_id", id))
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(errs.NewNotFoundError("Experiment not found"))
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
		SELECT id, experiment_id, name, description, traffic_ratio, created_at, updated_at 
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
		err := rows.Scan(&v.ID, &v.ExperimentID, &v.Name, &v.Description, &v.TrafficRatio, &v.CreatedAt, &v.UpdatedAt)
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
		SELECT id, name, start_date, end_date 
		FROM experiments 
		WHERE id = ?`, id).Scan(&exp.ID, &exp.Name, &exp.StartDate, &exp.EndDate)

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
	
	var assignedVariant string
	var cumulativeRatio float64
	for _, v := range variants {
		cumulativeRatio += v.TrafficRatio
		if float64(bucket) < cumulativeRatio*10000 {
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
