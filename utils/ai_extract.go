package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

type deepseekMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type deepseekRequest struct {
	Model    string            `json:"model"`
	Messages []deepseekMessage `json:"messages"`
	Stream   bool              `json:"stream"`
}

type deepseekResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func ExtractWithDeepseek(prompt string) (map[string]any, error) {
	apiURL := viper.GetString("deepseek.api_url")
	if apiURL == "" {
		apiURL = "https://api.deepseek.com/chat/completions"
	}

	apiKey := viper.GetString("deepseek.api_key")
	if apiKey == "" {
		return nil, fmt.Errorf("deepseek api key is not configured")
	}

	model := viper.GetString("deepseek.model")
	if model == "" {
		model = "deepseek-v4-pro"
	}

	messages := []deepseekMessage{
		{
			Role:    "system",
			Content: "你是一个信息抽取专家，请把用户提供的文章内容提取为纯JSON格式，只输出JSON对象，不输出任何其他说明文本。",
		},
		{
			Role:    "user",
			Content: prompt,
		},
	}

	reqBody, err := json.Marshal(deepseekRequest{
		Model:    model,
		Messages: messages,
		Stream:   false,
	})
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, apiURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiKey))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("deepseek api error: status=%d body=%s", resp.StatusCode, string(body))
	}

	var respWrap deepseekResponse
	if err := json.Unmarshal(body, &respWrap); err != nil {
		return nil, fmt.Errorf("failed to parse deepseek response: %w", err)
	}

	if respWrap.Error.Message != "" {
		return nil, fmt.Errorf("deepseek api error: %s", respWrap.Error.Message)
	}

	if len(respWrap.Choices) == 0 {
		return nil, fmt.Errorf("no choices in deepseek response")
	}

	content := respWrap.Choices[0].Message.Content
	if content == "" {
		return nil, fmt.Errorf("empty content from deepseek")
	}

	return parseJSONFromString(content)
}

func parseJSONFromString(raw string) (map[string]any, error) {
	var parsed any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return nil, err
	}
	if parsedMap, ok := parsed.(map[string]any); ok {
		return parsedMap, nil
	}
	return nil, fmt.Errorf("parsed response is not a JSON object")
}
