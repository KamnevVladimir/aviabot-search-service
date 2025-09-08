package aviasales

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Client — клиент для Travelpayouts Data API
type Client struct {
	baseURL string
	token   string
	marker  string
	hc      *http.Client
	logger  Logger
}

type Option func(*Client)

func WithHTTPClient(hc *http.Client) Option {
	return func(c *Client) { c.hc = hc }
}

// Logger defines minimal logging capability needed by this client
type Logger interface {
	ExternalAPI(apiName, endpoint string, statusCode int, duration time.Duration, metadata map[string]interface{}) error
}

// WithLogger injects a logger into the client
func WithLogger(l Logger) Option { return func(c *Client) { c.logger = l } }

func NewClient(baseURL, token, marker string, opts ...Option) *Client {
	c := &Client{baseURL: baseURL, token: token, marker: marker, hc: http.DefaultClient}
	for _, o := range opts {
		o(c)
	}
	return c
}

// SearchParams параметры поиска авиабилетов
type SearchParams struct {
	Origin      string // IATA код города отправления
	Destination string // IATA код города назначения
	DepartDate  string // Дата вылета (YYYY-MM-DD или YYYY-MM)
	ReturnDate  string // Дата возвращения (YYYY-MM-DD или YYYY-MM)
	Currency    string // Валюта (rub, usd, eur)
	Limit       int    // Максимальное количество результатов
}

// Flight представляет информацию о рейсе
type Flight struct {
	Origin       string    `json:"origin"`
	Destination  string    `json:"destination"`
	DepartDate   time.Time `json:"depart_date"`
	ReturnDate   time.Time `json:"return_date"`
	Price        int       `json:"price"`
	Airline      string    `json:"airline"`
	FlightNumber int       `json:"flight_number"`
	Duration     int       `json:"duration"`
	Distance     int       `json:"distance"`
	Gate         string    `json:"gate"`
	ExpiresAt    time.Time `json:"expires_at"`
	Actual       bool      `json:"actual"`
}

// TravelpayoutsResponse структура ответа от Travelpayouts API
type TravelpayoutsResponse struct {
	Success  bool                              `json:"success"`
	Data     map[string]map[string]interface{} `json:"data"`
	Currency string                            `json:"currency"`
	Error    string                            `json:"error,omitempty"`
}

// SearchCheap ищет самые дешевые билеты используя /v1/prices/cheap endpoint
func (c *Client) SearchCheap(ctx context.Context, p SearchParams) ([]Flight, error) {
	u, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, err
	}
	u.Path = "/v1/prices/cheap"

	q := u.Query()
	q.Set("origin", p.Origin)
	q.Set("destination", p.Destination)
	q.Set("depart_date", p.DepartDate)
	if p.ReturnDate != "" {
		q.Set("return_date", p.ReturnDate)
	}
	if p.Currency != "" {
		q.Set("currency", p.Currency)
	}
	q.Set("token", c.token)
	if c.marker != "" {
		q.Set("marker", c.marker)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if c.logger != nil {
		_ = c.logger.ExternalAPI(
			"travelpayouts",
			"/v1/prices/cheap",
			resp.StatusCode,
			time.Since(start),
			map[string]interface{}{
				"origin":      p.Origin,
				"destination": p.Destination,
			},
		)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("unexpected status: %s", resp.Status)
	}

	var apiResp TravelpayoutsResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	if !apiResp.Success {
		return nil, fmt.Errorf("API error: %s", apiResp.Error)
	}

	flights := c.parseFlights(apiResp.Data)

	// Ограничиваем количество результатов если указан лимит
	if p.Limit > 0 && len(flights) > p.Limit {
		flights = flights[:p.Limit]
	}

	return flights, nil
}

// parseFlights парсит данные из ответа API в структуру Flight
func (c *Client) parseFlights(data map[string]map[string]interface{}) []Flight {
	var flights []Flight

	for destination, routes := range data {
		for _, routeData := range routes {
			if routeMap, ok := routeData.(map[string]interface{}); ok {
				flight := c.parseFlightData(destination, routeMap)
				if flight != nil {
					flights = append(flights, *flight)
				}
			}
		}
	}

	return flights
}

