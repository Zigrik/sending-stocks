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

// CalculateSums вычисляет суммы по группам и общую сумму всех остатков
func (p *IkonProcessor) CalculateSums(items []models.StockItem) (map[string]int, map[string]int, int, int) {
	summerSums := make(map[string]int)
	winterSums := make(map[string]int)
	ikonGroupTotal := 0
	allBrandsTotal := 0

	// Инициализируем суммы нулями
	for col := range p.config.SummerGroups {
		summerSums[col] = 0
	}
	for col := range p.config.WinterGroups {
		winterSums[col] = 0
	}

	// Суммируем по каждой позиции
	for _, item := range items {
		// Проверяем только позиции с количеством (без проверки цены)
		if item.Quantity <= 0 {
			continue
		}

		// Добавляем в общую сумму всех брендов (ВСЕ остатки, независимо от наличия цены)
		allBrandsTotal += item.Quantity

		// Проверяем летние группы (только для отчета Ikon)
		for col, brands := range p.config.SummerGroups {
			if p.itemInGroups(item, brands) && item.Season == "лето" {
				summerSums[col] += item.Quantity
				ikonGroupTotal += item.Quantity
				break
			}
		}

		// Проверяем зимние группы (только для отчета Ikon)
		for col, brands := range p.config.WinterGroups {
			if p.itemInGroups(item, brands) && item.Season == "зима" {
				winterSums[col] += item.Quantity
				ikonGroupTotal += item.Quantity
				break
			}
		}
	}

	return summerSums, winterSums, ikonGroupTotal, allBrandsTotal
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

	// Удаляем лист по умолчанию
	f.DeleteSheet("Sheet1")

	// Заголовки
	f.SetCellValue("Sheet1", "A1", "Клиент")
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

	// Название компании в A2
	f.SetCellValue("Sheet1", "A2", p.config.CompanyName)

	// Вычисляем суммы
	summerSums, winterSums, _, allBrandsTotal := p.CalculateSums(items)

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

	// L2 - все остатки по всем брендам (сумма всех Quantity)
	f.SetCellValue("Sheet1", "L2", allBrandsTotal)

	// Стили для заголовков
	headerStyle, _ := f.NewStyle(&excelize.Style{
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
	f.SetCellStyle("Sheet1", "A1", "L1", headerStyle)

	// Стиль для компании
	companyStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{
			Bold: true,
			Size: 11,
		},
	})
	f.SetCellStyle("Sheet1", "A2", "A2", companyStyle)

	// Стиль для чисел (выравнивание по центру)
	numberStyle, _ := f.NewStyle(&excelize.Style{
		Alignment: &excelize.Alignment{
			Horizontal: "center",
		},
	})
	f.SetCellStyle("Sheet1", "B2", "L2", numberStyle)

	// Устанавливаем ширину колонок
	colWidths := map[string]float64{
		"A": 20, "B": 12, "C": 12, "D": 10, "E": 10,
		"F": 15, "G": 12, "H": 12, "I": 10, "J": 15,
		"K": 10, "L": 40,
	}
	for col, width := range colWidths {
		f.SetColWidth("Sheet1", col, col, width)
	}

	return f, nil
}

// GenerateFilename генерирует имя файла
func (p *IkonProcessor) GenerateFilename() string {
	return fmt.Sprintf("Ikon_Report_%s.xlsx", time.Now().Format("20060102_150405"))
}
