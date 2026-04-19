package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

// Replace with a real line from a Claude Code JSONL session transcript.
var exampleAssistantLine = compactJSON(`
{
  "parentUuid": "FAKE-PARENT-UUID",
  "isSidechain": false,
  "message": {
    "model": "claude-opus-4-7",
    "id": "msg_FAKE-ID",
    "type": "message",
    "role": "assistant",
    "content": [
      {"type": "text", "text": "fake assistant response text"}
    ],
    "stop_reason": "tool_use",
    "stop_sequence": null,
    "stop_details": null,
    "usage": {
      "input_tokens": 6,
      "cache_creation_input_tokens": 2699,
      "cache_read_input_tokens": 32596,
      "output_tokens": 410,
      "server_tool_use": {"web_search_requests": 0, "web_fetch_requests": 0},
      "service_tier": "standard",
      "cache_creation": {
        "ephemeral_1h_input_tokens": 2699,
        "ephemeral_5m_input_tokens": 0
      },
      "inference_geo": "",
      "iterations": [
        {
          "input_tokens": 6,
          "output_tokens": 410,
          "cache_read_input_tokens": 32596,
          "cache_creation_input_tokens": 2699,
          "cache_creation": {
            "ephemeral_5m_input_tokens": 0,
            "ephemeral_1h_input_tokens": 2699
          },
          "type": "message"
        }
      ],
      "speed": "standard"
    }
  },
  "requestId": "req_FAKE-ID",
  "type": "assistant",
  "uuid": "FAKE-UUID",
  "timestamp": "2026-04-19T17:25:15.314Z",
  "userType": "external",
  "entrypoint": "cli",
  "cwd": "/path/to/project",
  "sessionId": "FAKE-SESSION-ID",
  "version": "2.1.114",
  "gitBranch": "branch-name"
}
`)

func compactJSON(s string) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, []byte(s)); err != nil {
		panic(err)
	}
	return buf.String()
}

func TestParseSessionMessage(t *testing.T) {
	msg, err := parseSessionMessage([]byte(exampleAssistantLine))
	if err != nil {
		t.Fatalf("parseSessionMessage: %v", err)
	}
	if msg.Type != "assistant" {
		t.Errorf("Type = %q, want %q", msg.Type, "assistant")
	}
	if msg.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
	want, _ := time.Parse(time.RFC3339Nano, "2026-04-19T17:25:15.314Z")
	if !msg.Timestamp.Equal(want) {
		t.Errorf("Timestamp = %v, want %v", msg.Timestamp, want)
	}

	cache := msg.Message.Usage.CacheCreation
	if cache.OneHour != 2699 {
		t.Errorf("CacheCreation.OneHour = %d, want 2699", cache.OneHour)
	}
	if cache.FiveMinute != 0 {
		t.Errorf("CacheCreation.FiveMinute = %d, want 0", cache.FiveMinute)
	}
	if got := cache.TTL(); got != time.Hour {
		t.Errorf("CacheCreation.TTL() = %v, want %v", got, time.Hour)
	}
}

func TestCacheCreationTTL(t *testing.T) {
	tests := []struct {
		name  string
		cache CacheCreation
		want  time.Duration
	}{
		{"one hour only", CacheCreation{OneHour: 100}, time.Hour},
		{"five minute only", CacheCreation{FiveMinute: 100}, 5 * time.Minute},
		{"both zero", CacheCreation{}, 0},
		{"both non-zero prefers one hour", CacheCreation{OneHour: 100, FiveMinute: 100}, time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.cache.TTL(); got != tt.want {
				t.Errorf("TTL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseSessionMessageFiveMinuteCache(t *testing.T) {
	line := `{"type":"assistant","timestamp":"2026-04-19T17:25:15.314Z","message":{"usage":{"cache_creation":{"ephemeral_1h_input_tokens":0,"ephemeral_5m_input_tokens":1234}}}}`
	msg, err := parseSessionMessage([]byte(line))
	if err != nil {
		t.Fatalf("parseSessionMessage: %v", err)
	}
	if got := msg.Message.Usage.CacheCreation.TTL(); got != 5*time.Minute {
		t.Errorf("TTL() = %v, want %v", got, 5*time.Minute)
	}
}
