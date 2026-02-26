package models

// StockItem данные из строки файла
type StockItem struct {
	RowNum          int     `json:"row_num"`          // номер строки
	Name            string  `json:"name"`             // столбец A - наименование
	Brand           string  `json:"brand"`            // столбец B - бренд (с сезоном)
	Code1C          string  `json:"code_1c"`          // столбец F - код в 1С
	ManufacturerSKU string  `json:"manufacturer_sku"` // столбец G - код производителя
	TireSize        string  `json:"tire_size"`        // столбец H - типоразмер
	Quantity        int     `json:"quantity"`         // столбец I - остаток
	Price           float64 `json:"price"`            // столбец J - цена

	// Обработанные поля
	CleanBrand string `json:"clean_brand"` // бренд без сезона и *
	Season     string `json:"season"`      // лето/зима
	IsYearOld  bool   `json:"is_year_old"` // есть ли "год" в названии
	IsPirelli  bool   `json:"is_pirelli"`  // относится к Pirelli/Formula
}

// ProcessedFile результат обработки
type ProcessedFile struct {
	Filename     string      `json:"filename"`
	OriginalFile string      `json:"original_file"`
	UploadDate   string      `json:"upload_date"`
	PirelliItems []StockItem `json:"pirelli_items"`
	AllItems     []StockItem `json:"all_items"`
	Stats        Stats       `json:"stats"`
}

// Stats статистика обработки
type Stats struct {
	TotalRows    int      `json:"total_rows"`
	ValidRows    int      `json:"valid_rows"`
	InvalidRows  int      `json:"invalid_rows"`
	PirelliCount int      `json:"pirelli_count"`
	Errors       []string `json:"errors,omitempty"`
}

// UploadResult результат загрузки
type UploadResult struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PirelliResponse ответ от API Pirelli
type PirelliResponse struct {
	Status  bool   `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    []struct {
		DateTime     string `json:"datetime"`
		OriginalName string `json:"original_name"`
	} `json:"data"`
}
