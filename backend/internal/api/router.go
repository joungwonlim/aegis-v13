package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"

	"github.com/wonny/aegis/v13/backend/internal/api/handlers"
	"github.com/wonny/aegis/v13/backend/pkg/logger"
)

// NewRouter creates and configures the HTTP router
// ⭐ SSOT: 라우팅 설정은 이 함수에서만
func NewRouter(dataHandler *handlers.DataHandler, tradingHandler *handlers.TradingHandler, log *logger.Logger) http.Handler {
	r := mux.NewRouter()

	// Health check
	r.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// API v1
	api := r.PathPrefix("/api").Subrouter()

	// Data endpoints
	api.HandleFunc("/data/quality", dataHandler.GetQuality).Methods("GET")
	api.HandleFunc("/data/universe", dataHandler.GetUniverse).Methods("GET")
	api.HandleFunc("/data/collect", dataHandler.Collect).Methods("POST")

	// Trading endpoints (KIS API)
	api.HandleFunc("/trading/balance", tradingHandler.GetBalance).Methods("GET")
	api.HandleFunc("/trading/positions", tradingHandler.GetPositions).Methods("GET")
	api.HandleFunc("/trading/orders", tradingHandler.GetOrders).Methods("GET")
	api.HandleFunc("/trading/orders/unfilled", tradingHandler.GetUnfilledOrders).Methods("GET")
	api.HandleFunc("/trading/orders/filled", tradingHandler.GetFilledOrders).Methods("GET")
	api.HandleFunc("/trading/orders", tradingHandler.PlaceOrder).Methods("POST")
	api.HandleFunc("/trading/orders", tradingHandler.CancelOrder).Methods("DELETE")
	api.HandleFunc("/trading/price", tradingHandler.GetCurrentPrice).Methods("GET")

	// WebSocket management endpoints
	api.HandleFunc("/trading/ws/status", tradingHandler.GetWebSocketStatus).Methods("GET")
	api.HandleFunc("/trading/ws/subscribe", tradingHandler.Subscribe).Methods("POST")
	api.HandleFunc("/trading/ws/unsubscribe", tradingHandler.Unsubscribe).Methods("POST")

	// Apply middleware
	r.Use(corsMiddleware())
	r.Use(loggingMiddleware(log))
	r.Use(recoveryMiddleware(log))

	return r
}

// healthCheckHandler returns server health status
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"service": "aegis-v13-api",
	})
}

// loggingMiddleware logs HTTP requests
func loggingMiddleware(log *logger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Call next handler
			next.ServeHTTP(w, r)

			// Log request
			log.WithFields(map[string]interface{}{
				"method":   r.Method,
				"path":     r.URL.Path,
				"duration": time.Since(start),
			}).Debug("HTTP request")
		})
	}
}

// recoveryMiddleware recovers from panics
func recoveryMiddleware(log *logger.Logger) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					log.WithFields(map[string]interface{}{
						"error": err,
						"path":  r.URL.Path,
					}).Error("Panic recovered")

					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusInternalServerError)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Internal server error",
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware handles CORS preflight requests
func corsMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Set CORS headers
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "86400")

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
