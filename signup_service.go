package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type PlayerAsset struct {
	Kind string `json:"kind"`
	Name string `json:"name"`
}
type LiveEvent struct {
	Name  string `json:"name"`
	State string `json:"state"`
}
type ModerationItem struct {
	PlayerID string `json:"player_id"`
	Reason   string `json:"reason"`
}
type Player struct {
	ID       string           `json:"id"`
	Email    string           `json:"email"`
	Verified bool             `json:"verified"`
	Assets   []PlayerAsset    `json:"assets"`
	Events   []LiveEvent      `json:"events"`
	Queue    []ModerationItem `json:"moderation_queue"`
}
type signupRequest struct {
	PlayerID string `json:"player_id"`
	Email    string `json:"email"`
}
type signupResponse struct {
	PlayerID          string `json:"player_id"`
	MessageID         string `json:"message_id"`
	VerificationState string `json:"verification_state"`
}

type SignupService struct{ Sender *EmailSender }

func (s SignupService) Signup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}
	var input signupRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil || input.PlayerID == "" || !strings.Contains(input.Email, "@") {
		http.Error(w, "player_id and email are required", http.StatusBadRequest)
		return
	}
	token := make([]byte, 16)
	if _, err := rand.Read(token); err != nil {
		http.Error(w, "token generation failed", http.StatusInternalServerError)
		return
	}
	requestID := hex.EncodeToString(token)
	result, err := s.Sender.Send(input.Email, "Verify your game account", fmt.Sprintf("<p>Confirm player %s: /verify/%s</p>", input.PlayerID, requestID), requestID)
	if err != nil {
		http.Error(w, "verification email rejected", http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(signupResponse{PlayerID: input.PlayerID, MessageID: result.MessageID, VerificationState: "pending"})
}
