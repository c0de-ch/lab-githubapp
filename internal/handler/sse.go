package handler

import (
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/c0de-ch/lab-githubapp/internal/sse"
)

type SSEHandler struct {
	broker *sse.Broker
	logger *slog.Logger
}

func NewSSEHandler(broker *sse.Broker, logger *slog.Logger) *SSEHandler {
	return &SSEHandler{broker: broker, logger: logger}
}

func (h *SSEHandler) RepoStream(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	topic := fmt.Sprintf("repo:%s/%s", owner, repo)
	h.stream(w, r, topic)
}

func (h *SSEHandler) RunStream(w http.ResponseWriter, r *http.Request) {
	owner := r.PathValue("owner")
	repo := r.PathValue("repo")
	runID := r.PathValue("runID")

	// Validate runID
	if _, err := strconv.ParseInt(runID, 10, 64); err != nil {
		http.Error(w, "Invalid run ID", http.StatusBadRequest)
		return
	}

	topic := fmt.Sprintf("run:%s/%s/%s", owner, repo, runID)
	h.stream(w, r, topic)
}

func (h *SSEHandler) stream(w http.ResponseWriter, r *http.Request, topic string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	rc := http.NewResponseController(w)

	ch, unsubscribe := h.broker.Subscribe(topic)
	defer unsubscribe()

	// Send initial connected event
	fmt.Fprintf(w, "event: connected\ndata: ok\n\n")
	if err := rc.Flush(); err != nil {
		return
	}

	h.logger.Debug("SSE client connected", "topic", topic)

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			h.logger.Debug("SSE client disconnected", "topic", topic)
			return
		case event := <-ch:
			w.Write(event.Format())
			if err := rc.Flush(); err != nil {
				return
			}
		case <-ticker.C:
			fmt.Fprintf(w, ": keepalive\n\n")
			if err := rc.Flush(); err != nil {
				return
			}
		}
	}
}
