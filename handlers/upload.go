package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
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
	adminPassword         string
	uploadDir             string
	processedDir          string
	customerCode          string
	pirelliBrands         []string
	cordiantBrands        []string
	pirelliEmails         []string
	ikonEmails            []string
	cordiantEmails        []string
	parser                *processors.StockParser
	pirelliAPI            *services.PirelliAPIService
	cordiantAPI           *services.CordiantAPIService
	smtpService           *services.SMTPService
	pirelliProcessor      *processors.PirelliProcessor
	ikonProcessor         *processors.IkonProcessor
	pirelliExcelProcessor *processors.PirelliExcelProcessor
	cordiantProcessor     *processors.CordiantProcessor
}

// NewUploadHandler создает новый обработчик
func NewUploadHandler(
	adminPassword string,
	uploadDir string,
	processedDir string,
	customerCode string,
	pirelliBrands []string,
	cordiantBrands []string,
	pirelliEmails []string,
	ikonEmails []string,
	cordiantEmails []string,
	parser *processors.StockParser,
	pirelliAPI *services.PirelliAPIService,
	cordiantAPI *services.CordiantAPIService,
	smtpService *services.SMTPService,
	ikonProc *processors.IkonProcessor,
	pirelliExcelProc *processors.PirelliExcelProcessor,
	cordiantProc *processors.CordiantProcessor,
) *UploadHandler {
	return &UploadHandler{
		adminPassword:         adminPassword,
		uploadDir:             uploadDir,
		processedDir:          processedDir,
		customerCode:          customerCode,
		pirelliBrands:         pirelliBrands,
		cordiantBrands:        cordiantBrands,
		pirelliEmails:         pirelliEmails,
		ikonEmails:            ikonEmails,
		cordiantEmails:        cordiantEmails,
		parser:                parser,
		pirelliAPI:            pirelliAPI,
		cordiantAPI:           cordiantAPI,
		smtpService:           smtpService,
		pirelliProcessor:      processors.NewPirelliProcessor(customerCode),
		ikonProcessor:         ikonProc,
		pirelliExcelProcessor: pirelliExcelProc,
		cordiantProcessor:     cordiantProc,
	}
}

// HandleCheckPassword проверяет пароль
func (h *UploadHandler) HandleCheckPassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSON(w, false, "Ошибка парсинга запроса", nil, http.StatusBadRequest)
		return
	}

	if req.Password == h.adminPassword {
		log.Println("Проверка пароля: успешно")
		sendJSON(w, true, "Пароль верный", nil, http.StatusOK)
	} else {
		log.Println("Проверка пароля: неверный пароль")
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
	}
}

