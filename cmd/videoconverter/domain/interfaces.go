package domain

import (
	"os"
)

// Storager describe methods of storage Service
type Storager interface {
	Videos() ([]Video, error)
	UpdatePropertyByID(id int64, value string) error
	InsertProperty(elementID int64, propertyID int64, value string) error
	QualityIDs() (*QualityProperty, error)
}

// Encoder describe methods of storage Encode
type Encoder interface {
	Convert(tmp string, filePath string, quality VQ) (string, error)
	CreatePreview(tmp, filePath string) (string, error)
}

// Clouder describe methods of Cloud service
type Clouder interface {
	DownloadFile(u string, f *os.File) error
	UploadFile(path string, f *os.File) (string, error)
	Delete(filepath string) error
}
