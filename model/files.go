package model

type FileType string

const (
	ImageType    FileType = "image"
	VideoType    FileType = "video"
	DocumentType FileType = "document"
	AudioType    FileType = "audio"
)

type FileRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	File        string `json:"file"`
	Type        string `json:"type"`
	Entity      string `json:"entity"`
	EntityId    int    `json:"entity_id"`
}
