package service

import (
	"github.com/gocraft/dbr"
	"github.com/pkg/errors"
	"log"
	"videoconverter/domain"
)

type Storage struct {
	db   *dbr.Connection
	qids *domain.QualityProperty
}

func NewStorage(dbConn *dbr.Connection) *Storage {
	return &Storage{
		db: dbConn,
	}
}

func (s *Storage) Videos() ([]domain.Video, error) {
	var v []domain.Video

	_, err := s.db.NewSession(nil).
		SelectBySql(`
SELECT 
  p.IBLOCK_ELEMENT_ID id,
  p.ID AS id_original,
  p.VALUE AS link_original,
  p360.ID AS id_360,
  p360.VALUE AS link_360,
  p480.ID AS id_480,
  p480.VALUE AS link_480,
  p720.ID AS id_720,
  p720.VALUE AS link_720,
  p1080.ID AS id_1080,
  p1080.VALUE AS link_1080,
  preview.ID AS id_preview,
  preview.VALUE AS link_preview
FROM b_iblock
  JOIN b_iblock_property bip
    ON bip.IBLOCK_ID = b_iblock.ID AND bip.CODE = 'VIDEO_LINK'
  JOIN b_iblock_property bip360
    ON bip360.IBLOCK_ID = b_iblock.ID AND bip360.CODE = 'VIDEO_LINK_360p'
  JOIN b_iblock_property bip480
    ON bip480.IBLOCK_ID = b_iblock.ID AND bip480.CODE = 'VIDEO_LINK_480p'
  JOIN b_iblock_property bip720
    ON bip720.IBLOCK_ID = b_iblock.ID AND bip720.CODE = 'VIDEO_LINK_720p'
  JOIN b_iblock_property bip1080
    ON bip1080.IBLOCK_ID = b_iblock.ID AND bip1080.CODE = 'VIDEO_LINK_1080p'
  JOIN b_iblock_property bipPreview
    ON bipPreview.IBLOCK_ID = b_iblock.ID and bipPreview.CODE = 'VIDEO_LINK_PREVIEW'
  LEFT JOIN b_iblock_element_property AS p
    ON p.IBLOCK_PROPERTY_ID = bip.ID
  LEFT JOIN b_iblock_element_property AS p360
    ON p360.IBLOCK_PROPERTY_ID = bip360.ID AND p360.IBLOCK_ELEMENT_ID = p.IBLOCK_ELEMENT_ID
  LEFT JOIN b_iblock_element_property AS p480
    ON p480.IBLOCK_PROPERTY_ID = bip480.ID AND p480.IBLOCK_ELEMENT_ID = p.IBLOCK_ELEMENT_ID
  LEFT JOIN b_iblock_element_property AS p720
    ON p720.IBLOCK_PROPERTY_ID = bip720.ID AND p720.IBLOCK_ELEMENT_ID = p.IBLOCK_ELEMENT_ID
  LEFT JOIN b_iblock_element_property AS p1080
    ON p1080.IBLOCK_PROPERTY_ID = bip1080.ID AND p1080.IBLOCK_ELEMENT_ID = p.IBLOCK_ELEMENT_ID
  LEFT JOIN b_iblock_element_property AS preview
    ON preview.IBLOCK_PROPERTY_ID = bipPreview.ID AND preview.IBLOCK_ELEMENT_ID = p.IBLOCK_ELEMENT_ID
WHERE b_iblock.CODE = 'lessons' AND b_iblock.IBLOCK_TYPE_ID = 'content'
`).
		Load(&v)

	if err != nil {
		return []domain.Video{}, err
	}

	return v, nil
}

func (s *Storage) UpdatePropertyByID(id int64, value string) error {
	_, err := s.db.NewSession(nil).
		Update(
			"b_iblock_element_property",
		).
		Set(
			"VALUE", value,
		).
		Where(
			dbr.Eq(
				"ID", id,
			),
		).
		Exec()

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Storage) InsertProperty(elementID int64, propertyID int64, value string) error {
	var maxID int64

	session := s.db.NewSession(nil)
	_, err := session.Select("MAX(ID)").From("b_iblock_element_property").Load(&maxID)
	if err != nil {
		return errors.WithStack(err)
	}

	maxID++

	_, err = session.
		InsertInto(
			"b_iblock_element_property",
		).
		Columns(
			"ID", "IBLOCK_PROPERTY_ID", "IBLOCK_ELEMENT_ID", "VALUE",
		).
		Values(maxID, propertyID, elementID, value).
		Exec()

	if err != nil {
		return errors.WithStack(err)
	}

	return nil
}

func (s *Storage) qualityIDs() (*domain.QualityProperty, error) {
	var qp domain.QualityProperty

	session := s.db.NewSession(nil)

	_, err := session.SelectBySql(
		`
SELECT
bip1080.ID id_1080,
bip720.ID id_720,
bip480.ID id_480,
bip360.ID id_360,
bipPreview.ID id_preview
FROM b_iblock
  JOIN b_iblock_property bip360
    ON bip360.IBLOCK_ID = b_iblock.ID AND bip360.CODE = 'VIDEO_LINK_360p'
  JOIN b_iblock_property bip480
    ON bip480.IBLOCK_ID = b_iblock.ID AND bip480.CODE = 'VIDEO_LINK_480p'
  JOIN b_iblock_property bip720
    ON bip720.IBLOCK_ID = b_iblock.ID AND bip720.CODE = 'VIDEO_LINK_720p'
  JOIN b_iblock_property bip1080
    ON bip1080.IBLOCK_ID = b_iblock.ID AND bip1080.CODE = 'VIDEO_LINK_1080p'
  JOIN b_iblock_property bipPreview
    ON bipPreview.IBLOCK_ID = b_iblock.ID and bipPreview.CODE = 'VIDEO_LINK_PREVIEW'
WHERE b_iblock.CODE = 'lessons' AND b_iblock.IBLOCK_TYPE_ID = 'content'
`,
	).Load(&qp)

	if err != nil {
		return nil, errors.WithStack(err)
	}

	return &qp, nil
}

func (s *Storage) QualityIDs() (*domain.QualityProperty, error) {
	if s.qids != nil {
		return s.qids, nil
	}

	qp, err := s.qualityIDs()
	if err != nil {
		return nil, err
	}

	log.Printf("Поля форматов в БД: %#v", qp)

	return qp, nil
}
