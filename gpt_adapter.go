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

func (g *GptAdapter) AskGpt(systemMsg, userMsg string) string {
	body, err := g.createRequestBody("gpt-3.5-turbo", systemMsg, userMsg)
	if err != nil {
		log.Fatalf("Failed to create request body: %v", err)
	}

	request, err := http.NewRequest("POST", g.baseURL, bytes.NewBuffer(body))
	if err != nil {
		log.Fatalf("Failed to create request: %v", err)
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", fmt.Sprintf("Bearer %s", g.apiToken))

	response, err := g.client.Do(request)
	if err != nil {
		log.Fatalf("Request failed: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatalf("Failed to read response: %v", err)
	}

	res := GptResponse{}
	err = json.Unmarshal(responseBody, &res)
	if err != nil {
		return ""
	}
	log.Println("GptResponse:", res.Choices[0].Message.Content)

	return res.Choices[0].Message.Content
}
