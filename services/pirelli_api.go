package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"time"

	"sending-stocks/models"
)

// PirelliAPIService клиент для API Pirelli
type PirelliAPIService struct {
	BaseURL      string
	AuthLogin    string
	AuthToken    string
	CustomerCode string
	HTTPClient   *http.Client
}

// NewPirelliAPIService создает новый клиент
func NewPirelliAPIService(baseURL, login, token, customerCode string) *PirelliAPIService {
	return &PirelliAPIService{
		BaseURL:      baseURL,
		AuthLogin:    login,
		AuthToken:    token,
		CustomerCode: customerCode,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadFile отправляет файл в Pirelli
func (s *PirelliAPIService) UploadFile(filePath, fileName string) (*models.PirelliResponse, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("не удалось открыть файл: %v", err)
	}
	defer file.Close()

	// Создаем multipart форму
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Устанавливаем boundary как в примере 1С
	writer.SetBoundary("----WebKitFormBoundary7MA4YWxkTrZu0gW")

	// Добавляем поля
	if err := writer.WriteField("action", "upload"); err != nil {
		return nil, fmt.Errorf("ошибка добавления поля action: %v", err)
	}
	if err := writer.WriteField("auth_login", s.AuthLogin); err != nil {
		return nil, fmt.Errorf("ошибка добавления поля auth_login: %v", err)
	}
	if err := writer.WriteField("auth_token", s.AuthToken); err != nil {
		return nil, fmt.Errorf("ошибка добавления поля auth_token: %v", err)
	}

	// Создаем заголовок для файла
	headers := make(textproto.MIMEHeader)
	headers.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="file"; filename="%s"`, fileName))
	headers.Set("Content-Type", "text/csv")

	part, err := writer.CreatePart(headers)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать часть для файла: %v", err)
	}

	// Копируем файл
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("не удалось скопировать файл: %v", err)
	}

	writer.Close()

	// Создаем запрос
	req, err := http.NewRequest("POST", s.BaseURL, &requestBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", "Mozilla/5.0")

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
	var response models.PirelliResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("ошибка парсинга ответа: %v", err)
	}

	return &response, nil
}
