package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"sending-stocks/handlers"
	"sending-stocks/processors"
	"sending-stocks/services"
)

type Config struct {
	ServerPort    string
	AdminPassword string
	UploadDir     string
	ProcessedDir  string

	// SMTP Configuration
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string

	// Email recipients
	PirelliEmails  []string
	IkonEmails     []string
	CordiantEmails []string
	HankookEmails  []string

	// Pirelli бренды
	PirelliBrands []string

	// Pirelli API
	PirelliBaseURL      string
	PirelliLogin        string
	PirelliToken        string
	PirelliCustomerCode string

	// Ikon настройки
	IkonCompanyName string
	IkonSummerA     []string
	IkonSummerB     []string
	IkonSummerC     []string
	IkonSummerD     []string
	IkonWinterA     []string
	IkonWinterB     []string
	IkonWinterC     []string

	// Cordiant бренды
	CordiantBrands []string

	// Cordiant API
	CordiantBaseURL  string
	CordiantToken    string
	CordiantLogin    string
	CordiantPassword string

	// Hankook бренды
	HankookBrands []string
}

var (
	config                Config
	parser                *processors.StockParser
	pirelliAPI            *services.PirelliAPIService
	ikonProcessor         *processors.IkonProcessor
	pirelliExcelProcessor *processors.PirelliExcelProcessor
	cordiantProcessor     *processors.CordiantProcessor
	cordiantAPI           *services.CordiantAPIService
	hankookProcessor      *processors.HankookProcessor
	smtpService           *services.SMTPService
)

