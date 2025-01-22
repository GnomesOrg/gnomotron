package gptadapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
)

type GptAdapter struct {
	client   *http.Client
	baseURL  string
	apiToken string
	l        *slog.Logger
}

type GptResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func New(apiToken string, l *slog.Logger) *GptAdapter {
	return &GptAdapter{
		client:   &http.Client{},
		baseURL:  "https://lk.neuroapi.host/v1/chat/completions",
		apiToken: apiToken,
		l:        l,
	}
}

func (g *GptAdapter) createSingleRequestBody(model string, systemMsg string, userMsg string) ([]byte, error) {
	requestData := map[string]interface{}{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": systemMsg},
			{"role": "user", "content": userMsg},
		},
	}

	return json.Marshal(requestData)
}

func (g *GptAdapter) AskGpt(systemMsg, userMsg string) (string, error) {
	body, err := g.createSingleRequestBody("gpt-4o", systemMsg, userMsg)
	if err != nil {
		return "", fmt.Errorf("cannot create request body: %w", err)
	}

	request, err := http.NewRequest("POST", g.baseURL, bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("cannot create new http request: %w", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.apiToken))

	response, err := g.client.Do(request)
	if err != nil {
		return "", fmt.Errorf("cannot get response from gpt server: %w", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("cannot read the response from gpt server: %w", err)
	}

	res := GptResponse{}
	if err := json.Unmarshal(responseBody, &res); err != nil {
		return "", fmt.Errorf("cannot unmarshal gpt response %q as json: %w", string(responseBody), err)
	}

	g.l.Debug("got gpt response", slog.Any("res", res))

	if len(res.Choices) < 1 {
		return "", fmt.Errorf("gpt couldn't answer the question, 0 choices were returned from server, the body is: %q", string(responseBody))
	}

	return res.Choices[0].Message.Content, nil
}
