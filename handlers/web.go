package handlers

import (
	"html/template"
	"net/http"
	"os"
)

// WebHandler обработчик веб-интерфейса
type WebHandler struct{}

// NewWebHandler создает новый обработчик
func NewWebHandler() *WebHandler {
	return &WebHandler{}
}

// HandleForm отображает форму загрузки
func (h *WebHandler) HandleForm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не поддерживается", http.StatusMethodNotAllowed)
		return
	}

	// Читаем HTML шаблон
	htmlContent, err := os.ReadFile("templates/form.html")
	if err != nil {
		htmlContent = []byte(embeddedTemplate())
	}

	t, err := template.New("form").Parse(string(htmlContent))
	if err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	t.Execute(w, nil)
}

func embeddedTemplate() string {
	return `<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Загрузка остатков</title>
</head>
<body>
    <h1>Загрузка остатков</h1>
    <p>Файл шаблона не найден. Пожалуйста, создайте templates/form.html</p>
</body>
</html>`
}
