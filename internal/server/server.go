package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/yumazak/ccdc/internal/notify"
)

type notifyRequest struct {
	HookEventName        string `json:"hook_event_name"`
	Message              string `json:"message"`
	Title                string `json:"title"`
	Cwd                  string `json:"cwd"`
	LastAssistantMessage string `json:"last_assistant_message"`
}

// ListenAndServe starts the notification HTTP server on the given port.
func ListenAndServe(port int) error {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /notify", handleNotify)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("Listening on %s", addr)
	return http.ListenAndServe(addr, mux)
}

func handleNotify(w http.ResponseWriter, r *http.Request) {
	var req notifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	title := "Claude Code"
	if req.Cwd != "" {
		title = fmt.Sprintf("Claude Code - %s", filepath.Base(req.Cwd))
	}
	message := req.HookEventName
	if req.Message != "" {
		message = req.Message
	}

	if err := notify.Send(title, message); err != nil {
		log.Printf("notification failed: %v", err)
		http.Error(w, "notification failed", http.StatusInternalServerError)
		return
	}

	log.Printf("notified: [%s] %s", title, message)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "ok")
}
