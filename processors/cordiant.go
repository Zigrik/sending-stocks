package processors

import (
	"encoding/base64"
	"encoding/csv"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/encoding/charmap"

	"sending-stocks/models"
)

// CordiantProcessor обработчик для брендов Cordiant
type CordiantProcessor struct {
	CordiantBrands []string
}

// NewCordiantProcessor создает новый процессор
func NewCordiantProcessor(cordiantBrands []string) *CordiantProcessor {
	return &CordiantProcessor{
		CordiantBrands: cordiantBrands,
	}
}

// isCordiantBrand проверяет, относится ли бренд к группе Cordiant
func (p *CordiantProcessor) isCordiantBrand(brand string) bool {
	brandLower := strings.ToLower(brand)
	for _, cb := range p.CordiantBrands {
		cbLower := strings.ToLower(cb)
		if strings.Contains(brandLower, cbLower) || strings.Contains(cbLower, brandLower) {
			return true
		}
	}
	return false
}

// FilterItems фильтрует позиции для Cordiant (только с кодом производителя)
func (p *CordiantProcessor) FilterItems(items []models.StockItem) []models.CordiantItem {
	result := make([]models.CordiantItem, 0)

	for _, item := range items {
		// Проверяем, относится ли к брендам Cordiant и есть ли количество
		if !p.isCordiantBrand(item.CleanBrand) || item.Quantity <= 0 {
			continue
		}

		// Проверяем наличие кода производителя (ManufacturerSKU)
		if item.ManufacturerSKU == "" {
			continue
		}

		cordiantItem := models.CordiantItem{
			RowNum:     item.RowNum,
			Code:       item.ManufacturerSKU, // Используем ManufacturerSKU как код
			TireSize:   item.TireSize,
			Brand:      item.Name, // Используем Name вместо CleanBrand
			Quantity:   item.Quantity,
			CleanBrand: item.CleanBrand,
		}

		result = append(result, cordiantItem)
	}

	return result
}

// CreateCSV создает CSV для отправки в Cordiant (формат как в примере)
func (p *CordiantProcessor) CreateCSV(items []models.CordiantItem) ([]byte, error) {
	// Используем буфер для накопления данных
	var buf strings.Builder

	// Создаем CSV writer с разделителем ";"
	writer := csv.NewWriter(&buf)
	writer.Comma = ';'

	// Записываем данные
	for _, item := range items {
		row := []string{
			item.Code,
			item.TireSize,
			item.Brand, // Теперь здесь наименование, а не бренд
			fmt.Sprintf("%d", item.Quantity),
		}
		if err := writer.Write(row); err != nil {
			return nil, fmt.Errorf("ошибка записи строки: %v", err)
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, fmt.Errorf("ошибка при записи CSV: %v", err)
	}

	return []byte(buf.String()), nil
}

// CreateCSVWithEncoding создает CSV в указанной кодировке (Windows-1251 для совместимости)
func (p *CordiantProcessor) CreateCSVWithEncoding(items []models.CordiantItem, encoding string) ([]byte, error) {
	// Сначала создаем UTF-8 CSV
	utf8Data, err := p.CreateCSV(items)
	if err != nil {
		return nil, err
	}

	// Если нужна Windows-1251, конвертируем
	if encoding == "windows-1251" {
		encoder := charmap.Windows1251.NewEncoder()
		encoded, err := encoder.String(string(utf8Data))
		if err != nil {
			return nil, fmt.Errorf("ошибка конвертации в Windows-1251: %v", err)
		}
		return []byte(encoded), nil
	}

	return utf8Data, nil
}

// PrepareBase64File подготавливает файл в base64 для отправки
func (p *CordiantProcessor) PrepareBase64File(items []models.CordiantItem) (string, error) {
	csvData, err := p.CreateCSVWithEncoding(items, "windows-1251")
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(csvData), nil
}

// GetCurrentYearMonth возвращает текущий год и месяц
func (p *CordiantProcessor) GetCurrentYearMonth() (string, string) {
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("1") // без ведущего нуля
	return year, month
}

// GenerateFilename генерирует имя файла
func (p *CordiantProcessor) GenerateFilename() string {
	return fmt.Sprintf("Cordiant_Report_%s.csv", time.Now().Format("20060102_150405"))
}
