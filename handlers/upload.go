package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"sending-stocks/models"
	"sending-stocks/processors"
	"sending-stocks/services"
)

// UploadHandler обработчик загрузки
type UploadHandler struct {
	adminPassword    string
	uploadDir        string
	processedDir     string
	customerCode     string
	parser           *processors.StockParser
	pirelliAPI       *services.PirelliAPIService
	pirelliProcessor *processors.PirelliProcessor
}

// NewUploadHandler создает новый обработчик
func NewUploadHandler(
	adminPassword string,
	uploadDir string,
	processedDir string,
	customerCode string,
	parser *processors.StockParser,
	api *services.PirelliAPIService,
) *UploadHandler {
	return &UploadHandler{
		adminPassword:    adminPassword,
		uploadDir:        uploadDir,
		processedDir:     processedDir,
		customerCode:     customerCode,
		parser:           parser,
		pirelliAPI:       api,
		pirelliProcessor: processors.NewPirelliProcessor(customerCode),
	}
}

// HandleUpload загружает файл
func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Проверяем пароль
	password := r.FormValue("password")
	if password != h.adminPassword {
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	// Получаем файл
	file, header, err := r.FormFile("file")
	if err != nil {
		sendJSON(w, false, "Ошибка чтения файла: "+err.Error(), nil, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Проверяем расширение
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		sendJSON(w, false, "Можно загружать только XLSX файлы", nil, http.StatusBadRequest)
		return
	}

	// Сохраняем файл
	filename := fmt.Sprintf("%s_%s", time.Now().Format("20060102_150405"), header.Filename)
	filepath := filepath.Join(h.uploadDir, filename)

	dst, err := os.Create(filepath)
	if err != nil {
		sendJSON(w, false, "Ошибка сохранения файла", nil, http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		sendJSON(w, false, "Ошибка сохранения файла", nil, http.StatusInternalServerError)
		return
	}

	sendJSON(w, true, "Файл загружен", map[string]string{
		"filename": filename,
	}, http.StatusOK)
}

// HandleProcess обрабатывает файл
func (h *UploadHandler) HandleProcess(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка парсинга запроса", http.StatusBadRequest)
		return
	}

	if req.Password != h.adminPassword {
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	// Открываем Excel файл
	filePath := filepath.Join(h.uploadDir, req.Filename)
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		sendJSON(w, false, "Ошибка открытия Excel файла: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Парсим
	processed, err := h.parser.Parse(f)
	if err != nil {
		sendJSON(w, false, "Ошибка обработки: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	processed.OriginalFile = req.Filename

	// Сохраняем результат
	resultPath := filepath.Join(h.processedDir, processed.Filename)
	resultData, _ := json.MarshalIndent(processed, "", "  ")
	if err := os.WriteFile(resultPath, resultData, 0644); err != nil {
		sendJSON(w, false, "Ошибка сохранения результата", nil, http.StatusInternalServerError)
		return
	}

	sendJSON(w, true, "Файл обработан", processed, http.StatusOK)
}

// HandleDownloadPirelli скачивает CSV файл для Pirelli
func (h *UploadHandler) HandleDownloadPirelli(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	password := r.URL.Query().Get("password")
	filename := r.URL.Query().Get("file")

	if password != h.adminPassword {
		http.Error(w, "Неверный пароль", http.StatusUnauthorized)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
		return
	}

	if len(processed.PirelliItems) == 0 {
		http.Error(w, "Нет данных Pirelli для скачивания", http.StatusNotFound)
		return
	}

	// Отправляем CSV
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%s", h.pirelliProcessor.GenerateFilename()))

	if err := h.pirelliProcessor.CreateCSV(processed.PirelliItems, w); err != nil {
		http.Error(w, "Ошибка создания CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

// HandleSendPirelli отправляет CSV файл в Pirelli через API
func (h *UploadHandler) HandleSendPirelli(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Ошибка парсинга запроса", http.StatusBadRequest)
		return
	}

	if req.Password != h.adminPassword {
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	if h.pirelliAPI == nil {
		sendJSON(w, false, "API Pirelli не настроен", nil, http.StatusInternalServerError)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, req.Filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		sendJSON(w, false, "Файл не найден", nil, http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		sendJSON(w, false, "Ошибка чтения данных", nil, http.StatusInternalServerError)
		return
	}

	if len(processed.PirelliItems) == 0 {
		sendJSON(w, false, "Нет данных Pirelli для отправки", nil, http.StatusBadRequest)
		return
	}

	// Валидируем
	if err := h.pirelliProcessor.Validate(processed.PirelliItems); err != nil {
		sendJSON(w, false, "Ошибка валидации: "+err.Error(), nil, http.StatusBadRequest)
		return
	}

	// Создаем временный CSV файл
	tmpFile, err := os.CreateTemp("", "pirelli-*.csv")
	if err != nil {
		sendJSON(w, false, "Ошибка создания временного файла", nil, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := h.pirelliProcessor.CreateCSV(processed.PirelliItems, tmpFile); err != nil {
		sendJSON(w, false, "Ошибка создания CSV: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	// Отправляем в Pirelli
	filename := h.pirelliProcessor.GenerateFilename()
	response, err := h.pirelliAPI.UploadFile(tmpFile.Name(), filename)
	if err != nil {
		sendJSON(w, false, "Ошибка отправки: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	sendJSON(w, response.Status, response.Message, response, http.StatusOK)
}

func sendJSON(w http.ResponseWriter, success bool, message string, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(models.UploadResult{
		Success: success,
		Message: message,
		Data:    data,
	})
}
