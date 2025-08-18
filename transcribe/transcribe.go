package transcribe

import (
	"KevinGo/upload"
	"bytes"
	"encoding/json"
	"net/http"
)

func Transcribe() string {
	audioURL := upload.Upload()

	const API_KEY = "Your Key"
	const TRANSCRIBE_URL = "https://api.assemblyai.com/v2/transcript"

	values := map[string]string{"audio_url": audioURL}
	jsonData, _ := json.Marshal(values)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", TRANSCRIBE_URL, bytes.NewBuffer(jsonData))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("authorization", API_KEY)

	res, err := client.Do(req)

	if err != nil {
		return ""
	}

	defer res.Body.Close()

	var result map[string]interface{}
	json.NewDecoder(res.Body).Decode(&result)

	if id, ok := result["id"].(string); ok {
		return id
	}

	return ""
}
