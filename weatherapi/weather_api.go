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
	apiKey := "your_key"
	apiURL := fmt.Sprintf("https://api.openweathermap.org/data/2.5/forecast?q=%s&appid=%s&units=metric", city, apiKey)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error calling API: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status: %d", res.StatusCode)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %w", err)
	}

	var apiResponse OpenWeatherResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	if len(apiResponse.List) == 0 {
		return nil, fmt.Errorf("no weather data found for city %s", city)
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
