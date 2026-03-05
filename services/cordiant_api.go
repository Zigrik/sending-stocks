package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

	// Создаем HTTP запрос
	req, err := http.NewRequest("POST", s.BaseURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Добавляем базовую аутентификацию если нужно
	// req.SetBasicAuth(s.Login, s.Password)

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

	// Парсим ответ
	var response models.CordiantResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	return &response, nil
}

// ValidateToken проверяет валидность токена
func (s *CordiantAPIService) ValidateToken() error {
	// Можно сделать тестовый запрос для проверки токена
	// Пока просто проверяем что токен не пустой
	if s.Token == "" {
		return fmt.Errorf("токен не может быть пустым")
	}
	return nil
}