func main() {
	// Загружаем конфигурацию
	loadConfig()

	// Выводим конфигурацию для отладки
	log.Println("=== Конфигурация ===")
	log.Printf("ServerPort: %s", config.ServerPort)
	log.Printf("SMTP Host: %s", config.SMTPHost)
	log.Printf("Pirelli Brands: %v", config.PirelliBrands)
	log.Printf("Cordiant Brands: %v", config.CordiantBrands)
	log.Printf("Hankook Brands: %v", config.HankookBrands)
	log.Printf("Pirelli Emails: %v", config.PirelliEmails)
	log.Printf("Ikon Emails: %v", config.IkonEmails)
	log.Printf("Hankook Emails: %v", config.HankookEmails)
	log.Println("===================")

	// Создаем директории
	os.MkdirAll(config.UploadDir, 0755)
	os.MkdirAll(config.ProcessedDir, 0755)

	// Инициализируем парсер с конфигурацией Pirelli брендов
	parser = processors.NewStockParser(12, config.PirelliBrands)

	// Инициализируем SMTP сервис
	if config.SMTPHost != "" && config.SMTPUsername != "" {
		smtpService = services.NewSMTPService(
			config.SMTPHost,
			config.SMTPPort,
			config.SMTPUsername,
			config.SMTPPassword,
			config.SMTPFrom,
		)
		log.Println("SMTP сервис инициализирован")
	} else {
		log.Println("ВНИМАНИЕ: SMTP сервис не настроен")
	}

	// Инициализируем API для Pirelli
	if config.PirelliLogin != "" && config.PirelliToken != "" {
		pirelliAPI = services.NewPirelliAPIService(
			config.PirelliBaseURL,
			config.PirelliLogin,
			config.PirelliToken,
			config.PirelliCustomerCode,
		)
		log.Println("API Pirelli инициализирован")
	} else {
		log.Println("ВНИМАНИЕ: API Pirelli не настроен (нет логина или токена)")
	}

	// Инициализируем процессор Ikon
	summerGroups := map[string][]string{
		"B": config.IkonSummerA,
		"C": config.IkonSummerB,
		"D": config.IkonSummerC,
		"E": config.IkonSummerD,
	}
	winterGroups := map[string][]string{
		"G": config.IkonWinterA,
		"H": config.IkonWinterB,
		"I": config.IkonWinterC,
	}
	ikonProcessor = processors.NewIkonProcessor(config.IkonCompanyName, summerGroups, winterGroups)
	log.Println("Процессор Ikon инициализирован")

	// Инициализируем процессор Excel для Pirelli
	pirelliExcelProcessor = processors.NewPirelliExcelProcessor(config.PirelliCustomerCode, config.PirelliBrands)
	log.Println("Процессор Pirelli Excel инициализирован")

	// Инициализируем процессор Cordiant
	cordiantProcessor = processors.NewCordiantProcessor(config.CordiantBrands)
	log.Println("Процессор Cordiant инициализирован")

	// Инициализируем API для Cordiant
	if config.CordiantToken != "" {
		cordiantAPI = services.NewCordiantAPIService(
			config.CordiantBaseURL,
			config.CordiantToken,
			config.CordiantLogin,
			config.CordiantPassword,
		)
		log.Println("API Cordiant инициализирован")
	} else {
		log.Println("ВНИМАНИЕ: API Cordiant не настроен (нет токена)")
	}

	// Инициализируем процессор Hankook
	hankookProcessor = processors.NewHankookProcessor(config.HankookBrands)
	log.Println("Процессор Hankook инициализирован")

	// Настраиваем маршруты
	setupRoutes()

	log.Printf("Сервер запущен на порту %s", config.ServerPort)
	log.Printf("Веб-интерфейс: http://localhost:%s", config.ServerPort)

	server := &http.Server{
		Addr:         ":" + config.ServerPort,
		Handler:      nil,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func loadConfig() {
	// Загружаем .env файл
	if err := godotenv.Load(); err != nil {
		log.Println("Файл .env не найден, используем переменные окружения")
	}

	// Получаем список брендов Pirelli
	pirelliBrandsStr := getEnv("PIRELLI_BRANDS", "Pirelli,Formula")
	pirelliBrands := strings.Split(pirelliBrandsStr, ",")
	for i, brand := range pirelliBrands {
		pirelliBrands[i] = strings.TrimSpace(brand)
	}

	// Получаем список брендов Cordiant
	cordiantBrandsStr := getEnv("CORDIANT_BRANDS", "Cordiant,Gislaved,Torero,Tunga")
	cordiantBrands := strings.Split(cordiantBrandsStr, ",")
	for i, brand := range cordiantBrands {
		cordiantBrands[i] = strings.TrimSpace(brand)
	}

	// Получаем список брендов Hankook
	hankookBrandsStr := getEnv("HANKOOK_BRANDS", "Hankook,Laufenn")
	hankookBrands := strings.Split(hankookBrandsStr, ",")
	for i, brand := range hankookBrands {
		hankookBrands[i] = strings.TrimSpace(brand)
	}

	// Получаем списки email-адресов
	pirelliEmailsStr := getEnv("PIRELLI_EMAILS", "")
	ikonEmailsStr := getEnv("IKON_EMAILS", "")
	cordiantEmailsStr := getEnv("CORDIANT_EMAILS", "")
	hankookEmailsStr := getEnv("HANKOOK_EMAILS", "")

	config = Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		ProcessedDir:  getEnv("PROCESSED_DIR", "./uploads/processed"),

		// SMTP
		SMTPHost:     getEnv("SMTP_HOST", "smtp.mail.ru"),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),

		// Email recipients
		PirelliEmails:  parseEmailList(pirelliEmailsStr),
		IkonEmails:     parseEmailList(ikonEmailsStr),
		CordiantEmails: parseEmailList(cordiantEmailsStr),
		HankookEmails:  parseEmailList(hankookEmailsStr),

		// Pirelli
		PirelliBrands:       pirelliBrands,
		PirelliBaseURL:      getEnv("PIRELLI_BASE_URL", "https://reports.pirelli.ru/local/templates/dealer/ajax/api.php"),
		PirelliLogin:        getEnv("PIRELLI_LOGIN", ""),
		PirelliToken:        getEnv("PIRELLI_TOKEN", ""),
		PirelliCustomerCode: getEnv("PIRELLI_CUSTOMER_CODE", "5700097"),

		// Ikon
		IkonCompanyName: getEnv("IKON_COMPANY_NAME", "IP SEMISOTNOV"),
		IkonSummerA:     parseBrandList(getEnv("IKON_SUMMER_A", "Ikon Autograph,Nokian Hakka")),
		IkonSummerB:     parseBrandList(getEnv("IKON_SUMMER_B", "Ikon Character,Nordman by Nokian")),
		IkonSummerC:     parseBrandList(getEnv("IKON_SUMMER_C", "Bars")),
		IkonSummerD:     parseBrandList(getEnv("IKON_SUMMER_D", "Attar")),
		IkonWinterA:     parseBrandList(getEnv("IKON_WINTER_A", "Ikon Autograph,Nokian")),
		IkonWinterB:     parseBrandList(getEnv("IKON_WINTER_B", "Ikon Character,Nordman by Nokian")),
		IkonWinterC:     parseBrandList(getEnv("IKON_WINTER_C", "Attar")),

		// Cordiant
		CordiantBrands:   cordiantBrands,
		CordiantBaseURL:  getEnv("CORDIANT_BASE_URL", "https://b2b.cordiant.ru/rest/"),
		CordiantToken:    getEnv("CORDIANT_TOKEN", ""),
		CordiantLogin:    getEnv("CORDIANT_LOGIN", ""),
		CordiantPassword: getEnv("CORDIANT_PASSWORD", ""),

		// Hankook
		HankookBrands: hankookBrands,
	}
}

