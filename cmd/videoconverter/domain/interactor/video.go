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
	"videoconverter/domain"
)

type VideoCase struct {
	env     string
	tmp     string
	ch      map[int]chan int
	db      domain.Storager
	cloud   domain.Clouder
	encoder domain.Encoder
}

func NewVideoCase(ch map[int]chan int, env string, tmp string, db domain.Storager, cloud domain.Clouder, encoder domain.Encoder) *VideoCase {
	return &VideoCase{
		env:     env,
		ch:      ch,
		tmp:     tmp,
		db:      db,
		cloud:   cloud,
		encoder: encoder,
	}
}

// Start starts main logic
func (vc *VideoCase) Start(deadline time.Time) {
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
			log.Println("Time is over.")
			break
		}

		v := video

		if v.IsFull() {
			log.Printf("Видео %d имеет все форматы, пропускаю", v.ID)
			continue
		}

		if v.LinkOrig.String == "" {
			log.Printf("Видео %d имеет пустую ссылку на оригинал, пропускаю", v.ID)
			continue
		}

		escapedURL, err := url.PathUnescape(v.LinkOrig.String)
		if err != nil {
			log.Printf("Не удалось экранировать URL %s\nПропускаю обработку", v.LinkOrig.String)
			return
		}

		v.LinkOrig.String = escapedURL

		wg.Add(1)
		go vc.ProcessingVideo(&wg, &v)
	}

	wg.Wait()

	vc.ch[domain.ChDone] <- 1
}

func (vc *VideoCase) ProcessingVideo(g *sync.WaitGroup, v *domain.Video) {
	log.Println("Начинаю обработку видео с ID", v.ID)
	defer g.Done()

	cURL, err := url.Parse(v.LinkOrig.String)
	if err != nil {
		log.Printf("Ссылка на оригинал не является валидным URL : %s", v.LinkOrig.String)
		return
	}

	cloudDir, cloudFile := path.Split(cURL.Path)
	v.CloudDir = strings.ReplaceAll(cloudDir, "/synergy/", "")
	v.FilenameOrig = domain.FormatFileName(cloudFile)

	f, err := os.Create(vc.tmp + "/" + v.FilenameOrig)
	if err != nil {
		log.Printf("Create a temp file: %+v", err)
		return
	}

	defer func() {
		f.Close()

		err = os.Remove(f.Name())
		if err != nil {
			log.Printf("Remove file %s: %v", f.Name(), err)
		}
	}()

	log.Printf("Загружаю оригинал видео ID %d по ссылке %s", v.ID, v.LinkOrig.String)
	err = vc.cloud.DownloadFile(v.LinkOrig.String, f)
	if err != nil {
		log.Printf("DownloadFile file: %+v", err)
		return
	}

	v.LocalPathOrig = f.Name()

	var wg sync.WaitGroup
	wg.Add(5)

	go vc.p1080(&wg, v)
	go vc.p720(&wg, v)
	go vc.p480(&wg, v)
	go vc.p360(&wg, v)
	go vc.pPreview(&wg, v)

	wg.Wait()

	if v.IsFull() && vc.env != "prod" {
		log.Printf("Видео %d полностью обработано, удаляю оригинал", v.ID)

		vc.cloud.Delete(v.CloudDir + cloudFile)
	}
}

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
	defer os.Remove(newV)

	vc.ch[domain.ChConverted] <- 1

	f, err := os.Open(newV)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, vName := path.Split(f.Name())
	cloudPath := fmt.Sprintf("%s%s", v.CloudDir, vName)

	log.Printf("Загружаю на облако файл %s", f.Name())
	u, err := vc.cloud.UploadFile(cloudPath, f)
	if err != nil {
		vc.ch[domain.ChNotUploaded] <- 1
		return "", err
	}

	vc.ch[domain.ChUploaded] <- 1

	return "https://" + u, nil
}

func (vc *VideoCase) p1080(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.Q1080)
	if err != nil {
		log.Printf("Ошибка обработки видео %d: %+v", v.ID, err)
		return
	}

	qp, err := vc.db.QualityIDs()
	if err != nil {
		log.Printf("Ошибка получения ID форматов из БД: %+v", err)
		return
	}

	v.Link1080.String = u
	if v.ID1080.Valid {
		err = vc.db.UpdatePropertyByID(v.ID1080.Int64, u)
	} else {
		err = vc.db.InsertProperty(v.ID, qp.ID1080, u)
	}

	if err != nil {
		log.Printf("Ошибка обновления поля %d в БД: %+v", v.ID1080.Int64, err)
		return
	}
}

func (vc *VideoCase) p720(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.Q720)
	if err != nil {
		log.Printf("Ошибка обработки видео %d: %+v", v.ID, err)
		return
	}

	qp, err := vc.db.QualityIDs()
	if err != nil {
		log.Printf("Ошибка получения ID форматов из БД: %+v", err)
		return
	}

	v.Link720.String = u
	if v.ID720.Valid {
		err = vc.db.UpdatePropertyByID(v.ID720.Int64, u)
	} else {
		err = vc.db.InsertProperty(v.ID, qp.ID720, u)
	}

	if err != nil {
		log.Printf("Ошибка обновления поля %d в БД: %+v", v.ID720.Int64, err)
		return
	}
}

func (vc *VideoCase) p480(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.Q480)
	if err != nil {
		log.Printf("Ошибка обработки видео %d: %+v", v.ID, err)
		return
	}

	qp, err := vc.db.QualityIDs()
	if err != nil {
		log.Printf("Ошибка получения ID форматов из БД: %+v", err)
		return
	}

	v.Link480.String = u
	if v.ID480.Valid {
		err = vc.db.UpdatePropertyByID(v.ID480.Int64, u)
	} else {
		err = vc.db.InsertProperty(v.ID, qp.ID480, u)
	}

	if err != nil {
		log.Printf("Ошибка обновления поля %d в БД: %+v", v.ID480.Int64, err)
		return
	}
}

func (vc *VideoCase) p360(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.Q360)
	if err != nil {
		log.Printf("Ошибка обработки видео %d: %+v", v.ID, err)
		return
	}

	qp, err := vc.db.QualityIDs()
	if err != nil {
		log.Printf("Ошибка получения ID форматов из БД: %+v", err)
		return
	}

	v.Link360.String = u
	if v.ID360.Valid {
		err = vc.db.UpdatePropertyByID(v.ID360.Int64, u)
	} else {
		err = vc.db.InsertProperty(v.ID, qp.ID360, u)
	}

	if err != nil {
		log.Printf("Ошибка обновления поля %d в БД: %+v", v.ID360.Int64, err)
		return
	}
}

func (vc *VideoCase) pPreview(wg *sync.WaitGroup, v *domain.Video) {
	defer wg.Done()

	u, err := vc.process(v, domain.QPreview)
	if err != nil {
		log.Printf("Ошибка обработки видео %d: %+v", v.ID, err)
		return
	}

	qp, err := vc.db.QualityIDs()
	if err != nil {
		log.Printf("Ошибка получения ID форматов из БД: %+v", err)
		return
	}

	v.LinkPreview.String = u

	if v.IDPreview.Valid {
		err = vc.db.UpdatePropertyByID(v.IDPreview.Int64, u)
	} else {
		err = vc.db.InsertProperty(v.ID, qp.IDPreview, u)
	}

	if err != nil {
		log.Printf("Ошибка обновления поля %d в БД: %+v", v.IDPreview.Int64, err)
		return
	}
}
