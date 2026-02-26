package processors

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"sending-stocks/models"
)

// StockParser парсер Excel файлов
type StockParser struct {
	StartRow int // строка с которой начинаются данные (по умолчанию 12)
}

// NewStockParser создает новый парсер
func NewStockParser(startRow int) *StockParser {
	return &StockParser{
		StartRow: startRow,
	}
}

// Parse парсит Excel файл
func (p *StockParser) Parse(file *excelize.File) (*models.ProcessedFile, error) {
	// Получаем первый лист
	sheets := file.GetSheetList()
	if len(sheets) == 0 {
		return nil, fmt.Errorf("файл не содержит листов")
	}

	rows, err := file.GetRows(sheets[0])
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения строк: %v", err)
	}

	if len(rows) < p.StartRow {
		return nil, fmt.Errorf("в файле недостаточно строк (минимум %d)", p.StartRow)
	}

	result := &models.ProcessedFile{
		Filename:     time.Now().Format("20060102_150405") + "_processed.json",
		UploadDate:   time.Now().Format("2006-01-02 15:04:05"),
		PirelliItems: make([]models.StockItem, 0),
		AllItems:     make([]models.StockItem, 0),
		Stats: models.Stats{
			Errors: make([]string, 0),
		},
	}

	// Парсим строки
	for i := p.StartRow - 1; i < len(rows); i++ {
		row := rows[i]

		// Пропускаем пустые строки
		if len(row) < 10 {
			continue
		}

		item, err := p.parseRow(row, i+1)
		if err != nil {
			result.Stats.InvalidRows++
			result.Stats.Errors = append(result.Stats.Errors,
				fmt.Sprintf("Строка %d: %v", i+1, err))
			continue
		}

		result.Stats.ValidRows++
		result.AllItems = append(result.AllItems, *item)

		// Если это Pirelli или Formula, добавляем в отдельный список
		// Отправляем только позиции с количеством и кодом производителя
		if item.IsPirelli && item.Quantity > 0 && item.ManufacturerSKU != "" {
			result.PirelliItems = append(result.PirelliItems, *item)
			result.Stats.PirelliCount++
		}
	}

	result.Stats.TotalRows = result.Stats.ValidRows + result.Stats.InvalidRows
	return result, nil
}

// parseRow парсит одну строку Excel
func (p *StockParser) parseRow(row []string, rowNum int) (*models.StockItem, error) {
	item := &models.StockItem{
		RowNum: rowNum,
	}

	// Столбец A (индекс 0) - наименование
	if len(row) > 0 {
		item.Name = cleanString(row[0])
		item.IsYearOld = strings.Contains(strings.ToLower(item.Name), "год")
	}

	// Столбец B (индекс 1) - бренд (с сезоном)
	if len(row) > 1 {
		item.Brand = cleanString(row[1])
		item.CleanBrand, item.Season = extractBrandAndSeason(item.Brand)

		// Проверяем, относится ли к Pirelli/Formula
		brandLower := strings.ToLower(item.CleanBrand)
		item.IsPirelli = strings.Contains(brandLower, "pirelli") ||
			strings.Contains(brandLower, "пирелли") ||
			strings.Contains(brandLower, "formula") ||
			strings.Contains(brandLower, "формула")
	}

	// Столбец F (индекс 5) - код 1С
	if len(row) > 5 {
		item.Code1C = cleanCode(row[5])
	}

	// Столбец G (индекс 6) - код производителя
	if len(row) > 6 {
		item.ManufacturerSKU = cleanCode(row[6])
	}

	// Столбец H (индекс 7) - типоразмер
	if len(row) > 7 {
		item.TireSize = cleanString(row[7])
	}

	// Столбец I (индекс 8) - остаток
	if len(row) > 8 {
		quantity, err := strconv.Atoi(cleanNumber(row[8]))
		if err == nil {
			item.Quantity = quantity
		}
	}

	// Столбец J (индекс 9) - цена
	if len(row) > 9 {
		priceStr := strings.ReplaceAll(cleanNumber(row[9]), ",", ".")
		price, err := strconv.ParseFloat(priceStr, 64)
		if err == nil {
			item.Price = price
		}
	}

	// Валидация
	errors := make([]string, 0)
	if item.Code1C == "" {
		errors = append(errors, "отсутствует код 1С")
	}
	if item.TireSize == "" {
		errors = append(errors, "отсутствует типоразмер")
	}

	if len(errors) > 0 {
		return item, fmt.Errorf(strings.Join(errors, "; "))
	}

	return item, nil
}

// extractBrandAndSeason извлекает бренд и сезон
func extractBrandAndSeason(field string) (brand string, season string) {
	field = strings.ToLower(field)

	// Определяем сезон
	if strings.Contains(field, "зима") ||
		strings.Contains(field, "шип") ||
		strings.Contains(field, "ice") ||
		strings.Contains(field, "winter") {
		season = "зима"
	} else if strings.Contains(field, "лето") ||
		strings.Contains(field, "summer") {
		season = "лето"
	}

	// Удаляем сезон и *
	brand = field
	brand = strings.ReplaceAll(brand, "зима", "")
	brand = strings.ReplaceAll(brand, "лето", "")
	brand = strings.ReplaceAll(brand, "*", "")
	brand = strings.TrimSpace(brand)

	// Убираем лишние пробелы
	space := regexp.MustCompile(`\s+`)
	brand = space.ReplaceAllString(brand, " ")

	// Приводим первую букву к заглавной
	if len(brand) > 0 {
		brand = strings.ToUpper(string(brand[0])) + brand[1:]
	}

	return brand, season
}

// cleanString очищает строку
func cleanString(s string) string {
	s = strings.TrimSpace(s)
	space := regexp.MustCompile(`\s+`)
	s = space.ReplaceAllString(s, " ")
	return s
}

// cleanCode очищает код
func cleanCode(s string) string {
	s = strings.TrimSpace(s)
	space := regexp.MustCompile(`\s+`)
	s = space.ReplaceAllString(s, "")
	return s
}

// cleanNumber очищает число
func cleanNumber(s string) string {
	s = strings.TrimSpace(s)
	re := regexp.MustCompile(`[^\d.,-]`)
	s = re.ReplaceAllString(s, "")
	return s
}
