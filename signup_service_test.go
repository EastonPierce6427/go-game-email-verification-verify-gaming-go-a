package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSignupSendsVerificationAndKeepsPendingState(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/email/send" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" || r.Header.Get("Idempotency-Key") == "" {
			t.Fatal("auth or request key missing")
		}
		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		if body["to"] != "player@example.com" || body["subject"] == "" || body["html"] == "" {
			t.Fatal("email payload incomplete")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"ok":true,"data":{"message_id":"msg-7"},"error":null,"metadata":{}}`))
	}))
	defer api.Close()
	svc := SignupService{Sender: &EmailSender{HTTPClient: api.Client(), APIKey: "test-key", BaseURL: api.URL, Sleep: func(_ time.Duration) {}}}
	req := httptest.NewRequest(http.MethodPost, "/signup", strings.NewReader(`{"player_id":"p-1","email":"player@example.com"}`))
	rec := httptest.NewRecorder()
	svc.Signup(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var got signupResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.MessageID != "msg-7" || got.VerificationState != "pending" {
		t.Fatalf("unexpected response: %+v", got)
	}
}
