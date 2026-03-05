package models

// CordiantItem представляет позицию для отчета Cordiant
type CordiantItem struct {
	RowNum     int    `json:"row_num"`
	Code       string `json:"code"`        // код товара
	TireSize   string `json:"tire_size"`   // типоразмер
	Brand      string `json:"brand"`       // бренд
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

// CordiantResponse ответ от API Cordiant
type CordiantResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}