// parseFlightData парсит данные одного рейса
func (c *Client) parseFlightData(destination string, data map[string]interface{}) *Flight {
	flight := &Flight{
		Destination: destination,
	}

	if price, ok := data["price"].(float64); ok {
		flight.Price = int(price)
	}
	if origin, ok := data["origin"].(string); ok {
		flight.Origin = origin
	}
	if airline, ok := data["airline"].(string); ok {
		flight.Airline = airline
	}
	if flightNum, ok := data["flight_number"].(float64); ok {
		flight.FlightNumber = int(flightNum)
	}
	if duration, ok := data["duration"].(float64); ok {
		flight.Duration = int(duration)
	}
	if distance, ok := data["distance"].(float64); ok {
		flight.Distance = int(distance)
	}
	if gate, ok := data["gate"].(string); ok {
		flight.Gate = gate
	}
	if actual, ok := data["actual"].(bool); ok {
		flight.Actual = actual
	}

	// Парсим даты
	if departStr, ok := data["departure_at"].(string); ok {
		if departTime, err := time.Parse("2006-01-02T15:04:05.000Z", departStr); err == nil {
			flight.DepartDate = departTime
		}
	}
	if returnStr, ok := data["return_at"].(string); ok {
		if returnTime, err := time.Parse("2006-01-02T15:04:05.000Z", returnStr); err == nil {
			flight.ReturnDate = returnTime
		}
	}
	if expiresStr, ok := data["expires_at"].(string); ok {
		if expiresTime, err := time.Parse("2006-01-02T15:04:05.000Z", expiresStr); err == nil {
			flight.ExpiresAt = expiresTime
		}
	}

	return flight
}

// GeneratePartnerLink генерирует партнерскую ссылку для покупки билета
func (c *Client) GeneratePartnerLink(flight Flight, passengers int) string {
	// Формат ссылки Aviasales: https://www.aviasales.com/search/ORIGIN+DDMM+DESTINATION+DDMM
	baseURL := "https://www.aviasales.com/search/"

	// Форматируем даты в формат DDMM
	departDate := flight.DepartDate.Format("0201") // MMDD
	returnDate := flight.ReturnDate.Format("0201") // MMDD

	// Строим поисковый запрос
	searchQuery := fmt.Sprintf("%s%s%s%s",
		flight.Origin, departDate,
		flight.Destination, returnDate)

	// Добавляем параметры
	params := url.Values{}
	params.Set("marker", c.marker)
	params.Set("passengers", strconv.Itoa(passengers))

	return fmt.Sprintf("%s%s?%s", baseURL, searchQuery, params.Encode())
}

// FormatFlightMessage форматирует сообщение с информацией о рейсах для отправки пользователю
func (c *Client) FormatFlightMessage(originCity, destCity string, flights []Flight, passengers int) string {
	if len(flights) == 0 {
		return fmt.Sprintf("😔 К сожалению, билеты %s → %s не найдены", originCity, destCity)
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("✈️ <b>%s → %s</b>\n\n", originCity, destCity))

	for i, flight := range flights {
		if i >= 3 { // Показываем максимум 3 варианта
			break
		}

		msg.WriteString(fmt.Sprintf("🎫 <b>%s</b>\n", c.formatPrice(flight.Price)))
		msg.WriteString(fmt.Sprintf("📅 %s → %s\n",
			c.formatDate(flight.DepartDate),
			c.formatDate(flight.ReturnDate)))
		msg.WriteString(fmt.Sprintf("🛫 %s", flight.Airline))

		if flight.Duration > 0 {
			msg.WriteString(fmt.Sprintf(" • %s", c.formatDuration(flight.Duration)))
		}
		msg.WriteString("\n")

		// Добавляем ссылку на покупку
		link := c.GeneratePartnerLink(flight, passengers)
		msg.WriteString(fmt.Sprintf("🔗 <a href=\"%s\">Купить билет</a>\n\n", link))
	}

	msg.WriteString("💡 <i>Цены указаны за одного пассажира в обе стороны</i>")

	return msg.String()
}

// formatPrice форматирует цену с разделителями тысяч
func (c *Client) formatPrice(price int) string {
	priceStr := strconv.Itoa(price)
	var result strings.Builder

	for i, digit := range priceStr {
		if i > 0 && (len(priceStr)-i)%3 == 0 {
			result.WriteString(" ")
		}
		result.WriteRune(digit)
	}

	return result.String() + " ₽"
}

// formatDate форматирует дату для отображения
func (c *Client) formatDate(t time.Time) string {
	months := []string{
		"янв", "фев", "мар", "апр", "май", "июн",
		"июл", "авг", "сен", "окт", "ноя", "дек",
	}

	return fmt.Sprintf("%d %s", t.Day(), months[t.Month()-1])
}

// formatDuration форматирует длительность полета
func (c *Client) formatDuration(minutes int) string {
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%dч %02dм", hours, mins)
}

// Legacy метод для обратной совместимости (будет удален)
func (c *Client) Search(ctx context.Context, p SearchParams) ([]map[string]any, error) {
	// Конвертируем в старый формат для обратной совместимости
	flights, err := c.SearchCheap(ctx, p)
	if err != nil {
		return nil, err
	}

	var result []map[string]any
	for _, flight := range flights {
		flightMap := map[string]any{
			"price":       flight.Price,
			"origin":      flight.Origin,
			"destination": flight.Destination,
			"airline":     flight.Airline,
		}
		result = append(result, flightMap)
	}

	return result, nil
}
