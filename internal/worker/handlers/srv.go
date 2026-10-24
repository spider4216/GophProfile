package handlers

import (
	"log/slog"
	"net/http"

	"github.com/spider4216/GophProfile/internal/queue"
)

type SrvHandler struct {
	queue  *queue.Queue
	logger *slog.Logger
}

func NewSrvHandler(queue *queue.Queue, logger *slog.Logger) *SrvHandler {
	return &SrvHandler{
		queue:  queue,
		logger: logger,
	}
}

func (h *SrvHandler) Liveness(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("worker liveness")

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("ok")); err != nil {
		h.logger.Warn("cannot write response in worker", "error", err)
	}
}

func (h *SrvHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	h.logger.Debug("worker readiness")

	if !h.queue.IsConnected() {
		h.logger.Error("worker unavailable")
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("ok")); err != nil {
		h.logger.Warn("cannot write response in worker", "error", err)
	}
}
