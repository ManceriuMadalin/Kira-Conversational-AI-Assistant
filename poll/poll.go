package poll

import (
	"KevinGo/transcribe"
	"encoding/json"
	"log"
	"net/http"
)

func StartPolling() string {
	const API_KEY = "Your Key"
	const TRANSCRIBE_URL = "https://api.assemblyai.com/v2/transcript"
	transcriptID := transcribe.Transcribe()

	pollingURL := TRANSCRIBE_URL + "/" + transcriptID
	client := &http.Client{}

	for {
		req, err := http.NewRequest("GET", pollingURL, nil)
		if err != nil {
			log.Println(err)
			return ""
		}

		req.Header.Set("content-type", "application/json")
		req.Header.Set("authorization", API_KEY)

		res, err := client.Do(req)
		if err != nil {
			log.Println(err)
			return ""
		}

		var result map[string]interface{}
		json.NewDecoder(res.Body).Decode(&result)
		res.Body.Close()

		status := result["status"].(string)

		if status == "completed" {
			return result["text"].(string)
			break
		}
	}

	return ""
}
