package enhancedcontext

import (
	"KevinGo/weatherapi"
	"fmt"
	"regexp"
	"strings"
)

func analyzeQueryType(query string) string {
	lowerQuery := strings.ToLower(query)

	weatherKeywords := []string{"weather", "rain", "sun", "cold", "hot", "wind"}

	for _, keyword := range weatherKeywords {
		if strings.Contains(lowerQuery, keyword) {
			return "weather"
		}
	}

	return "general"
}

func extractCityFromQuery(query string) string {
	lowerQuery := strings.ToLower(query)

	patterns := []string{
		`(?i)weather in (.+?)(\s|$|\?)`,
		`(?i)vremea în (.+?)(\s|$|\?)`,
		`(?i)how.+weather.+in (.+?)(\s|$|\?)`,
		`(?i)cum.+vreme.+în (.+?)(\s|$|\?)`,
		`(?i)temperature in (.+?)(\s|$|\?)`,
		`(?i)temperatura în (.+?)(\s|$|\?)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(query)
		if len(matches) >= 2 {
			city := strings.TrimSpace(matches[1])
			city = strings.Trim(city, ".,!?")
			return city
		}
	}

	words := strings.Fields(lowerQuery)
	for i, word := range words {
		if (word == "in" || word == "în") && i+1 < len(words) {
			return strings.TrimSpace(words[i+1])
		}
	}

	return "Bucharest"
}

func getClothingRecommendations(weather *weatherapi.WeatherData) string {
	temp := weather.Temperature
	description := strings.ToLower(weather.Description)
	wind := weather.Wind
	humidity := weather.Humidity
	precipitation := weather.Precipitation

	var recommendations []string

	if temp < 0 {
		recommendations = append(recommendations, "🧥 Winter coat, warm layers, gloves, hat, and warm boots")
	} else if temp < 10 {
		recommendations = append(recommendations, "🧥 Jacket or coat, sweater, long pants, closed shoes")
	} else if temp < 20 {
		recommendations = append(recommendations, "👕 Light jacket or cardigan, long pants, comfortable shoes")
	} else if temp < 25 {
		recommendations = append(recommendations, "👔 T-shirt or light shirt, jeans or light pants")
	} else {
		recommendations = append(recommendations, "👕 Light clothing, t-shirt, shorts, sandals or breathable shoes")
	}

	if precipitation > 0 || strings.Contains(description, "rain") || strings.Contains(description, "drizzle") {
		recommendations = append(recommendations, "☔ Umbrella or raincoat, waterproof shoes")
	}

	if wind > 10 {
		recommendations = append(recommendations, "🌪️ Windbreaker or jacket to protect against strong wind")
	}

	if humidity > 80 {
		recommendations = append(recommendations, "💨 Breathable, moisture-wicking fabrics")
	}

	if strings.Contains(description, "snow") {
		recommendations = append(recommendations, "❄️ Warm winter clothing, waterproof boots, gloves")
	}

	if strings.Contains(description, "fog") || strings.Contains(description, "mist") {
		recommendations = append(recommendations, "🌫️ Light jacket due to reduced visibility and humidity")
	}

	return strings.Join(recommendations, "\n• ")
}

func GetSpecializedContext(query string) string {
	queryType := analyzeQueryType(query)

	switch queryType {
	case "weather":
		city := extractCityFromQuery(query)

		weather, err := weatherapi.GetWeather(city)
		if err != nil {
			return fmt.Sprintf(`
WEATHER ERROR CONTEXT:
Sorry, I couldn't get the weather data for %s. Error: %v
Please try asking about weather for another city.`, city, err)
		}

		clothingRecommendations := getClothingRecommendations(weather)

		return fmt.Sprintf(`
CURRENT WEATHER CONTEXT FOR %s:
📍 Location: %s
🗓️ Date/Time: %s
🌡️ Temperature: %.1f°C
🌤️ Conditions: %s
💧 Humidity: %.0f%%
💨 Wind Speed: %.1f m/s
🌧️ Precipitation: %.1f mm

CLOTHING RECOMMENDATIONS:
• %s

INSTRUCTIONS:
- Present this weather information in a natural, conversational way
- Include the clothing recommendations as helpful advice
- Mention that this is current/recent weather data
- Be friendly and helpful in your response
- Always respond in English as instructed in your main context`,
			city, weather.Location, weather.Day, weather.Temperature,
			weather.Description, weather.Humidity, weather.Wind,
			weather.Precipitation, clothingRecommendations)

	default:
		return `
GENERAL WEB CONTEXT:
- Use the most recent information available
- Verify and compare multiple sources when possible
- Mention when information was last updated`
	}
}
