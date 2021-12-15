package interactor

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"
	"videoconverter/bootstrap"
	"videoconverter/domain"
)

// VideoCase describe a video interactor
// used for start vide use cases
type VideoCase struct {
	env         string
	tmp         string
	rmOrig      bool
	skipNotFull bool
	ch          map[int]chan int
	db          domain.Storager
	cloud       domain.Clouder
	encoder     domain.Encoder

	l *bootstrap.Logger
}

// NewVideoCase returns a ready for use instance ofr VideoCase
func NewVideoCase(ch map[int]chan int, env string, tmp string, isRmOrig bool, isSkipNotFull bool, db domain.Storager, cloud domain.Clouder, encoder domain.Encoder, l *bootstrap.Logger) *VideoCase {
	return &VideoCase{
		env:         env,
		ch:          ch,
		rmOrig:      isRmOrig,
		skipNotFull: isSkipNotFull,
		tmp:         tmp,
		db:          db,
		cloud:       cloud,
		encoder:     encoder,
		l:           l,
	}
}

// Start starts the processing all videos case
func (vc *VideoCase) Start(deadline time.Time) {
	defer func() {
		vc.ch[domain.ChDone] <- 1
	}()

	videos, err := vc.db.Videos()
	if err != nil {
		log.Println("Get videos:", err)
		return
	}

	vc.ch[domain.ChAll] <- len(videos)

	var wg sync.WaitGroup

	for _, video := range videos {
		now := time.Now().UTC()

		if now.After(deadline) {
			vc.l.D(fmt.Sprintln("Time is over."))
			break
		}

		v := video

		if v.IsFull() {
			vc.l.D(fmt.Sprintf("Видео %d имеет все форматы, пропускаю", v.ID))
			continue
		}

		if vc.skipNotFull && v.IsHasAnyFormat() {
			vc.l.D(fmt.Sprintf("Проверьте видео %d, оно имеет один или несколько форматов, пропускаю", v.ID))
			continue
		}

		if v.LinkOrig.String == "" {
			vc.l.D(fmt.Sprintf("Видео %d имеет пустую ссылку на оригинал, пропускаю", v.ID))
			continue
		}

		escapedURL, err := url.PathUnescape(v.LinkOrig.String)
		if err != nil {
			vc.l.E(fmt.Sprintf("Не удалось экранировать URL %s\nПропускаю обработку", v.LinkOrig.String))

			continue
		}

		cURL, err := url.Parse(v.LinkOrig.String)
		if err != nil {
			vc.l.E(fmt.Sprintf("Ссылка на оригинал не является валидным URL : %s", v.LinkOrig.String))
			continue
		}

		cloudDir, cloudFile := path.Split(cURL.Path)
		v.CloudDir = strings.ReplaceAll(cloudDir, "/synergy/", "")
		v.FilenameOrig = domain.FormatFileName(cloudFile)

		f, err := os.Create(vc.tmp + "/" + v.FilenameOrig)
		if err != nil {
			vc.l.E(fmt.Sprintf("Create a temp file: %+v", err))
			continue
		}

		vc.l.D(fmt.Sprintf("Загружаю оригинал видео ID %d по ссылке %s", v.ID, v.LinkOrig.String))

		err = vc.cloud.DownloadFile(v.LinkOrig.String, f)
		if err != nil {
			vc.l.E(fmt.Sprintf(" Ошибка загрузки ориганала ID %d по ссылке %s: %+v", v.ID, v.LinkOrig.String, err))
			f.Close()

			continue
		}

		f.Close()

		v.LocalPathOrig = f.Name()

		v.LinkOrig.String = escapedURL

		wg.Add(1)
		go vc.ProcessingVideo(&wg, &v, cloudFile)
	}

	wg.Wait()
}

