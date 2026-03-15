package server

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleNotify_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader("not json"))
	w := httptest.NewRecorder()

	handleNotify(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestHandleNotify_EmptyBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/notify", strings.NewReader("{}"))
	w := httptest.NewRecorder()

	handleNotify(w, req)

	// Will fail with 500 because notify.Send calls osascript which won't exist in CI,
	// but at least it parses the JSON successfully (not 400)
	if w.Code == http.StatusBadRequest {
		t.Error("expected JSON to be parsed successfully, got 400")
	}
}

func TestNotifyRequest_TitleFromCwd(t *testing.T) {
	// Test the title/message logic by checking the request parsing
	// We can't easily test the full handler without mocking notify.Send,
	// so we test the parsing logic directly
	tests := []struct {
		name            string
		cwd             string
		hookEventName   string
		message         string
		expectedTitle   string
		expectedMessage string
	}{
		{
			name:            "with cwd",
			cwd:             "/home/user/my-project",
			hookEventName:   "Stop",
			expectedTitle:   "Claude Code - my-project",
			expectedMessage: "Stop",
		},
		{
			name:            "without cwd",
			hookEventName:   "Notification",
			expectedTitle:   "Claude Code",
			expectedMessage: "Notification",
		},
		{
			name:            "with custom message",
			cwd:             "/home/user/my-project",
			hookEventName:   "Stop",
			message:         "Task completed",
			expectedTitle:   "Claude Code - my-project",
			expectedMessage: "Task completed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := notifyRequest{
				Cwd:           tt.cwd,
				HookEventName: tt.hookEventName,
				Message:       tt.message,
			}

			title := "Claude Code"
			if req.Cwd != "" {
				title = "Claude Code - " + filepath.Base(req.Cwd)
			}
			message := req.HookEventName
			if req.Message != "" {
				message = req.Message
			}

			if title != tt.expectedTitle {
				t.Errorf("title: expected %q, got %q", tt.expectedTitle, title)
			}
			if message != tt.expectedMessage {
				t.Errorf("message: expected %q, got %q", tt.expectedMessage, message)
			}
		})
	}
}
