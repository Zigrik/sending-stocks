package processors

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"sending-stocks/models"
)

// PirelliExcelProcessor обработчик для Excel отчета Pirelli
type PirelliExcelProcessor struct {
	CustomerCode  string
	PirelliBrands []string
}

// NewPirelliExcelProcessor создает новый процессор
func NewPirelliExcelProcessor(customerCode string, pirelliBrands []string) *PirelliExcelProcessor {
	return &PirelliExcelProcessor{
		CustomerCode:  customerCode,
		PirelliBrands: pirelliBrands,
	}
}

// isPirelliBrand проверяет, относится ли бренд к Pirelli/Formula
func (p *PirelliExcelProcessor) isPirelliBrand(brand string) bool {
	brandLower := strings.ToLower(brand)
	for _, pb := range p.PirelliBrands {
		pbLower := strings.ToLower(pb)
		if strings.Contains(brandLower, pbLower) || strings.Contains(pbLower, brandLower) {
			return true
		}
	}
	return false
}

// CreateExcelReport создает Excel отчет для Pirelli (включает все позиции, даже без кода производителя)
func (p *PirelliExcelProcessor) CreateExcelReport(items []models.StockItem) (*excelize.File, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Создаем лист
	index, _ := f.NewSheet("Pirelli Report")
	f.SetActiveSheet(index)

	// Удаляем лист по умолчанию
	f.DeleteSheet("Sheet1")

	// Заголовки
	headers := []string{
		"Season", "Brand", "IP code", "Size", "Tread pattern", "Quantity",
	}

	for i, header := range headers {
		col := string(rune('A' + i))
		f.SetCellValue("Pirelli Report", fmt.Sprintf("%s1", col), header)
	}

	// Фильтруем позиции Pirelli/Formula (включаем все, даже без кода производителя)
	pirelliItems := make([]models.StockItem, 0)
	for _, item := range items {
		if p.isPirelliBrand(item.CleanBrand) && item.Quantity > 0 {
			pirelliItems = append(pirelliItems, item)
		}
	}

	// Заполняем данные
	row := 2
	for _, item := range pirelliItems {
		// Season
		season := ""
		if item.Season == "лето" {
			season = "Summer"
		} else if item.Season == "зима" {
			season = "Winter"
		}
		f.SetCellValue("Pirelli Report", fmt.Sprintf("A%d", row), season)

		// Brand
		f.SetCellValue("Pirelli Report", fmt.Sprintf("B%d", row), item.CleanBrand)

		// IP code (код 1С)
		f.SetCellValue("Pirelli Report", fmt.Sprintf("C%d", row), item.Code1C)

		// Size (типоразмер)
		f.SetCellValue("Pirelli Report", fmt.Sprintf("D%d", row), item.TireSize)

		// Tread pattern (наименование)
		f.SetCellValue("Pirelli Report", fmt.Sprintf("E%d", row), item.Name)

		// Quantity
		f.SetCellValue("Pirelli Report", fmt.Sprintf("F%d", row), item.Quantity)

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
			Color:   []string{"#FFB6C1"}, // Светло-красный для Pirelli
			Pattern: 1,
		},
	})
	f.SetCellStyle("Pirelli Report", "A1", "F1", headerStyle)

	// Устанавливаем ширину колонок
	colWidths := map[string]float64{
		"A": 10, "B": 15, "C": 15, "D": 15, "E": 50, "F": 10,
	}
	for col, width := range colWidths {
		f.SetColWidth("Pirelli Report", col, col, width)
	}

	return f, nil
}

// GenerateFilename генерирует имя файла
func (p *PirelliExcelProcessor) GenerateFilename() string {
	return fmt.Sprintf("Pirelli_Report_%s.xlsx", time.Now().Format("20060102_150405"))
}
