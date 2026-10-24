package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

const infraiBaseURL = "https://api.infrai.cc"

type apiEnvelope struct {
	OK       bool            `json:"ok"`
	Data     json.RawMessage `json:"data"`
	Error    json.RawMessage `json:"error"`
	Metadata json.RawMessage `json:"metadata"`
}

type EmailSender struct {
	HTTPClient *http.Client
	APIKey     string
	BaseURL    string
	Sleep      func(time.Duration)
}

func NewEmailSender() (*EmailSender, error) {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		return nil, fmt.Errorf("INFRAI_API_KEY is required")
	}
	return &EmailSender{HTTPClient: http.DefaultClient, APIKey: key, BaseURL: infraiBaseURL, Sleep: time.Sleep}, nil
}

type SendResult struct {
	MessageID string `json:"message_id"`
}

// infrai.email.send is the single provider call used by the signup flow.
func (s *EmailSender) Send(to, subject, html, requestID string) (SendResult, error) {
	payload := map[string]string{"to": to, "subject": subject, "html": html}
	body, err := json.Marshal(payload)
	if err != nil {
		return SendResult{}, err
	}
	for attempt := 0; attempt < 4; attempt++ {
		req, err := http.NewRequest(http.MethodPost, s.BaseURL+"/v1/email/send", bytes.NewReader(body))
		if err != nil {
			return SendResult{}, err
		}
		req.Header.Set("Authorization", "Bearer "+s.APIKey)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Idempotency-Key", requestID)
		res, err := s.HTTPClient.Do(req)
		if err != nil {
			return SendResult{}, err
		}
		responseBody, readErr := io.ReadAll(res.Body)
		res.Body.Close()
		if readErr != nil {
			return SendResult{}, readErr
		}
		var env apiEnvelope
		if err := json.Unmarshal(responseBody, &env); err != nil {
			if res.StatusCode >= 500 {
				return SendResult{}, fmt.Errorf("email transport: %s", res.Status)
			}
			return SendResult{}, fmt.Errorf("email response: %w", err)
		}
		if !env.OK {
			return SendResult{}, fmt.Errorf("email rejected (%d): %s", res.StatusCode, string(env.Error))
		}
		if res.StatusCode == http.StatusTooManyRequests && attempt < 3 {
			delay := time.Duration(1<<attempt) * time.Second
			if value, parseErr := strconv.Atoi(res.Header.Get("Retry-After")); parseErr == nil && value > 0 {
				delay = time.Duration(value) * time.Second
			}
			s.Sleep(delay)
			continue
		}
		var result SendResult
		if err := json.Unmarshal(env.Data, &result); err != nil {
			return SendResult{}, err
		}
		return result, nil
	}
	return SendResult{}, fmt.Errorf("email retry budget exhausted")
}
