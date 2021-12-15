package domain

import "github.com/gocraft/dbr"

// Video describe video entity with required db and business logic fields
type Video struct {
	ID int64 `db:"id"`

	IDOrig   dbr.NullInt64  `db:"id_original"`
	LinkOrig dbr.NullString `db:"link_original"`

	IDPreview   dbr.NullInt64  `db:"id_preview"`
	LinkPreview dbr.NullString `db:"link_preview"`

	ID1080   dbr.NullInt64  `db:"id_1080"`
	Link1080 dbr.NullString `db:"link_1080"`

	ID720   dbr.NullInt64  `db:"id_720"`
	Link720 dbr.NullString `db:"link_720"`

	ID480   dbr.NullInt64  `db:"id_480"`
	Link480 dbr.NullString `db:"link_480"`

	ID360   dbr.NullInt64  `db:"id_360"`
	Link360 dbr.NullString `db:"link_360"`

	FilenameOrig  string
	LocalPathOrig string
	CloudDir      string
}

// IsFull checks that a video has all required formats
func (v *Video) IsFull() bool {
	return v.Link1080.String != "" &&
		v.Link720.String != "" &&
		v.Link480.String != "" &&
		v.Link360.String != "" &&
		v.LinkPreview.String != ""
}

func (v *Video) IsHasAnyFormat() bool {
	return v.Link1080.Valid ||
		v.Link1080.Valid ||
		v.Link720.Valid ||
		v.Link480.Valid ||
		v.Link360.Valid ||
		v.LinkPreview.Valid
}

// QualityProperty describe property ids for every format of video in the database
type QualityProperty struct {
	ID1080    int64 `db:"id_1080"`
	ID720     int64 `db:"id_720"`
	ID480     int64 `db:"id_480"`
	ID360     int64 `db:"id_360"`
	IDPreview int64 `db:"id_preview"`
}
