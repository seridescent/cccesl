package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type StatusInput struct {
	TranscriptPath string `json:"transcript_path"`
}

// SessionMessage is a single entry in a Claude Code JSONL session transcript.
type SessionMessage struct {
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   Message   `json:"message"`
}

type Message struct {
	Usage Usage `json:"usage"`
}

type Usage struct {
	CacheCreation CacheCreation `json:"cache_creation"`
}

type CacheCreation struct {
	OneHour    int `json:"ephemeral_1h_input_tokens"`
	FiveMinute int `json:"ephemeral_5m_input_tokens"`
}

// TTL returns the cache lifetime indicated by whichever ephemeral token count
// is non-zero. Prefers the 1h bucket if both are set. Returns 0 when neither
// is set (no cache created by this message).
func (c CacheCreation) TTL() time.Duration {
	if c.OneHour > 0 {
		return time.Hour
	}
	if c.FiveMinute > 0 {
		return 5 * time.Minute
	}
	return 0
}

// parseSessionMessage parses one line of a Claude Code JSONL session transcript.
func parseSessionMessage(line []byte) (SessionMessage, error) {
	var msg SessionMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		return SessionMessage{}, err
	}
	return msg, nil
}

func isAssistantLine(line []byte) bool {
	msg, err := parseSessionMessage(line)
	return err == nil && msg.Type == "assistant" && !msg.Timestamp.IsZero()
}

func main() {
	inputBytes, _ := io.ReadAll(os.Stdin)
	var input StatusInput
	json.Unmarshal(inputBytes, &input)

	fmt.Println(CacheStatus(input.TranscriptPath))
}

// CacheStatus returns a human-readable string describing the prompt cache state.
func CacheStatus(transcriptPath string) string {
	if transcriptPath == "" {
		return "no cache"
	}

	line, err := lastAssistantLine(transcriptPath)
	if err != nil {
		return "transcript error"
	}

	msg, err := parseSessionMessage(line)
	if err != nil {
		return "transcript error"
	}

	ttl := msg.Message.Usage.CacheCreation.TTL()
	if ttl == 0 {
		return "no cache"
	}

	expiry := msg.Timestamp.Add(ttl)
	if time.Now().After(expiry) {
		return "cache expired"
	}
	return fmt.Sprintf("cache expires at %s", expiry.Local().Format("15:04:05"))
}

// lastAssistantLine reads a Claude Code transcript (JSONL) from the end
// and returns the raw bytes of the most recent assistant message line.
func lastAssistantLine(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := stat.Size()
	if size == 0 {
		return nil, fmt.Errorf("empty file")
	}

	// Read from end in chunks, looking for assistant messages
	const chunkSize = 64 * 1024
	buf := make([]byte, chunkSize)
	var trailing []byte

	for offset := size; offset > 0; {
		readSize := min(int64(chunkSize), offset)
		offset -= readSize

		n, err := f.ReadAt(buf[:readSize], offset)
		if err != nil && err != io.EOF {
			return nil, err
		}

		// Prepend to any trailing partial line from previous chunk
		chunk := append(buf[:n], trailing...)
		trailing = nil

		// Process lines in reverse (split by newlines)
		for len(chunk) > 0 {
			lastNL := -1
			for i := len(chunk) - 1; i >= 0; i-- {
				if chunk[i] == '\n' {
					lastNL = i
					break
				}
			}

			var line []byte
			if lastNL == -1 {
				// No newline found - this is a partial line, save for next chunk
				trailing = chunk
				break
			} else if lastNL == len(chunk)-1 {
				// Trailing newline, skip it
				chunk = chunk[:lastNL]
				continue
			} else {
				line = chunk[lastNL+1:]
				chunk = chunk[:lastNL]
			}

			if isAssistantLine(line) {
				return append([]byte(nil), line...), nil
			}
		}
	}

	// Handle any remaining partial line at the start of file
	if len(trailing) > 0 && isAssistantLine(trailing) {
		return append([]byte(nil), trailing...), nil
	}

	return nil, fmt.Errorf("no assistant messages")
}
