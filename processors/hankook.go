package processors

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"sending-stocks/models"
)

// HankookProcessor обработчик для брендов Hankook и Laufenn
type HankookProcessor struct {
	HankookBrands []string
}

// NewHankookProcessor создает новый процессор
func NewHankookProcessor(hankookBrands []string) *HankookProcessor {
	return &HankookProcessor{
		HankookBrands: hankookBrands,
	}
}

// isHankookBrand проверяет, относится ли бренд к Hankook/Laufenn
func (p *HankookProcessor) isHankookBrand(brand string) bool {
	brandLower := strings.ToLower(brand)
	for _, hb := range p.HankookBrands {
		hbLower := strings.ToLower(hb)
		if strings.Contains(brandLower, hbLower) || strings.Contains(hbLower, brandLower) {
			return true
		}
	}
	return false
}

// truncateManufacturerSKU обрезает код производителя до последних 7 символов
func (p *HankookProcessor) truncateManufacturerSKU(sku string) string {
	// Удаляем пробелы
	sku = strings.TrimSpace(sku)
	if sku == "" {
		return ""
	}

	// Если длина меньше или равна 7, возвращаем как есть
	if len(sku) <= 7 {
		return sku
	}

	// Возвращаем последние 7 символов
	return sku[len(sku)-7:]
}

// FilterItems фильтрует позиции для Hankook
func (p *HankookProcessor) FilterItems(items []models.StockItem) []models.StockItem {
	result := make([]models.StockItem, 0)

	for _, item := range items {
		// Проверяем, относится ли к брендам Hankook/Laufenn и есть ли количество
		if !p.isHankookBrand(item.CleanBrand) || item.Quantity <= 0 {
			continue
		}

		result = append(result, item)
	}

	return result
}

// CreateExcelReport создает Excel отчет для Hankook
func (p *HankookProcessor) CreateExcelReport(items []models.StockItem) (*excelize.File, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Создаем лист
	index, _ := f.NewSheet("Hankook Report")
	f.SetActiveSheet(index)

	// Удаляем лист по умолчанию
	f.DeleteSheet("Sheet1")

	// Заголовки
	headers := []string{
		"Manufacturer Code", "Product Name", "Brand", "Season", "Quantity",
	}

	for i, header := range headers {
		col := string(rune('A' + i))
		f.SetCellValue("Hankook Report", fmt.Sprintf("%s1", col), header)
	}

	// Фильтруем позиции
	hankookItems := p.FilterItems(items)

	// Заполняем данные
	row := 2
	for _, item := range hankookItems {
		// Manufacturer Code (обрезанный до 7 символов)
		manufacturerCode := p.truncateManufacturerSKU(item.ManufacturerSKU)
		f.SetCellValue("Hankook Report", fmt.Sprintf("A%d", row), manufacturerCode)

		// Product Name (наименование товара)
		f.SetCellValue("Hankook Report", fmt.Sprintf("B%d", row), item.Name)

		// Brand (очищенный бренд)
		f.SetCellValue("Hankook Report", fmt.Sprintf("C%d", row), item.CleanBrand)

		// Season
		season := ""
		if item.Season == "лето" {
			season = "Summer"
		} else if item.Season == "зима" {
			season = "Winter"
		}
		f.SetCellValue("Hankook Report", fmt.Sprintf("D%d", row), season)

		// Quantity
		f.SetCellValue("Hankook Report", fmt.Sprintf("E%d", row), item.Quantity)

		row++
	}

	// Стили для заголовков
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"}, // Серый фон для Hankook
			Pattern: 1,
		},
	})
	f.SetCellStyle("Hankook Report", "A1", "E1", headerStyle)

	// Стиль для ячеек с кодом (выравнивание по левому краю)
	leftAlignStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "left",
		},
	})
	f.SetCellStyle("Hankook Report", "A2", fmt.Sprintf("A%d", row-1), leftAlignStyle)

	// Стиль для чисел (выравнивание по центру)
	numberStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})
	f.SetCellStyle("Hankook Report", "E2", fmt.Sprintf("E%d", row-1), numberStyle)

	// Устанавливаем ширину колонок
	colWidths := map[string]float64{
		"A": 20, // Manufacturer Code
		"B": 50, // Product Name
		"C": 15, // Brand
		"D": 10, // Season
		"E": 12, // Quantity
	}
	for col, width := range colWidths {
		f.SetColWidth("Hankook Report", col, col, width)
	}

	return f, nil
}

// GenerateFilename генерирует имя файла
func (p *HankookProcessor) GenerateFilename() string {
	return fmt.Sprintf("Hankook_Report_%s.xlsx", time.Now().Format("20060102_150405"))
}
