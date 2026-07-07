package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"sending-stocks/models"
)

// CordiantAPIService клиент для API Cordiant
type CordiantAPIService struct {
	BaseURL    string
	Token      string
	Login      string
	Password   string
	HTTPClient *http.Client
}

// NewCordiantAPIService создает новый клиент
func NewCordiantAPIService(baseURL, token, login, password string) *CordiantAPIService {
	return &CordiantAPIService{
		BaseURL:  baseURL,
		Token:    token,
		Login:    login,
		Password: password,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendReport отправляет отчет в Cordiant
func (s *CordiantAPIService) SendReport(fileBase64 string, year, month string) (*models.CordiantResponse, error) {
	// Формируем запрос
	request := models.CordiantRequest{
		Year:   year,
		Month:  month,
		Token:  s.Token,
		Action: "importProcess",
		File:   fileBase64,
	}

	// Сериализуем в JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("ошибка сериализации запроса: %v", err)
	}

	// Логируем запрос (без содержимого файла)
	log.Printf("=== ОТПРАВКА В CORDIANT API ===")
	log.Printf("URL: %s", s.BaseURL)
	log.Printf("Year: %s", year)
	log.Printf("Month: %s", month)
	log.Printf("Token: %s... (скрыт)", s.Token[:20])
	log.Printf("Action: %s", request.Action)
	log.Printf("File size (base64): %d байт", len(fileBase64))
	log.Println("===============================")

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", s.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Выполняем запрос
	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %v", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %v", err)
	}

	// Логируем ответ
	log.Printf("=== ОТВЕТ ОТ CORDIANT API ===")
	log.Printf("Status: %d", resp.StatusCode)
	log.Printf("Response: %s", string(body))
	log.Println("=============================")

	// Пробуем распарсить как универсальный объект
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(body, &rawResponse); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	// Создаем структуру для ответа
	response := &models.CordiantResponse{
		Success: false,
		Message: "Неизвестный формат ответа",
		Data:    rawResponse,
	}

	// Проверяем наличие поля error (ошибка доступа и т.д.)
	if errData, ok := rawResponse["error"]; ok {
		if errMap, ok := errData.(map[string]interface{}); ok {
			code, _ := errMap["code"].(string)
			message, _ := errMap["message"].(string)
			response.Message = fmt.Sprintf("Ошибка %s: %s", code, message)
			return response, nil
		}
	}

	// Проверяем наличие поля data
	if dataVal, ok := rawResponse["data"]; ok {
		// Проверяем, является ли data строкой (например "failure")
		if dataStr, ok := dataVal.(string); ok {
			if dataStr == "failure" {
				response.Success = false
				response.Message = "Ошибка обработки запроса"
				return response, nil
			}
		}

		// Пробуем распарсить data как объект
		dataBytes, err := json.Marshal(dataVal)
		if err == nil {
			var dataObj CordiantResponseData
			if err := json.Unmarshal(dataBytes, &dataObj); err == nil {
				response.Data = dataObj

				// Формируем сообщение на основе статуса
				if dataObj.Status == "success" {
					response.Success = true
					response.Message = dataObj.Message
				} else {
					response.Success = false
					// Собираем подробное сообщение об ошибке
					errorMsg := dataObj.Message
					if dataObj.Title != "" {
						errorMsg = dataObj.Title + " " + dataObj.Message
					}
					if len(dataObj.Warnings) > 0 {
						errorMsg += "\n\nПредупреждения:\n" + formatWarnings(dataObj.Warnings)
					}
					if len(dataObj.ErrorFileStrings) > 0 {
						errorMsg += "\n\nСтроки с ошибками: " + formatIntSlice(dataObj.ErrorFileStrings)
					}
					response.Message = errorMsg
				}
				return response, nil
			}
		}
	}

	return response, nil
}

// ValidateToken проверяет валидность токена
func (s *CordiantAPIService) ValidateToken() error {
	if s.Token == "" {
		return fmt.Errorf("токен не может быть пустым")
	}
	return nil
}

// Вспомогательная функция для форматирования предупреждений
func formatWarnings(warnings []string) string {
	result := ""
	for _, w := range warnings {
		result += "• " + w + "\n"
	}
	return result
}

// Вспомогательная функция для форматирования списка чисел
func formatIntSlice(ints []int) string {
	result := ""
	for i, v := range ints {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%d", v)
	}
	return result
}

// CordiantResponseData структура для успешного ответа с data объектом
type CordiantResponseData struct {
	Status              string   `json:"status"`
	Message             string   `json:"message"`
	Title               string   `json:"title"`
	Content             string   `json:"content"`
	Warnings            []string `json:"warnings"`
	Function            string   `json:"function"`
	IsHavePrevRecords   bool     `json:"isHavePrevRecords"`
	Warehouses          int      `json:"warehouses"`
	WarehousesPositions int      `json:"warehousesPositions"`
	ErrorFileStrings    []int    `json:"errorFileStrings"`
}
