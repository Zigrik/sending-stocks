package processors

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"sending-stocks/models"
)

// PirelliProcessor обработчик для Pirelli
type PirelliProcessor struct {
	CustomerCode string
}

// NewPirelliProcessor создает новый процессор
func NewPirelliProcessor(customerCode string) *PirelliProcessor {
	return &PirelliProcessor{
		CustomerCode: customerCode,
	}
}

// CreateCSV создает CSV для отправки в Pirelli (стандартный формат без лишних запятых)
func (p *PirelliProcessor) CreateCSV(items []models.StockItem, writer io.Writer) error {
	csvWriter := csv.NewWriter(writer)
	defer csvWriter.Flush()

	// Заголовки
	headers := []string{
		"Pirelli Customer Code",
		"Customer Material Code",
		"Pirelli Material Code",
		"Material Description",
		"Stock Quantity",
		"Stock Date",
	}
	if err := csvWriter.Write(headers); err != nil {
		return fmt.Errorf("ошибка записи заголовков: %v", err)
	}

	stockDate := time.Now().Format("20060102")

	// Данные
	for _, item := range items {
		row := []string{
			p.CustomerCode,
			item.Code1C,
			item.ManufacturerSKU,
			item.Name,
			fmt.Sprintf("%d", item.Quantity),
			stockDate,
		}
		if err := csvWriter.Write(row); err != nil {
			return fmt.Errorf("ошибка записи строки: %v", err)
		}
	}

	return nil
}

// GenerateFilename генерирует имя файла для скачивания/сохранения (только дата)
func (p *PirelliProcessor) GenerateFilename() string {
	return fmt.Sprintf("IR_%s_%s.csv",
		p.CustomerCode,
		time.Now().Format("20060102"))
}

// GenerateFilenameWithTime генерирует имя файла с временем (для отладки)
func (p *PirelliProcessor) GenerateFilenameWithTime() string {
	return fmt.Sprintf("IR_%s_%s.csv",
		p.CustomerCode,
		time.Now().Format("20060102_150405"))
}

// Validate проверяет данные
func (p *PirelliProcessor) Validate(items []models.StockItem) error {
	for _, item := range items {
		if item.Quantity <= 0 {
			return fmt.Errorf("позиция %s имеет нулевое количество", item.Code1C)
		}
		if item.ManufacturerSKU == "" {
			return fmt.Errorf("позиция %s не имеет кода производителя", item.Code1C)
		}
	}
	return nil
}