// ProcessingVideo start the processing of one video,
// delete original after processing
func (vc *VideoCase) ProcessingVideo(g *sync.WaitGroup, v *domain.Video, cloudFile string) {
	log.Println("Начинаю обработку видео с ID", v.ID)

	defer func() {
		err := os.Remove(v.LocalPathOrig)
		if err != nil {
			fmt.Sprintf("Remove file %s: %v", v.LocalPathOrig, err)
		}

		g.Done()
	}()

	var wg sync.WaitGroup

	switch {
	case v.Link1080.String == "":
		wg.Add(1)
		go vc.p1080(&wg, v)
		fallthrough

	case v.Link720.String == "":
		wg.Add(1)
		go vc.p720(&wg, v)
		fallthrough

	case v.Link480.String == "":
		wg.Add(1)
		go vc.p480(&wg, v)
		fallthrough

	case v.Link360.String == "":
		wg.Add(1)
		go vc.p360(&wg, v)
		fallthrough

	case v.LinkPreview.String == "":
		wg.Add(1)
		go vc.pPreview(&wg, v)
	}

	wg.Wait()

	if v.IsFull() && vc.rmOrig {
		vc.l.D(fmt.Sprintf("Видео %s полностью обработано, удаляю оригинал", v.FilenameOrig))

		if err := vc.cloud.Delete(v.CloudDir + cloudFile); err != nil {
			vc.l.E(fmt.Sprintf("Ошибка удаления оригинала из облака %s\n%v", cloudFile, err))

			return
		}

		if err := vc.db.UpdatePropertyByID(v.IDOrig.Int64, ""); err != nil {
			vc.l.E(fmt.Sprintf("Ошибка очистки ссылки на оригинал в БД %s\n%v", cloudFile, err))
		}
	}
}

// process converts a video to required format and uploads to the cloud
func (vc *VideoCase) process(v *domain.Video, q domain.VQ) (string, error) {
	var newV string
	var err error

	if q == domain.QPreview {
		newV, err = vc.encoder.CreatePreview(vc.tmp, v.LocalPathOrig)
	} else {
		newV, err = vc.encoder.Convert(vc.tmp, v.LocalPathOrig, q)
	}

	if err != nil {
		vc.ch[domain.ChNotConverted] <- 1

		return "", err
	}

	defer func() {
		vc.l.D(fmt.Sprintf("Remove file: %s", newV))

		if err := os.Remove(newV); err != nil {
			vc.l.E(fmt.Sprintf("Error remove file: %s", newV))
		}
	}()

	vc.ch[domain.ChConverted] <- 1

	f, err := os.Open(newV)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, vName := path.Split(f.Name())
	cloudPath := fmt.Sprintf("%s%s", v.CloudDir, vName)

	vc.l.D(fmt.Sprintf("Загружаю на облако файл %s", f.Name()))
	u, err := vc.cloud.UploadFile(cloudPath, f)
	if err != nil {
		vc.ch[domain.ChNotUploaded] <- 1
		return "", err
	}

	vc.l.D(fmt.Sprintf("Успешно загрузили файл %s", f.Name()))

	vc.ch[domain.ChUploaded] <- 1

	eu, err := url.Parse(u)
	if err != nil {
		return "", err
	}

	return eu.String(), nil
}

// p1080 start process method and update video data in the database
func (vc *VideoCase) p1080(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.Q1080)
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка обработки видео %d: %+v", v.ID, err))
		return
	}

	vc.l.D("Ссылка на облако", u)

	qp, err := vc.db.QualityIDs()
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка получения ID форматов из БД: %+v", err))
		vc.ch[domain.ChDone] <- 1
		return
	}

	v.Link1080.String = u

	if v.ID1080.Valid {
		if err := vc.db.UpdatePropertyByID(v.ID1080.Int64, u); err != nil {
			vc.l.E(fmt.Sprintf("Ошибка обновления поля %d для видео %d в БД: %+v", v.ID1080.Int64, v.ID, err))
			vc.ch[domain.ChDone] <- 1
		}

		return
	}

	if err := vc.db.InsertProperty(v.ID, qp.ID1080, u); err != nil {
		vc.l.E(fmt.Sprintf("Ошибка  добавления поля 1080 для видео %d в БД: %+v", v.ID, err))
		vc.ch[domain.ChDone] <- 1
	}
}

