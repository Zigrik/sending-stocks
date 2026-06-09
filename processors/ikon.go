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
	CompanyName   string
	SummerGroups  map[string][]string // ключ - колонка, значение - список брендов
	WinterGroups  map[string][]string // ключ - колонка, значение - список брендов
	SummerExclude []string            // бренды, исключаемые из SUMMER C total
	WinterExclude []string            // бренды, исключаемые из WINTER C total
}

// IkonProcessor обработчик для Ikon
type IkonProcessor struct {
	config *IkonConfig
}

// NewIkonProcessor создает новый процессор
func NewIkonProcessor(companyName string, summerGroups, winterGroups map[string][]string, summerExclude, winterExclude []string) *IkonProcessor {
	return &IkonProcessor{
		config: &IkonConfig{
			CompanyName:   companyName,
			SummerGroups:  summerGroups,
			WinterGroups:  winterGroups,
			SummerExclude: summerExclude,
			WinterExclude: winterExclude,
		},
	}
}

// isExcludedBrand проверяет, нужно ли исключить бренд из общей суммы
func (p *IkonProcessor) isExcludedBrand(brand string, excludeList []string) bool {
	brandLower := strings.ToLower(brand)
	for _, exclude := range excludeList {
		excludeLower := strings.ToLower(exclude)
		if strings.Contains(brandLower, excludeLower) || strings.Contains(excludeLower, brandLower) {
			return true
		}
	}
	return false
}

// CalculateSums вычисляет суммы по группам и общую сумму всех остатков
func (p *IkonProcessor) CalculateSums(items []models.StockItem) (map[string]int, map[string]int, int, int, int) {
	summerSums := make(map[string]int)
	winterSums := make(map[string]int)
	allBrandsTotal := 0
	summerCTotal := 0
	winterCTotal := 0

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

		// Добавляем в общую сумму всех брендов (ВСЕ остатки)
		allBrandsTotal += item.Quantity

		// SUMMER C total - все летние остатки, кроме исключенных брендов
		if item.Season == "лето" && !p.isExcludedBrand(item.CleanBrand, p.config.SummerExclude) {
			summerCTotal += item.Quantity
		}

		// WINTER C total - все зимние остатки, кроме исключенных брендов
		if item.Season == "зима" && !p.isExcludedBrand(item.CleanBrand, p.config.WinterExclude) {
			winterCTotal += item.Quantity
		}

		// Проверяем летние группы
		for col, brands := range p.config.SummerGroups {
			if p.itemInGroups(item, brands) && item.Season == "лето" {
				summerSums[col] += item.Quantity
				break
			}
		}

		// Проверяем зимние группы
		for col, brands := range p.config.WinterGroups {
			if p.itemInGroups(item, brands) && item.Season == "зима" {
				winterSums[col] += item.Quantity
				break
			}
		}
	}

	return summerSums, winterSums, allBrandsTotal, summerCTotal, winterCTotal
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
	f.SetCellValue("Sheet1", "G1", "SUMMER C total")
	f.SetCellValue("Sheet1", "H1", "Winter A")
	f.SetCellValue("Sheet1", "I1", "Winter B")
	f.SetCellValue("Sheet1", "J1", "Attar")
	f.SetCellValue("Sheet1", "K1", "WINTER total")
	f.SetCellValue("Sheet1", "L1", "WINTER C total")
	f.SetCellValue("Sheet1", "M1", "TOTAL")
	f.SetCellValue("Sheet1", "N1", "Все остатки клиента (по всем конкурентам и Нокиан в том числе)")

	// Название компании в A2
	f.SetCellValue("Sheet1", "A2", p.config.CompanyName)

	// Вычисляем суммы
	summerSums, winterSums, allBrandsTotal, summerCTotal, winterCTotal := p.CalculateSums(items)

	// Заполняем суммы по группам
	f.SetCellValue("Sheet1", "B2", summerSums["B"])
	f.SetCellValue("Sheet1", "C2", summerSums["C"])
	f.SetCellValue("Sheet1", "D2", summerSums["D"])
	f.SetCellValue("Sheet1", "E2", summerSums["E"])

	f.SetCellValue("Sheet1", "H2", winterSums["H"])
	f.SetCellValue("Sheet1", "I2", winterSums["I"])
	f.SetCellValue("Sheet1", "J2", winterSums["J"])

	// Заполняем новые колонки
	f.SetCellValue("Sheet1", "G2", summerCTotal)
	f.SetCellValue("Sheet1", "L2", winterCTotal)

	// Формулы
	summerTotalFormula := "=SUM(B2:E2)"
	f.SetCellFormula("Sheet1", "F2", summerTotalFormula)

	winterTotalFormula := "=SUM(H2:J2)"
	f.SetCellFormula("Sheet1", "K2", winterTotalFormula)

	totalFormula := "=SUM(F2,K2)"
	f.SetCellFormula("Sheet1", "M2", totalFormula)

	// N2 - все остатки по всем брендам
	f.SetCellValue("Sheet1", "N2", allBrandsTotal)

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
	f.SetCellStyle("Sheet1", "A1", "N1", headerStyle)

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
	f.SetCellStyle("Sheet1", "B2", "N2", numberStyle)

	// Устанавливаем ширину колонок
	colWidths := map[string]float64{
		"A": 20, "B": 12, "C": 12, "D": 10, "E": 10,
		"F": 15, "G": 18,
		"H": 12, "I": 12, "J": 10, "K": 15, "L": 18,
		"M": 10, "N": 40,
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
