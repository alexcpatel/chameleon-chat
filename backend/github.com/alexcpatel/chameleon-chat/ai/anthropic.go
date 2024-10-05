package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
)

func callClaudeAPI(prompt string) (string, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("ANTHROPIC_API_KEY environment variable not set")
	}

	model := os.Getenv("CLAUDE_MODEL")
	if model == "" {
		return "", fmt.Errorf("CLAUDE_MODEL environment variable not set")
	}

	temperatureStr := os.Getenv("CLAUDE_TEMPERATURE")
	if temperatureStr == "" {
		return "", fmt.Errorf("CLAUDE_TEMPERATURE environment variable not set")
	}

	temperature, err := strconv.ParseFloat(temperatureStr, 64)
	if err != nil {
		return "", fmt.Errorf("invalid CLAUDE_TEMPERATURE value: %v", err)
	}

	url := "https://api.anthropic.com/v1/messages"

	data := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"max_tokens":  100,
		"temperature": temperature,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("error marshaling request data: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	var result map[string]interface{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", fmt.Errorf("error unmarshaling response: %v", err)
	}

	if errorInfo, ok := result["error"].(map[string]interface{}); ok {
		return "", fmt.Errorf("API error: %v", errorInfo["message"])
	}

	content, ok := result["content"].([]interface{})
	if !ok || len(content) == 0 {
		return "", fmt.Errorf("unexpected response format: %v", result)
	}

	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected content format: %v", content[0])
	}

	text, ok := firstContent["text"].(string)
	if !ok {
		return "", fmt.Errorf("unexpected text format: %v", firstContent["text"])
	}

	return text, nil
}
