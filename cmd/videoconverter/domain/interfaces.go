package domain

import (
	"os"
)

type Storager interface {
	Videos() ([]Video, error)
	UpdatePropertyByID(id int64, value string) error
	InsertProperty(elementID int64, propertyID int64, value string) error
	QualityIDs() (*QualityProperty, error)
}
type Encoder interface {
	Convert(tmp string, filePath string, quality VQ) (string, error)
	CreatePreview(tmp, filePath string) (string, error)
}
type Clouder interface {
	DownloadFile(u string, f *os.File) error
	UploadFile(path string, f *os.File) (string, error)
	Delete(filepath string) error
}
