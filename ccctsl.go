package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type StatusInput struct {
	Model struct {
		DisplayName string `json:"display_name"`
	} `json:"model"`
	ContextWindow struct {
		UsedPercentage float64 `json:"used_percentage"`
	} `json:"context_window"`
}

func main() {
	inputBytes, _ := io.ReadAll(os.Stdin)
	var input StatusInput
	json.Unmarshal(inputBytes, &input)

	// Cache TTL is 5 minutes from now (each message refreshes it)
	expiry := time.Now().Add(5 * time.Minute)

	fmt.Printf("[%s] ctx:%.0f%% cache expires %s\n",
		input.Model.DisplayName,
		input.ContextWindow.UsedPercentage,
		expiry.Format("15:04:05"))
}