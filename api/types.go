package main

type Metadata struct {
	FileSize int64  `json:"fileSize"`
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
}

type ProgressMessage struct {
	Type     string  `json:"type"`
	Progress float64 `json:"progress"`
}
