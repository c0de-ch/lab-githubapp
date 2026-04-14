package webhook

import (
	"io"
	"log/slog"
	"net/http"
)

type Handler struct {
	secret    []byte
	processor *EventProcessor
	logger    *slog.Logger
}

func NewHandler(secret string, processor *EventProcessor, logger *slog.Logger) *Handler {
	return &Handler{
		secret:    []byte(secret),
		processor: processor,
		logger:    logger,
	}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("failed to read webhook body", "error", err)
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	signature := r.Header.Get("X-Hub-Signature-256")
	if err := VerifySignature(body, signature, h.secret); err != nil {
		h.logger.Warn("webhook signature verification failed", "error", err)
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	eventType := r.Header.Get("X-GitHub-Event")
	deliveryID := r.Header.Get("X-GitHub-Delivery")

	h.logger.Info("webhook received",
		"event", eventType,
		"delivery_id", deliveryID,
	)

	if err := h.processor.Process(eventType, body); err != nil {
		h.logger.Error("failed to process webhook",
			"event", eventType,
			"delivery_id", deliveryID,
			"error", err,
		)
		http.Error(w, "processing failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"ok":true}`))
}
