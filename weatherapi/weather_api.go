package weatherapi

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type WeatherData struct {
	Day           string  `json:"day"`
	Location      string  `json:"location"`
	Temperature   float32 `json:"temperature"`
	Description   string  `json:"description"`
	Precipitation float32 `json:"precipitation"`
	Wind          float32 `json:"wind"`
	Humidity      float32 `json:"humidity"`
}

type OpenWeatherResponse struct {
	List []struct {
		Main struct {
			Temp     float32 `json:"temp"`
			Humidity float32 `json:"humidity"`
		} `json:"main"`
		Weather []struct {
			Description string `json:"description"`
		} `json:"weather"`
		Wind struct {
			Speed float32 `json:"speed"`
		} `json:"wind"`
		Rain struct {
			ThreeH float32 `json:"3h"`
		} `json:"rain"`
		DtTxt string `json:"dt_txt"`
	} `json:"list"`
	City struct {
		Name string `json:"name"`
	} `json:"city"`
}

func GetWeather(city string) (*WeatherData, error) {
	apiKey := "Your Key"
	apiURL := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?q=%s&appid=%s&units=metric", city, apiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("eroare la crearea request-ului: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eroare la apelarea API-ului: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API a returnat status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("eroare la citirea răspunsului: %w", err)
	}

	var apiResponse OpenWeatherResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("eroare la parsarea JSON: %w", err)
	}

	if len(apiResponse.List) == 0 {
		return nil, fmt.Errorf("nu s-au găsit date meteo pentru orașul %s", city)
	}

	firstForecast := apiResponse.List[0]

	weather := &WeatherData{
		Day:         firstForecast.DtTxt,
		Location:    apiResponse.City.Name,
		Temperature: firstForecast.Main.Temp,
		Description: "",
		Wind:        firstForecast.Wind.Speed,
		Humidity:    firstForecast.Main.Humidity,
	}

	if len(firstForecast.Weather) > 0 {
		weather.Description = firstForecast.Weather[0].Description
	}

	weather.Precipitation = firstForecast.Rain.ThreeH

	return weather, nil
}
