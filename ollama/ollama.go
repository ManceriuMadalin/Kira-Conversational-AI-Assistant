package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type OllamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
}

func AskQuestion(question string) (string, error) {
	const OLLAMA_URL = "http://localhost:11434/api/generate"

	requestBody := OllamaRequest{
		Model:  "llama3.2",
		Prompt: question,
		Stream: false,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON error: %v", err)
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", OLLAMA_URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("request error: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama not responding - check if 'ollama serve' is running: %v", err)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := ioutil.ReadAll(res.Body)
		return "", fmt.Errorf("Ollama error %d: %s", res.StatusCode, string(body))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", fmt.Errorf("response reading error: %v", err)
	}

	var response OllamaResponse
	err = json.Unmarshal(body, &response)
	if err != nil {
		return "", fmt.Errorf("JSON parse error: %v", err)
	}

	return response.Response, nil
}

func AskWithContext(question, context string) (string, error) {
	fullPrompt := fmt.Sprintf("Context: %s\n\nQuestion: %s", context, question)
	return AskQuestion(fullPrompt)
}

func CheckOllamaStatus() error {
	const OLLAMA_URL = "http://localhost:11434/api/tags"
	
	client := &http.Client{}
	req, err := http.NewRequest("GET", OLLAMA_URL, nil)
	if err != nil {
		return fmt.Errorf("request error: %v", err)
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Ollama is not running. Start it with: ollama serve")
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return fmt.Errorf("Ollama not responding correctly")
	}

	return nil
}
