package database

// StructuredItem представляет один элемент из structured_data
type StructuredItem struct {
	Title       string `json:"title"`
	TitleShort  string `json:"title_short"`
	Enhancement string `json:"enhancement"`
	Price       string `json:"price"`
	Package     bool   `json:"package"`
	Owner       string `json:"owner"`
	Count       string `json:"count"`
}

// OCRJSONResult представляет структуру JSON ответа
type OCRJSONResult struct {
	ImageFile  string `json:"image_file"`
	Processing struct {
		Enlargement         string `json:"enlargement"`
		Grayscale           bool   `json:"grayscale"`
		Denoising           string `json:"denoising"`
		ContrastEnhancement string `json:"contrast_enhancement"`
		Binarization        string `json:"binarization"`
		OCREngine           string `json:"ocr_engine"`
		OCRLanguages        string `json:"ocr_languages"`
		OCRMode             string `json:"ocr_mode"`
	} `json:"processing"`
	TextRecognition struct {
		Success        bool             `json:"success"`
		RawText        string           `json:"raw_text"`
		StructuredData []StructuredItem `json:"structured_data"`
		Confidence     string           `json:"confidence"`
	} `json:"text_recognition"`
}
