package upload

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
)

func Upload() string {
	const API_KEY = "Your Key"
	const UPLOAD_URL = "https://api.assemblyai.com/v2/upload"
	audioPath := filepath.Join("assets", "audio.m4a")

	data, err := ioutil.ReadFile(audioPath)
	if err != nil {
		log.Println(err)
		return ""
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", UPLOAD_URL, bytes.NewBuffer(data))
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return ""
	}

	req.Header.Set("authorization", API_KEY)
	req.Header.Set("content-type", "audio/m4a")

	res, err := client.Do(req)
	if err != nil {
		log.Println(err)
		return ""
	}

	defer res.Body.Close()

	var result map[string]interface{}
	err = json.NewDecoder(res.Body).Decode(&result)
	if err != nil {
		log.Println(err)
		return ""
	}

	if uploadURL, ok := result["upload_url"].(string); ok {
		return uploadURL
	}

	return ""
}
