package processors

import (
	"fmt"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"sending-stocks/models"
)

// IkonConfig конфигурация для отчета Ikon
type IkonConfig struct {
	CompanyName  string
	SummerGroups map[string][]string // ключ - колонка, значение - список брендов
	WinterGroups map[string][]string // ключ - колонка, значение - список брендов
}

// IkonProcessor обработчик для Ikon
type IkonProcessor struct {
	config *IkonConfig
}

// NewIkonProcessor создает новый процессор
func NewIkonProcessor(companyName string, summerGroups, winterGroups map[string][]string) *IkonProcessor {
	return &IkonProcessor{
		config: &IkonConfig{
			CompanyName:  companyName,
			SummerGroups: summerGroups,
			WinterGroups: winterGroups,
		},
	}
}

// CalculateSums вычисляет суммы по группам
func (p *IkonProcessor) CalculateSums(items []models.StockItem) (map[string]int, map[string]int, int) {
	summerSums := make(map[string]int)
	winterSums := make(map[string]int)
	totalSum := 0

	// Инициализируем суммы нулями
	for col := range p.config.SummerGroups {
		summerSums[col] = 0
	}
	for col := range p.config.WinterGroups {
		winterSums[col] = 0
	}

	// Суммируем по каждой позиции
	for _, item := range items {
		// Проверяем только позиции с количеством
		if item.Quantity <= 0 {
			continue
		}

		// Проверяем летние группы
		for col, brands := range p.config.SummerGroups {
			if p.itemInGroups(item, brands) && item.Season == "лето" {
				summerSums[col] += item.Quantity
				totalSum += item.Quantity
				break
			}
		}

		// Проверяем зимние группы
		for col, brands := range p.config.WinterGroups {
			if p.itemInGroups(item, brands) && item.Season == "зима" {
				winterSums[col] += item.Quantity
				totalSum += item.Quantity
				break
			}
		}
	}

	return summerSums, winterSums, totalSum
}

// itemInGroups проверяет, относится ли позиция к одной из групп брендов
func (p *IkonProcessor) itemInGroups(item models.StockItem, brands []string) bool {
	itemBrand := strings.ToLower(item.CleanBrand)
	for _, brand := range brands {
		brandLower := strings.ToLower(brand)
		if strings.Contains(itemBrand, brandLower) || strings.Contains(brandLower, itemBrand) {
			return true
		}
	}
	return false
}

// CreateReport создает Excel отчет
func (p *IkonProcessor) CreateReport(items []models.StockItem) (*excelize.File, error) {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println(err)
		}
	}()

	// Создаем лист
	index, _ := f.NewSheet("Sheet1")
	f.SetActiveSheet(index)

	// Заголовки
	headers := []string{
		"A", "B", "C", "D", "E", "F", "G", "H", "I", "J", "K", "L",
	}

	// Устанавливаем значения
	f.SetCellValue("Sheet1", "A1", p.config.CompanyName)
	f.SetCellValue("Sheet1", "B1", "Summer A")
	f.SetCellValue("Sheet1", "C1", "Summer B")
	f.SetCellValue("Sheet1", "D1", "Bars")
	f.SetCellValue("Sheet1", "E1", "Attar")
	f.SetCellValue("Sheet1", "F1", "SUMMER total")
	f.SetCellValue("Sheet1", "G1", "Winter A")
	f.SetCellValue("Sheet1", "H1", "Winter B")
	f.SetCellValue("Sheet1", "I1", "Attar")
	f.SetCellValue("Sheet1", "J1", "WINTER total")
	f.SetCellValue("Sheet1", "K1", "TOTAL")
	f.SetCellValue("Sheet1", "L1", "Все остатки клиента (по всем конкурентам и Нокиан в том числе)")

	// Вычисляем суммы
	summerSums, winterSums, totalSum := p.CalculateSums(items)

	// Заполняем суммы
	f.SetCellValue("Sheet1", "B2", summerSums["B"])
	f.SetCellValue("Sheet1", "C2", summerSums["C"])
	f.SetCellValue("Sheet1", "D2", summerSums["D"])
	f.SetCellValue("Sheet1", "E2", summerSums["E"])

	f.SetCellValue("Sheet1", "G2", winterSums["G"])
	f.SetCellValue("Sheet1", "H2", winterSums["H"])
	f.SetCellValue("Sheet1", "I2", winterSums["I"])

	// Формулы
	summerTotalFormula := "=SUM(B2:E2)"
	f.SetCellFormula("Sheet1", "F2", summerTotalFormula)

	winterTotalFormula := "=SUM(G2:I2)"
	f.SetCellFormula("Sheet1", "J2", winterTotalFormula)

	totalFormula := "=SUM(F2,J2)"
	f.SetCellFormula("Sheet1", "K2", totalFormula)

	// L2 - все остатки (можно оставить пустым или тоже формула)
	f.SetCellValue("Sheet1", "L2", totalSum)

	// Стили
	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 12,
		},
		Fill: excelize.Fill{
			Type:    "pattern",
			Color:   []string{"#E0E0E0"},
			Pattern: 1,
		},
	})
	f.SetCellStyle("Sheet1", "A1", "L1", style)

	// Устанавливаем ширину колонок
	for _, col := range headers {
		f.SetColWidth("Sheet1", col, col, 20)
	}

	return f, nil
}

// GenerateFilename генерирует имя файла
func (p *IkonProcessor) GenerateFilename() string {
	return fmt.Sprintf("Ikon_Report_%s.xlsx", time.Now().Format("20060102_150405"))
}
