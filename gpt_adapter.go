package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
)

type GptAdapter struct {
	client   *http.Client
	baseURL  string
	apiToken string
}

type GptResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func NewGptAdapter(apiToken string) *GptAdapter {
	return &GptAdapter{
		client:   &http.Client{},
		baseURL:  "https://lk.neuroapi.host/v1/chat/completions",
		apiToken: apiToken,
	}
}

func (g *GptAdapter) createRequestBody(model, systemMsg, userMsg string) ([]byte, error) {
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
	body, err := g.createRequestBody("gpt-3.5-turbo", systemMsg, userMsg)
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

	log.Printf("GptResponse: %+v", res)
	if len(res.Choices) < 1 {
		return "", fmt.Errorf("gpt couldn't answer the question, 0 choices were returned from server")
	}

	return res.Choices[0].Message.Content, nil
}