func parseBrandList(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func parseEmailList(s string) []string {
	if s == "" {
		return []string{}
	}
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" && strings.Contains(p, "@") {
			result = append(result, p)
		}
	}
	return result
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

func getEnvInt(key string, defaultValue int) int {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	var intValue int
	fmt.Sscanf(value, "%d", &intValue)
	return intValue
}

func setupRoutes() {
	webHandler := handlers.NewWebHandler(
		config.PirelliEmails,
		config.IkonEmails,
		config.HankookEmails,
	)

	uploadHandler := handlers.NewUploadHandler(
		config.AdminPassword,
		config.UploadDir,
		config.ProcessedDir,
		config.PirelliCustomerCode,
		config.PirelliBrands,
		config.CordiantBrands,
		config.HankookBrands,
		config.PirelliEmails,
		config.IkonEmails,
		config.CordiantEmails,
		config.HankookEmails,
		parser,
		pirelliAPI,
		cordiantAPI,
		smtpService,
		ikonProcessor,
		pirelliExcelProcessor,
		cordiantProcessor,
		hankookProcessor,
	)

	// Статические файлы
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Маршруты
	http.HandleFunc("/", webHandler.HandleForm)
	http.HandleFunc("/api/check-password", uploadHandler.HandleCheckPassword)
	http.HandleFunc("/api/upload", uploadHandler.HandleUpload)
	http.HandleFunc("/api/process", uploadHandler.HandleProcess)

	// Pirelli
	http.HandleFunc("/api/download-pirelli-csv", uploadHandler.HandleDownloadPirelliCSV)
	http.HandleFunc("/api/send-pirelli", uploadHandler.HandleSendPirelli)
	http.HandleFunc("/api/download-pirelli-excel", uploadHandler.HandleDownloadPirelliExcel)
	http.HandleFunc("/api/send-pirelli-excel", uploadHandler.HandleSendPirelliExcel)

	// Ikon
	http.HandleFunc("/api/download-ikon", uploadHandler.HandleDownloadIkon)
	http.HandleFunc("/api/send-ikon", uploadHandler.HandleSendIkon)

	// Cordiant
	http.HandleFunc("/api/download-cordiant-csv", uploadHandler.HandleDownloadCordiantCSV)
	http.HandleFunc("/api/send-cordiant", uploadHandler.HandleSendCordiant)

	// Hankook
	http.HandleFunc("/api/download-hankook-excel", uploadHandler.HandleDownloadHankookExcel)
	http.HandleFunc("/api/send-hankook", uploadHandler.HandleSendHankook)

	http.HandleFunc("/api/clear", uploadHandler.HandleClear)
}
