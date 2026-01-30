package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

type StatusInput struct {
	TranscriptPath string `json:"transcript_path"`
}

func main() {
	// claude code transcripts imply the cache TTL is 1 hour.
	// the typical line.message.usage.cache_creation struct has
	// zero `ephemeral_5m_input_tokens` and non-zero `ephemeral_1h_input_tokens`.
	ttl := flag.Duration("ttl", 1*time.Hour, "cache TTL duration")
	flag.Parse()

	inputBytes, _ := io.ReadAll(os.Stdin)
	var input StatusInput
	json.Unmarshal(inputBytes, &input)

	fmt.Println(CacheStatus(input.TranscriptPath, *ttl))
}

// CacheStatus returns a human-readable string describing the prompt cache state.
func CacheStatus(transcriptPath string, ttl time.Duration) string {
	if transcriptPath == "" {
		return "no cache"
	}

	lastAssistantTime, err := lastAssistantTimestamp(transcriptPath)
	if err != nil {
		return "no cache"
	}

	expiry := lastAssistantTime.Add(ttl)
	if time.Now().After(expiry) {
		return "cache expired"
	}

	return fmt.Sprintf("cache expires at %s", expiry.Local().Format("15:04:05"))
}

// lastAssistantTimestamp reads a Claude Code transcript (JSONL) from the end
// and returns the timestamp of the most recent assistant message.
func lastAssistantTimestamp(path string) (time.Time, error) {
	f, err := os.Open(path)
	if err != nil {
		return time.Time{}, err
	}
	defer f.Close()

	// Get file size
	stat, err := f.Stat()
	if err != nil {
		return time.Time{}, err
	}
	size := stat.Size()
	if size == 0 {
		return time.Time{}, fmt.Errorf("empty file")
	}

	// Read from end in chunks, looking for assistant messages
	const chunkSize = 64 * 1024
	buf := make([]byte, chunkSize)
	var trailing []byte

	for offset := size; offset > 0; {
		// Calculate read position
		readSize := min(int64(chunkSize), offset)
		offset -= readSize

		// Read chunk
		n, err := f.ReadAt(buf[:readSize], offset)
		if err != nil && err != io.EOF {
			return time.Time{}, err
		}

		// Prepend to any trailing partial line from previous chunk
		chunk := append(buf[:n], trailing...)
		trailing = nil

		// Process lines in reverse (split by newlines)
		for len(chunk) > 0 {
			// Find the last newline
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
				// Extract the last complete line
				line = chunk[lastNL+1:]
				chunk = chunk[:lastNL]
			}

			// Try to parse as assistant message
			var entry struct {
				Type      string `json:"type"`
				Timestamp string `json:"timestamp"`
			}
			if json.Unmarshal(line, &entry) == nil && entry.Type == "assistant" && entry.Timestamp != "" {
				if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
					return t, nil
				}
			}
		}
	}

	// Handle any remaining partial line at the start of file
	if len(trailing) > 0 {
		var entry struct {
			Type      string `json:"type"`
			Timestamp string `json:"timestamp"`
		}
		if json.Unmarshal(trailing, &entry) == nil && entry.Type == "assistant" && entry.Timestamp != "" {
			if t, err := time.Parse(time.RFC3339Nano, entry.Timestamp); err == nil {
				return t, nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("no assistant messages")
}
