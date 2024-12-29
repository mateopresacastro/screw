package main

type Metadata struct {
	FileSize int64  `json:"fileSize"`
	FileName string `json:"fileName"`
	MimeType string `json:"mimeType"`
}

type CompletionMessage struct {
	TotalSize int64 `json:"totalSize"`
}

type ProgressMessage struct {
	Type         string  `json:"type"`
	Progress     float64 `json:"progress"`
	ReceivedSize int64   `json:"receivedSize"`
	TotalSize    int64   `json:"totalSize"`
}