// HandleUpload загружает файл
func (h *UploadHandler) HandleUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Получаем пароль из формы
	password := r.FormValue("password")

	if password != h.adminPassword {
		log.Println("Ошибка загрузки: неверный пароль")
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	// Получаем файл
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("Ошибка чтения файла: %v", err)
		sendJSON(w, false, "Ошибка чтения файла: "+err.Error(), nil, http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Проверяем расширение
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".xlsx") {
		log.Printf("Ошибка: неверный формат файла %s", header.Filename)
		sendJSON(w, false, "Можно загружать только XLSX файлы", nil, http.StatusBadRequest)
		return
	}

	// Сохраняем файл
	filename := fmt.Sprintf("%s_%s", time.Now().Format("20060102_150405"), header.Filename)
	filepath := filepath.Join(h.uploadDir, filename)

	dst, err := os.Create(filepath)
	if err != nil {
		log.Printf("Ошибка создания файла: %v", err)
		sendJSON(w, false, "Ошибка сохранения файла", nil, http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	bytesWritten, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("Ошибка копирования файла: %v", err)
		sendJSON(w, false, "Ошибка сохранения файла", nil, http.StatusInternalServerError)
		return
	}

	log.Printf("Файл загружен: %s (размер: %d байт)", filename, bytesWritten)

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
		log.Printf("Ошибка парсинга запроса process: %v", err)
		http.Error(w, "Ошибка парсинга запроса", http.StatusBadRequest)
		return
	}

	if req.Password != h.adminPassword {
		log.Println("Ошибка обработки: неверный пароль")
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	// Открываем Excel файл
	filePath := filepath.Join(h.uploadDir, req.Filename)
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		log.Printf("Ошибка открытия Excel файла %s: %v", req.Filename, err)
		sendJSON(w, false, "Ошибка открытия Excel файла: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Парсим
	processed, err := h.parser.Parse(f)
	if err != nil {
		log.Printf("Ошибка парсинга файла %s: %v", req.Filename, err)
		sendJSON(w, false, "Ошибка обработки: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	processed.OriginalFile = req.Filename

	// Сохраняем результат
	resultPath := filepath.Join(h.processedDir, processed.Filename)
	resultData, _ := json.MarshalIndent(processed, "", "  ")
	if err := os.WriteFile(resultPath, resultData, 0644); err != nil {
		log.Printf("Ошибка сохранения результата: %v", err)
		sendJSON(w, false, "Ошибка сохранения результата", nil, http.StatusInternalServerError)
		return
	}

	log.Printf("Файл обработан: %s, всего строк: %d, Pirelli: %d",
		req.Filename, processed.Stats.TotalRows, processed.Stats.PirelliCount)

	sendJSON(w, true, "Файл обработан", processed, http.StatusOK)
}

// HandleDownloadPirelliCSV скачивает CSV файл для Pirelli (только с SKU)
func (h *UploadHandler) HandleDownloadPirelliCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	password := r.URL.Query().Get("password")
	filename := r.URL.Query().Get("file")

	if password != h.adminPassword {
		log.Println("Ошибка скачивания Pirelli CSV: неверный пароль")
		http.Error(w, "Неверный пароль", http.StatusUnauthorized)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		log.Printf("Файл не найден: %s", filename)
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		log.Printf("Ошибка чтения данных из %s: %v", filename, err)
		http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
		return
	}

	if len(processed.PirelliItems) == 0 {
		log.Printf("Нет данных Pirelli в файле %s", filename)
		http.Error(w, "Нет данных Pirelli для скачивания", http.StatusNotFound)
		return
	}

	// Отправляем CSV
	downloadFilename := h.pirelliProcessor.GenerateFilename()
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%s", downloadFilename))

	if err := h.pirelliProcessor.CreateCSV(processed.PirelliItems, w); err != nil {
		log.Printf("Ошибка создания CSV: %v", err)
		http.Error(w, "Ошибка создания CSV: "+err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Скачан файл Pirelli CSV: %s", downloadFilename)
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
		log.Printf("Ошибка парсинга запроса send: %v", err)
		http.Error(w, "Ошибка парсинга запроса", http.StatusBadRequest)
		return
	}

	if req.Password != h.adminPassword {
		log.Println("Ошибка отправки: неверный пароль")
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	if h.pirelliAPI == nil {
		log.Println("Ошибка отправки: API Pirelli не настроен")
		sendJSON(w, false, "API Pirelli не настроен", nil, http.StatusInternalServerError)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, req.Filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		log.Printf("Файл не найден: %s", req.Filename)
		sendJSON(w, false, "Файл не найден", nil, http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		log.Printf("Ошибка чтения данных из %s: %v", req.Filename, err)
		sendJSON(w, false, "Ошибка чтения данных", nil, http.StatusInternalServerError)
		return
	}

	if len(processed.PirelliItems) == 0 {
		log.Printf("Нет данных Pirelli в файле %s", req.Filename)
		sendJSON(w, false, "Нет данных Pirelli для отправки", nil, http.StatusBadRequest)
		return
	}

	// Валидируем
	if err := h.pirelliProcessor.Validate(processed.PirelliItems); err != nil {
		log.Printf("Ошибка валидации: %v", err)
		sendJSON(w, false, "Ошибка валидации: "+err.Error(), nil, http.StatusBadRequest)
		return
	}

	// Создаем временный CSV файл
	tmpFile, err := os.CreateTemp("", "pirelli-*.csv")
	if err != nil {
		log.Printf("Ошибка создания временного файла: %v", err)
		sendJSON(w, false, "Ошибка создания временного файла", nil, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := h.pirelliProcessor.CreateCSV(processed.PirelliItems, tmpFile); err != nil {
		log.Printf("Ошибка создания CSV: %v", err)
		sendJSON(w, false, "Ошибка создания CSV: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	// Отправляем в Pirelli
	filename := h.pirelliProcessor.GenerateFilename()
	response, err := h.pirelliAPI.UploadFile(tmpFile.Name(), filename)
	if err != nil {
		log.Printf("Ошибка отправки в Pirelli: %v", err)
		sendJSON(w, false, "Ошибка отправки: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	if response.Status {
		log.Printf("Файл отправлен в Pirelli: %s, ответ: %s", filename, response.Message)
	} else {
		log.Printf("Ошибка отправки в Pirelli: %s", response.Message)
	}

	sendJSON(w, response.Status, response.Message, response, http.StatusOK)
}

// HandleDownloadPirelliExcel скачивает Excel отчет для Pirelli (все позиции)
func (h *UploadHandler) HandleDownloadPirelliExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	password := r.URL.Query().Get("password")
	filename := r.URL.Query().Get("file")

	if password != h.adminPassword {
		log.Println("Ошибка скачивания Pirelli Excel: неверный пароль")
		http.Error(w, "Неверный пароль", http.StatusUnauthorized)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		log.Printf("Файл не найден: %s", filename)
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		log.Printf("Ошибка чтения данных из %s: %v", filename, err)
		http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
		return
	}

	if h.pirelliExcelProcessor == nil {
		log.Println("Ошибка: процессор Pirelli Excel не инициализирован")
		http.Error(w, "Процессор Pirelli Excel не настроен", http.StatusInternalServerError)
		return
	}

	// Создаем Excel отчет
	f, err := h.pirelliExcelProcessor.CreateExcelReport(processed.AllItems)
	if err != nil {
		log.Printf("Ошибка создания отчета Pirelli Excel: %v", err)
		http.Error(w, "Ошибка создания отчета", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Сохраняем во временный файл
	tmpFile, err := os.CreateTemp("", "pirelli-excel-*.xlsx")
	if err != nil {
		log.Printf("Ошибка создания временного файла: %v", err)
		http.Error(w, "Ошибка создания файла", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := f.SaveAs(tmpFile.Name()); err != nil {
		log.Printf("Ошибка сохранения Excel: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}

	// Отправляем файл
	downloadFilename := h.pirelliExcelProcessor.GenerateFilename()
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%s", downloadFilename))

	http.ServeFile(w, r, tmpFile.Name())

	log.Printf("Скачан отчет Pirelli Excel: %s", downloadFilename)
}

// HandleSendPirelliExcel отправляет Excel отчет по email
func (h *UploadHandler) HandleSendPirelliExcel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
		Filename string `json:"filename"`
		Emails   string `json:"emails"` // Получаем email-адреса из формы
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Ошибка парсинга запроса send-pirelli-excel: %v", err)
		sendJSON(w, false, "Ошибка парсинга запроса", nil, http.StatusBadRequest)
		return
	}

	if req.Password != h.adminPassword {
		log.Println("Ошибка отправки Pirelli Excel: неверный пароль")
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	if h.smtpService == nil {
		log.Println("Ошибка отправки: SMTP сервис не настроен")
		sendJSON(w, false, "SMTP сервис не настроен", nil, http.StatusInternalServerError)
		return
	}

	// Парсим email-адреса из формы
	emailList := parseEmailList(req.Emails)
	if len(emailList) == 0 {
		log.Println("Ошибка отправки: не указаны email-адреса для Pirelli")
		sendJSON(w, false, "Не указаны email-адреса получателей", nil, http.StatusBadRequest)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, req.Filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		log.Printf("Файл не найден: %s", req.Filename)
		sendJSON(w, false, "Файл не найден", nil, http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		log.Printf("Ошибка чтения данных из %s: %v", req.Filename, err)
		sendJSON(w, false, "Ошибка чтения данных", nil, http.StatusInternalServerError)
		return
	}

	// Фильтруем позиции Pirelli
	allPirelliItems := make([]models.StockItem, 0)
	for _, item := range processed.AllItems {
		brandLower := strings.ToLower(item.CleanBrand)
		for _, pb := range h.pirelliBrands {
			if strings.Contains(brandLower, strings.ToLower(pb)) && item.Quantity > 0 {
				allPirelliItems = append(allPirelliItems, item)
				break
			}
		}
	}

	if len(allPirelliItems) == 0 {
		sendJSON(w, false, "Нет данных Pirelli для отправки", nil, http.StatusBadRequest)
		return
	}

	// Создаем Excel файл
	f, err := h.pirelliExcelProcessor.CreateExcelReport(processed.AllItems)
	if err != nil {
		log.Printf("Ошибка создания отчета Pirelli Excel: %v", err)
		sendJSON(w, false, "Ошибка создания отчета", nil, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Сохраняем во временный файл
	tmpFile, err := os.CreateTemp("", "pirelli-excel-*.xlsx")
	if err != nil {
		log.Printf("Ошибка создания временного файла: %v", err)
		sendJSON(w, false, "Ошибка создания файла", nil, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := f.SaveAs(tmpFile.Name()); err != nil {
		log.Printf("Ошибка сохранения Excel: %v", err)
		sendJSON(w, false, "Ошибка сохранения файла", nil, http.StatusInternalServerError)
		return
	}

	// Читаем файл для отправки
	fileData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		log.Printf("Ошибка чтения файла: %v", err)
		sendJSON(w, false, "Ошибка чтения файла", nil, http.StatusInternalServerError)
		return
	}

	// Отправляем по email
	filename := h.pirelliExcelProcessor.GenerateFilename()
	subject := fmt.Sprintf("Отчет Pirelli от %s", time.Now().Format("02.01.2006"))
	body := fmt.Sprintf("Отчет Pirelli сформирован %s.\nВсего позиций: %d\nОбщее количество: %d",
		time.Now().Format("02.01.2006 15:04:05"),
		len(allPirelliItems),
		len(processed.AllItems))

	err = h.smtpService.SendEmail(emailList, subject, body, fileData, filename)
	if err != nil {
		log.Printf("Ошибка отправки email: %v", err)
		sendJSON(w, false, "Ошибка отправки email: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	sendJSON(w, true, fmt.Sprintf("Отчет отправлен на %d адресов", len(emailList)), map[string]interface{}{
		"emails": emailList,
		"count":  len(allPirelliItems),
	}, http.StatusOK)
}

// HandleSendIkon отправляет отчет Ikon по email
func (h *UploadHandler) HandleSendIkon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
		Filename string `json:"filename"`
		Emails   string `json:"emails"` // Получаем email-адреса из формы
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Ошибка парсинга запроса send-ikon: %v", err)
		sendJSON(w, false, "Ошибка парсинга запроса", nil, http.StatusBadRequest)
		return
	}

	if req.Password != h.adminPassword {
		log.Println("Ошибка отправки Ikon: неверный пароль")
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	if h.smtpService == nil {
		log.Println("Ошибка отправки: SMTP сервис не настроен")
		sendJSON(w, false, "SMTP сервис не настроен", nil, http.StatusInternalServerError)
		return
	}

	// Парсим email-адреса из формы
	emailList := parseEmailList(req.Emails)
	if len(emailList) == 0 {
		log.Println("Ошибка отправки: не указаны email-адреса для Ikon")
		sendJSON(w, false, "Не указаны email-адреса получателей", nil, http.StatusBadRequest)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, req.Filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		log.Printf("Файл не найден: %s", req.Filename)
		sendJSON(w, false, "Файл не найден", nil, http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		log.Printf("Ошибка чтения данных из %s: %v", req.Filename, err)
		sendJSON(w, false, "Ошибка чтения данных", nil, http.StatusInternalServerError)
		return
	}

	if h.ikonProcessor == nil {
		log.Println("Ошибка: процессор Ikon не инициализирован")
		sendJSON(w, false, "Процессор Ikon не настроен", nil, http.StatusInternalServerError)
		return
	}

	// Создаем отчет Ikon
	f, err := h.ikonProcessor.CreateReport(processed.AllItems)
	if err != nil {
		log.Printf("Ошибка создания отчета Ikon: %v", err)
		sendJSON(w, false, "Ошибка создания отчета", nil, http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Сохраняем во временный файл
	tmpFile, err := os.CreateTemp("", "ikon-*.xlsx")
	if err != nil {
		log.Printf("Ошибка создания временного файла: %v", err)
		sendJSON(w, false, "Ошибка создания файла", nil, http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := f.SaveAs(tmpFile.Name()); err != nil {
		log.Printf("Ошибка сохранения Excel: %v", err)
		sendJSON(w, false, "Ошибка сохранения файла", nil, http.StatusInternalServerError)
		return
	}

	// Читаем файл для отправки
	fileData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		log.Printf("Ошибка чтения файла: %v", err)
		sendJSON(w, false, "Ошибка чтения файла", nil, http.StatusInternalServerError)
		return
	}

	// Отправляем по email
	filename := h.ikonProcessor.GenerateFilename()
	subject := fmt.Sprintf("Отчет Ikon от %s", time.Now().Format("02.01.2006"))

	// Получаем статистику для тела письма
	_, _, ikonTotal, allTotal := h.ikonProcessor.CalculateSums(processed.AllItems)

	body := fmt.Sprintf("Отчет Ikon сформирован %s.\nВсего позиций в отчете: %d\nОбщее количество по всем брендам: %d",
		time.Now().Format("02.01.2006 15:04:05"),
		ikonTotal,
		allTotal)

	err = h.smtpService.SendEmail(emailList, subject, body, fileData, filename)
	if err != nil {
		log.Printf("Ошибка отправки email: %v", err)
		sendJSON(w, false, "Ошибка отправки email: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	sendJSON(w, true, fmt.Sprintf("Отчет отправлен на %d адресов", len(emailList)), map[string]interface{}{
		"emails": emailList,
		"total":  ikonTotal,
	}, http.StatusOK)
}

// Вспомогательная функция для парсинга email-адресов
func parseEmailList(emailsStr string) []string {
	if emailsStr == "" {
		return []string{}
	}
	parts := strings.Split(emailsStr, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && strings.Contains(p, "@") {
			result = append(result, p)
		}
	}
	return result
}

// HandleDownloadIkon скачивает Excel отчет для Ikon
func (h *UploadHandler) HandleDownloadIkon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	password := r.URL.Query().Get("password")
	filename := r.URL.Query().Get("file")

	if password != h.adminPassword {
		log.Println("Ошибка скачивания Ikon: неверный пароль")
		http.Error(w, "Неверный пароль", http.StatusUnauthorized)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		log.Printf("Файл не найден: %s", filename)
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		log.Printf("Ошибка чтения данных из %s: %v", filename, err)
		http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
		return
	}

	if h.ikonProcessor == nil {
		log.Println("Ошибка: процессор Ikon не инициализирован")
		http.Error(w, "Процессор Ikon не настроен", http.StatusInternalServerError)
		return
	}

	// Создаем отчет Ikon
	f, err := h.ikonProcessor.CreateReport(processed.AllItems)
	if err != nil {
		log.Printf("Ошибка создания отчета Ikon: %v", err)
		http.Error(w, "Ошибка создания отчета", http.StatusInternalServerError)
		return
	}
	defer f.Close()

	// Сохраняем во временный файл
	tmpFile, err := os.CreateTemp("", "ikon-*.xlsx")
	if err != nil {
		log.Printf("Ошибка создания временного файла: %v", err)
		http.Error(w, "Ошибка создания файла", http.StatusInternalServerError)
		return
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if err := f.SaveAs(tmpFile.Name()); err != nil {
		log.Printf("Ошибка сохранения Excel: %v", err)
		http.Error(w, "Ошибка сохранения файла", http.StatusInternalServerError)
		return
	}

	// Отправляем файл
	downloadFilename := h.ikonProcessor.GenerateFilename()
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%s", downloadFilename))

	http.ServeFile(w, r, tmpFile.Name())

	log.Printf("Скачан отчет Ikon: %s", downloadFilename)
}

// HandleDownloadCordiantCSV скачивает CSV файл для Cordiant
func (h *UploadHandler) HandleDownloadCordiantCSV(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	password := r.URL.Query().Get("password")
	filename := r.URL.Query().Get("file")

	if password != h.adminPassword {
		log.Println("Ошибка скачивания Cordiant CSV: неверный пароль")
		http.Error(w, "Неверный пароль", http.StatusUnauthorized)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		log.Printf("Файл не найден: %s", filename)
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		log.Printf("Ошибка чтения данных из %s: %v", filename, err)
		http.Error(w, "Ошибка чтения данных", http.StatusInternalServerError)
		return
	}

	if h.cordiantProcessor == nil {
		log.Println("Ошибка: процессор Cordiant не инициализирован")
		http.Error(w, "Процессор Cordiant не настроен", http.StatusInternalServerError)
		return
	}

	// Фильтруем позиции для Cordiant
	cordiantItems := h.cordiantProcessor.FilterItems(processed.AllItems)

	if len(cordiantItems) == 0 {
		log.Printf("Нет данных Cordiant в файле %s", filename)
		http.Error(w, "Нет данных Cordiant для скачивания", http.StatusNotFound)
		return
	}

	// Создаем CSV
	csvData, err := h.cordiantProcessor.CreateCSVWithEncoding(cordiantItems, "windows-1251")
	if err != nil {
		log.Printf("Ошибка создания CSV: %v", err)
		http.Error(w, "Ошибка создания CSV", http.StatusInternalServerError)
		return
	}

	// Отправляем файл
	downloadFilename := h.cordiantProcessor.GenerateFilename()
	w.Header().Set("Content-Type", "text/csv; charset=windows-1251")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf("attachment; filename=%s", downloadFilename))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(csvData)))

	if _, err := w.Write(csvData); err != nil {
		log.Printf("Ошибка отправки файла: %v", err)
	}

	log.Printf("Скачан файл Cordiant CSV: %s, позиций: %d", downloadFilename, len(cordiantItems))
}

// HandleSendCordiant отправляет отчет в Cordiant через API
func (h *UploadHandler) HandleSendCordiant(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
		Filename string `json:"filename"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Ошибка парсинга запроса send-cordiant: %v", err)
		sendJSON(w, false, "Ошибка парсинга запроса", nil, http.StatusBadRequest)
		return
	}

	if req.Password != h.adminPassword {
		log.Println("Ошибка отправки Cordiant: неверный пароль")
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	if h.cordiantAPI == nil {
		log.Println("Ошибка отправки: API Cordiant не настроен")
		sendJSON(w, false, "API Cordiant не настроен", nil, http.StatusInternalServerError)
		return
	}

	// Загружаем обработанные данные
	resultPath := filepath.Join(h.processedDir, req.Filename)
	data, err := os.ReadFile(resultPath)
	if err != nil {
		log.Printf("Файл не найден: %s", req.Filename)
		sendJSON(w, false, "Файл не найден", nil, http.StatusNotFound)
		return
	}

	var processed models.ProcessedFile
	if err := json.Unmarshal(data, &processed); err != nil {
		log.Printf("Ошибка чтения данных из %s: %v", req.Filename, err)
		sendJSON(w, false, "Ошибка чтения данных", nil, http.StatusInternalServerError)
		return
	}

	if h.cordiantProcessor == nil {
		log.Println("Ошибка: процессор Cordiant не инициализирован")
		sendJSON(w, false, "Процессор Cordiant не настроен", nil, http.StatusInternalServerError)
		return
	}

	// Фильтруем позиции для Cordiant
	cordiantItems := h.cordiantProcessor.FilterItems(processed.AllItems)

	if len(cordiantItems) == 0 {
		log.Printf("Нет данных Cordiant в файле %s", req.Filename)
		sendJSON(w, false, "Нет данных Cordiant для отправки", nil, http.StatusBadRequest)
		return
	}

	// Подготавливаем файл в base64
	fileBase64, err := h.cordiantProcessor.PrepareBase64File(cordiantItems)
	if err != nil {
		log.Printf("Ошибка подготовки файла: %v", err)
		sendJSON(w, false, "Ошибка подготовки файла: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	// Получаем текущий год и месяц
	year, month := h.cordiantProcessor.GetCurrentYearMonth()

	// Отправляем в Cordiant
	response, err := h.cordiantAPI.SendReport(fileBase64, year, month)
	if err != nil {
		log.Printf("Ошибка отправки в Cordiant: %v", err)
		sendJSON(w, false, "Ошибка отправки: "+err.Error(), nil, http.StatusInternalServerError)
		return
	}

	if response.Success {
		log.Printf("Отчет успешно отправлен в Cordiant, позиций: %d", len(cordiantItems))
	} else {
		log.Printf("Ошибка отправки в Cordiant: %s", response.Message)
	}

	sendJSON(w, response.Success, response.Message, response.Data, http.StatusOK)
}

// HandleClear очищает директории загрузок
func (h *UploadHandler) HandleClear(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("Ошибка парсинга запроса clear: %v", err)
		sendJSON(w, false, "Ошибка парсинга запроса", nil, http.StatusBadRequest)
		return
	}

	if req.Password != h.adminPassword {
		log.Println("Ошибка очистки: неверный пароль")
		sendJSON(w, false, "Неверный пароль", nil, http.StatusUnauthorized)
		return
	}

	// Счетчики для логирования
	uploadCount := 0
	processedCount := 0

	// Удаляем файлы из uploads
	files, err := os.ReadDir(h.uploadDir)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() {
				filePath := filepath.Join(h.uploadDir, file.Name())
				if err := os.Remove(filePath); err == nil {
					uploadCount++
					log.Printf("Удален файл загрузки: %s", file.Name())
				}
			}
		}
	}

	// Удаляем файлы из processed
	files, err = os.ReadDir(h.processedDir)
	if err == nil {
		for _, file := range files {
			if !file.IsDir() {
				filePath := filepath.Join(h.processedDir, file.Name())
				if err := os.Remove(filePath); err == nil {
					processedCount++
					log.Printf("Удален обработанный файл: %s", file.Name())
				}
			}
		}
	}

	log.Printf("Очистка завершена: удалено %d файлов загрузки, %d обработанных файлов", uploadCount, processedCount)

	sendJSON(w, true, "Память очищена", map[string]interface{}{
		"upload_removed":    uploadCount,
		"processed_removed": processedCount,
	}, http.StatusOK)
}

func sendJSON(w http.ResponseWriter, success bool, message string, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.Encode(models.UploadResult{
		Success: success,
		Message: message,
		Data:    data,
	})
}
