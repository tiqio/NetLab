package observability

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"sync/atomic"
)

type Metrics struct {
	ReconcileRuns    atomic.Uint64
	ReconcileErrors  atomic.Uint64
	TasksRunning     atomic.Int64
	DriftedResources atomic.Int64
}

func NewLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
func MetricsHandler(metrics *Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		_, _ = w.Write([]byte("netlab_reconcile_runs_total " + itoa(metrics.ReconcileRuns.Load()) + "\nnetlab_reconcile_errors_total " + itoa(metrics.ReconcileErrors.Load()) + "\n"))
	}
}
func itoa(value uint64) string {
	if value == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for value > 0 {
		i--
		b[i] = byte('0' + value%10)
		value /= 10
	}
	return string(b[i:])
}
