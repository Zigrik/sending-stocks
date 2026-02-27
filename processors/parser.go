package processors

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"sending-stocks/models"
)

// StockParser парсер Excel файлов
type StockParser struct {
	StartRow      int
	PirelliBrands []string // список брендов для отчета Pirelli
}

// NewStockParser создает новый парсер
func NewStockParser(startRow int, pirelliBrands []string) *StockParser {
	return &StockParser{
		StartRow:      startRow,
		PirelliBrands: pirelliBrands,
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

		// Если это бренд из списка Pirelli, добавляем в отдельный список
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

	// Столбец C (индекс 2) - бренд + сезонность
	if len(row) > 2 {
		brandField := cleanString(row[2])
		item.Brand = brandField

		// Извлекаем бренд и сезон
		item.CleanBrand, item.Season = extractBrandAndSeason(brandField)

		// Проверяем, относится ли к брендам из списка Pirelli
		item.IsPirelli = p.isPirelliBrand(item.CleanBrand)
	}

	// Столбец F (индекс 5) - код 1С
	if len(row) > 5 {
		item.Code1C = cleanCodeDigits(row[5])
	}

	// Столбец G (индекс 6) - код производителя
	if len(row) > 6 {
		item.ManufacturerSKU = cleanCodeDigits(row[6])
	}

	// Столбец H (индекс 7) - типоразмер
	if len(row) > 7 {
		item.TireSize = cleanString(row[7])
	}

	// Столбец I (индекс 8) - остаток
	if len(row) > 8 {
		quantity, err := strconv.Atoi(cleanDigits(row[8]))
		if err == nil {
			item.Quantity = quantity
		}
	}

	// Столбец J (индекс 9) - цена
	if len(row) > 9 {
		priceStr := cleanPrice(row[9])
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

// isPirelliBrand проверяет, входит ли бренд в список Pirelli
func (p *StockParser) isPirelliBrand(brand string) bool {
	brandLower := strings.ToLower(brand)
	for _, pb := range p.PirelliBrands {
		pbLower := strings.ToLower(pb)
		// Проверяем вхождение (на случай если в бренде есть дополнительные слова)
		if strings.Contains(brandLower, pbLower) || strings.Contains(pbLower, brandLower) {
			return true
		}
	}
	return false
}

// extractBrandAndSeason извлекает бренд и сезон из поля (столбец C)
func extractBrandAndSeason(field string) (brand string, season string) {
	fieldLower := strings.ToLower(field)

	// Определяем сезон
	if strings.Contains(fieldLower, "зима") ||
		strings.Contains(fieldLower, "шип") ||
		strings.Contains(fieldLower, "ice") ||
		strings.Contains(fieldLower, "winter") {
		season = "зима"
	} else if strings.Contains(fieldLower, "лето") ||
		strings.Contains(fieldLower, "summer") {
		season = "лето"
	}

	// Удаляем сезон и *, но сохраняем регистр исходной строки
	brand = field
	brand = strings.Replace(brand, "зима", "", -1)
	brand = strings.Replace(brand, "Зима", "", -1)
	brand = strings.Replace(brand, "лето", "", -1)
	brand = strings.Replace(brand, "Лето", "", -1)
	brand = strings.Replace(brand, "зима", "", -1)
	brand = strings.Replace(brand, "лето", "", -1)
	brand = strings.Replace(brand, "*", "", -1)

	// Удаляем лишние пробелы
	brand = strings.TrimSpace(brand)

	// Заменяем множественные пробелы на один
	result := make([]rune, 0, len(brand))
	lastWasSpace := false

	for _, r := range brand {
		if r == ' ' {
			if !lastWasSpace {
				result = append(result, ' ')
				lastWasSpace = true
			}
		} else {
			result = append(result, r)
			lastWasSpace = false
		}
	}

	return string(result), season
}

// cleanString очищает строку от лишних пробелов, сохраняя кодировку
func cleanString(s string) string {
	// Не используем regexp для кириллицы, так как он может повредить кодировку
	s = strings.TrimSpace(s)

	// Заменяем множественные пробелы на один
	result := make([]rune, 0, len(s))
	lastWasSpace := false

	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			if !lastWasSpace {
				result = append(result, ' ')
				lastWasSpace = true
			}
		} else {
			result = append(result, r)
			lastWasSpace = false
		}
	}

	return string(result)
}

// cleanCodeDigits оставляет только цифры в коде
func cleanCodeDigits(s string) string {
	s = strings.TrimSpace(s)
	// Используем strings.Builder для эффективной сборки строки
	var builder strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// cleanDigits оставляет только цифры (для количества)
func cleanDigits(s string) string {
	s = strings.TrimSpace(s)
	var builder strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

// cleanPrice очищает цену, преобразуя "2 581,00" в 2581.00
func cleanPrice(s string) string {
	s = strings.TrimSpace(s)

	// Удаляем все пробелы
	var builder strings.Builder
	for _, r := range s {
		if r != ' ' && r != '\t' && r != '\n' && r != '\r' {
			builder.WriteRune(r)
		}
	}
	s = builder.String()

	// Заменяем запятую на точку
	s = strings.ReplaceAll(s, ",", ".")

	// Оставляем только цифры и точку
	builder.Reset()
	dotCount := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			builder.WriteRune(r)
		} else if r == '.' && dotCount == 0 {
			builder.WriteRune(r)
			dotCount++
		}
	}

	return builder.String()
}
