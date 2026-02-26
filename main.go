package main

import (
	"log"
	"net/http"
	"os"

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

	// Pirelli
	PirelliBaseURL      string
	PirelliLogin        string
	PirelliToken        string
	PirelliCustomerCode string
}

var (
	config     Config
	parser     *processors.StockParser
	pirelliAPI *services.PirelliAPIService
)

func main() {
	// Загружаем конфигурацию
	loadConfig()

	// Создаем директории
	os.MkdirAll(config.UploadDir, 0755)
	os.MkdirAll(config.ProcessedDir, 0755)

	// Инициализируем парсер
	parser = processors.NewStockParser(12) // данные начинаются с 12 строки

	// Инициализируем API для Pirelli
	if config.PirelliLogin != "" && config.PirelliToken != "" {
		pirelliAPI = services.NewPirelliAPIService(
			config.PirelliBaseURL,
			config.PirelliLogin,
			config.PirelliToken,
			config.PirelliCustomerCode,
		)
	}

	// Настраиваем маршруты
	setupRoutes()

	log.Printf("Сервер запущен на порту %s", config.ServerPort)
	log.Printf("Веб-интерфейс: http://localhost:%s", config.ServerPort)

	if err := http.ListenAndServe(":"+config.ServerPort, nil); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}

func loadConfig() {
	_ = godotenv.Load()

	config = Config{
		ServerPort:    getEnv("SERVER_PORT", "8080"),
		AdminPassword: getEnv("ADMIN_PASSWORD", "admin123"),
		UploadDir:     getEnv("UPLOAD_DIR", "./uploads"),
		ProcessedDir:  getEnv("PROCESSED_DIR", "./uploads/processed"),

		PirelliBaseURL:      getEnv("PIRELLI_BASE_URL", "https://reports.pirelli.ru/local/templates/dealer/ajax/api.php"),
		PirelliLogin:        getEnv("PIRELLI_LOGIN", ""),
		PirelliToken:        getEnv("PIRELLI_TOKEN", ""),
		PirelliCustomerCode: getEnv("PIRELLI_CUSTOMER_CODE", "5700097"),
	}
}

func setupRoutes() {
	webHandler := handlers.NewWebHandler()
	uploadHandler := handlers.NewUploadHandler(
		config.AdminPassword,
		config.UploadDir,
		config.ProcessedDir,
		config.PirelliCustomerCode,
		parser,
		pirelliAPI,
	)

	// Статические файлы
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Маршруты
	http.HandleFunc("/", webHandler.HandleForm)
	http.HandleFunc("/api/upload", uploadHandler.HandleUpload)
	http.HandleFunc("/api/process", uploadHandler.HandleProcess)
	http.HandleFunc("/api/download-pirelli", uploadHandler.HandleDownloadPirelli)
	http.HandleFunc("/api/send-pirelli", uploadHandler.HandleSendPirelli)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