// p480 start process method and update video data in the database
func (vc *VideoCase) p480(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.Q480)
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка обработки видео %d: %+v", v.ID, err))
		return
	}

	vc.l.D("Ссылка на облако", u)

	qp, err := vc.db.QualityIDs()
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка получения ID форматов из БД: %+v", err))
		vc.ch[domain.ChDone] <- 1
		return
	}

	v.Link480.String = u
	if v.ID480.Valid {
		if err := vc.db.UpdatePropertyByID(v.ID480.Int64, u); err != nil {
			vc.l.E(fmt.Sprintf("Ошибка обновления поля %d для видео %d в БД: %+v", v.ID480.Int64, v.ID, err))
			vc.ch[domain.ChDone] <- 1
		}

		return
	}

	if err := vc.db.InsertProperty(v.ID, qp.ID480, u); err != nil {
		vc.l.E(fmt.Sprintf("Ошибка добавления поля 480p видео %d в БД: %+v", v.ID, err))
		vc.ch[domain.ChDone] <- 1
	}
}

// p720 start process method and update video data in the database
func (vc *VideoCase) p720(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.Q720)
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка обработки видео %d: %+v", v.ID, err))
		return
	}

	vc.l.D("Ссылка на облако", u)

	qp, err := vc.db.QualityIDs()
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка получения ID форматов из БД: %+v", err))
		vc.ch[domain.ChDone] <- 1
		return
	}

	v.Link720.String = u

	if v.ID720.Valid {
		if err := vc.db.UpdatePropertyByID(v.ID720.Int64, u); err != nil {
			vc.l.E(fmt.Sprintf("Ошибка обновления поля %d у видео %d в БД: %+v", v.ID720.Int64, v.ID, err))
			vc.ch[domain.ChDone] <- 1

		}

		return
	}

	if err := vc.db.InsertProperty(v.ID, qp.ID720, u); err != nil {
		vc.l.E(fmt.Sprintf("Ошибка обновления поля %d в БД: %+v", v.ID720.Int64, err))
		vc.ch[domain.ChDone] <- 1
	}
}

// p360 start process method and update video data in the database
func (vc *VideoCase) p360(wg *sync.WaitGroup,
	v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.Q360)
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка обработки видео %d: %+v", v.ID, err))
		return
	}

	vc.l.D("Ссылка на облако", u)

	qp, err := vc.db.QualityIDs()
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка получения ID форматов из БД: %+v", err))
		vc.ch[domain.ChDone] <- 1

		return
	}

	v.Link360.String = u

	if v.ID360.Valid {
		if err = vc.db.UpdatePropertyByID(v.ID360.Int64, u); err != nil {
			vc.l.E(fmt.Sprintf("Ошибка обновления поля %d в БД у видео %d: %+v", v.ID360.Int64, v.ID, err))
			vc.ch[domain.ChDone] <- 1

			return
		}
	}

	if err = vc.db.InsertProperty(v.ID, qp.ID360, u); err != nil {
		vc.l.E(fmt.Sprintf("Ошибка добавления поля для формата 360 у видео %d в БД: %+v", v.ID, err))
		vc.ch[domain.ChDone] <- 1

		return
	}
}

// pPreview start process method and update video data in the database
func (vc *VideoCase) pPreview(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.QPreview)
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка обработки видео %d: %+v", v.ID, err))
		return
	}

	vc.l.D("Ссылка на облако", u)

	qp, err := vc.db.QualityIDs()
	if err != nil {
		vc.l.E(fmt.Sprintf("Ошибка получения ID форматов из БД: %+v", err))
		vc.ch[domain.ChDone] <- 1

		return
	}

	v.LinkPreview.String = u

	if v.IDPreview.Valid {
		if err := vc.db.UpdatePropertyByID(v.IDPreview.Int64, u); err != nil {
			vc.l.E(fmt.Sprintf("Ошибка обновления поля %d в БД: %+v", v.IDPreview.Int64, err))
			vc.ch[domain.ChDone] <- 1
		}

		return
	}

	if err := vc.db.InsertProperty(v.ID, qp.IDPreview, u); err != nil {
		vc.l.E(fmt.Sprintf("Ошибка обновления поля %d в БД: %+v", v.IDPreview.Int64, err))
		vc.ch[domain.ChDone] <- 1
	}
}
