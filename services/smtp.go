package services

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"log"
	"net/smtp"
	"strings"
	"time"
)

// SMTPService сервис для отправки email
type SMTPService struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// NewSMTPService создает новый SMTP сервис
func NewSMTPService(host string, port int, username, password, from string) *SMTPService {
	return &SMTPService{
		Host:     host,
		Port:     port,
		Username: username,
		Password: password,
		From:     from,
	}
}

// SendEmail отправляет email с вложением
func (s *SMTPService) SendEmail(to []string, subject, body string, attachment []byte, filename string) error {
	if len(to) == 0 {
		return fmt.Errorf("нет получателей")
	}

	// Формируем границу для multipart
	boundary := "boundary_" + time.Now().Format("20060102150405")

	// Создаем буфер для письма
	var buf bytes.Buffer

	// Заголовки
	buf.WriteString(fmt.Sprintf("From: %s\r\n", s.From))
	buf.WriteString(fmt.Sprintf("To: %s\r\n", strings.Join(to, ", ")))
	buf.WriteString(fmt.Sprintf("Subject: %s\r\n", subject))
	buf.WriteString("MIME-Version: 1.0\r\n")
	buf.WriteString(fmt.Sprintf("Content-Type: multipart/mixed; boundary=\"%s\"\r\n", boundary))
	buf.WriteString("\r\n")

	// Текстовая часть
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/plain; charset=utf-8\r\n")
	buf.WriteString("Content-Transfer-Encoding: 8bit\r\n")
	buf.WriteString("\r\n")
	buf.WriteString(body)
	buf.WriteString("\r\n")

	// Вложение - XLSX файл
	if len(attachment) > 0 {
		buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
		buf.WriteString(fmt.Sprintf("Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet; name=\"%s\"\r\n", filename))
		buf.WriteString("Content-Transfer-Encoding: base64\r\n")
		buf.WriteString(fmt.Sprintf("Content-Disposition: attachment; filename=\"%s\"\r\n", filename))
		buf.WriteString("\r\n")

		// Кодируем вложение в base64 с правильными переносами строк
		encoded := make([]byte, base64.StdEncoding.EncodedLen(len(attachment)))
		base64.StdEncoding.Encode(encoded, attachment)

		// Разбиваем на строки по 76 символов (стандарт для email)
		for i := 0; i < len(encoded); i += 76 {
			end := i + 76
			if end > len(encoded) {
				end = len(encoded)
			}
			buf.Write(encoded[i:end])
			buf.WriteString("\r\n")
		}

		buf.WriteString("\r\n")
	}

	// Закрывающая граница
	buf.WriteString(fmt.Sprintf("--%s--\r\n", boundary))

	// Аутентификация
	auth := smtp.PlainAuth("", s.Username, s.Password, s.Host)

	// Отправка
	addr := fmt.Sprintf("%s:%d", s.Host, s.Port)
	err := smtp.SendMail(addr, auth, s.From, to, buf.Bytes())
	if err != nil {
		return fmt.Errorf("ошибка отправки email: %v", err)
	}

	log.Printf("Email отправлен на %v, тема: %s, вложение: %s (%d байт)",
		to, subject, filename, len(attachment))
	return nil
}
