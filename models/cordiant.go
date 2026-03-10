package models

// CordiantItem представляет позицию для отчета Cordiant
type CordiantItem struct {
	RowNum     int    `json:"row_num"`
	Code       string `json:"code"`        // код товара (ManufacturerSKU)
	TireSize   string `json:"tire_size"`   // типоразмер
	Brand      string `json:"brand"`       // наименование товара
	Quantity   int    `json:"quantity"`    // количество
	CleanBrand string `json:"clean_brand"` // очищенный бренд
}

// CordiantRequest запрос к API Cordiant
type CordiantRequest struct {
	Year   string `json:"year"`
	Month  string `json:"month"`
	Token  string `json:"token"`
	Action string `json:"action"`
	File   string `json:"file"` // файл в base64
}

// CordiantResponseData структура для успешного ответа с data объектом
type CordiantResponseData struct {
	Status              string   `json:"status"`
	Message             string   `json:"message"`
	Title               string   `json:"title"`
	Content             string   `json:"content"`
	Warnings            []string `json:"warnings"`
	Function            string   `json:"function"`
	IsHavePrevRecords   bool     `json:"isHavePrevRecords"`
	Warehouses          int      `json:"warehouses"`
	WarehousesPositions int      `json:"warehousesPositions"`
	ErrorFileStrings    []int    `json:"errorFileStrings"`
}

// CordiantResponse ответ от API Cordiant
type CordiantResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
