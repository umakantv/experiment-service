package server

import (
	"net/http"
	cachepackage "oauth-service/cache"
	"oauth-service/database"
	"oauth-service/handlers"
	"os"

	"github.com/umakantv/go-utils/httpserver"
	"github.com/umakantv/go-utils/logger"
	"go.uber.org/zap"
)

// checkAuth implements authentication for the service
func checkAuth(r *http.Request) (bool, httpserver.RequestAuth) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return false, httpserver.RequestAuth{}
	}

	// Simple Bearer token check (in production, validate JWT)
	if len(auth) > 7 && auth[:7] == "Bearer " {
		token := auth[7:]
		if token == "secret-token" { // Simple check for demo
			return true, httpserver.RequestAuth{
				Type:   "bearer",
				Client: "experiment-service-client",
				Claims: map[string]interface{}{"service": "experiment-service"},
			}
		}
	}

	return false, httpserver.RequestAuth{}
}

func StartServer() {
	// Initialize logger
	logger.Init(logger.LoggerConfig{
		CallerKey:  "file",
		TimeKey:    "timestamp",
		CallerSkip: 1,
	})

	logger.Info("Starting Experiment Service...")

	// Initialize database
	dbConn := database.InitializeDatabase()
	defer dbConn.Close()

	// Initialize cache
	cache := cachepackage.InitializeCache()
	defer cache.Close()

	// Initialize handlers
	handler := handlers.NewHandler(dbConn, cache)

	// Create HTTP server with authentication
	server := httpserver.New("8080", checkAuth)

	// Register routes
	server.Register(httpserver.Route{
		Name:     "HealthCheck",
		Method:   "GET",
		Path:     "/health",
		AuthType: "none",
	}, httpserver.HandlerFunc(handler.HealthCheck))

	// Experiment routes
	server.Register(httpserver.Route{
		Name:     "ListExperiments",
		Method:   "GET",
		Path:     "/experiments",
		AuthType: "bearer",
	}, httpserver.HandlerFunc(handler.ListExperiments))

	server.Register(httpserver.Route{
		Name:     "GetExperiment",
		Method:   "GET",
		Path:     "/experiments/{id}",
		AuthType: "bearer",
	}, httpserver.HandlerFunc(handler.GetExperiment))

	server.Register(httpserver.Route{
		Name:     "CreateExperiment",
		Method:   "POST",
		Path:     "/experiments",
		AuthType: "bearer",
	}, httpserver.HandlerFunc(handler.CreateExperiment))

	server.Register(httpserver.Route{
		Name:     "UpdateExperiment",
		Method:   "PUT",
		Path:     "/experiments/{id}",
		AuthType: "bearer",
	}, httpserver.HandlerFunc(handler.UpdateExperiment))

	server.Register(httpserver.Route{
		Name:     "DeleteExperiment",
		Method:   "DELETE",
		Path:     "/experiments/{id}",
		AuthType: "bearer",
	}, httpserver.HandlerFunc(handler.DeleteExperiment))

	server.Register(httpserver.Route{
		Name:     "EvaluateExperiment",
		Method:   "POST",
		Path:     "/experiments/{id}/evaluate",
		AuthType: "bearer",
	}, httpserver.HandlerFunc(handler.EvaluateExperiment))

	server.Register(httpserver.Route{
		Name:     "CreateManualEvaluation",
		Method:   "POST",
		Path:     "/evaluations",
		AuthType: "bearer",
	}, httpserver.HandlerFunc(handler.CreateManualEvaluation))

	logger.Info("Experiment Service started on port 8080")
	logger.Info("Health check: GET /health")
	logger.Info("API endpoints: GET/POST/PUT/DELETE /experiments")

	// Start server
	if err := server.Start(); err != nil {
		logger.Error("Server failed to start", zap.Error(err))
		os.Exit(1)
	}
}
