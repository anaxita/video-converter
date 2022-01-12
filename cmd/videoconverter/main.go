package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"
	"videoconverter/bootstrap"
	"videoconverter/domain"
	"videoconverter/domain/interactor"
	"videoconverter/service"

	_ "embed"
	_ "github.com/go-sql-driver/mysql"
)

//go:embed ffmpeg-4-2-4
var ffmpeg []byte

func main() {
	pathToConfig := flag.String("c", "./.env", "path to .env config")
	flag.Parse()
	now := time.Now()

	shutdown := make(chan os.Signal, 0)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP)
	defer close(shutdown)

	channels := map[int]chan int{
		domain.ChDone:         make(chan int),
		domain.ChAll:          make(chan int),
		domain.ChConverted:    make(chan int),
		domain.ChNotConverted: make(chan int),
		domain.ChUploaded:     make(chan int),
	}

	// configs
	c, err := bootstrap.New(*pathToConfig)
	if err != nil {
		log.Fatalln("Config load:", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Hour*time.Duration(c.Timeout))
	defer cancel()

	logger, err := bootstrap.NewLog(c.ENV, c.LogDir)
	if err != nil {
		log.Fatalln("Logfile error: ", err)
	}

	f, err := bootstrap.ExtractFfmpeg(ffmpeg)
	if err != nil {
		log.Fatalln("Не удалось распаковать ffmpeg")
	}

	defer func() {
		f.Close()
		os.Remove(f.Name())
		os.RemoveAll(c.Temp)

		timeFinish := time.Since(now)
		logger.D(fmt.Sprintf("Program is finished %v", timeFinish))

		if err := logger.Close(); err != nil {
			log.Println("Logfile close error: ", err)
		}

		closeChannels(channels)

		logger.Close()
	}()

	threadsCount := runtime.NumCPU()
	if c.ThreadMax > threadsCount {
		log.Fatalln("Current maximum threads are", threadsCount)
	}

	runtime.GOMAXPROCS(c.ThreadMax)

	conn, err := bootstrap.Open(c.DB)
	if err != nil {
		log.Fatalln("Database connection:", err)
	}

	defer conn.Close()

	httpClient, cloudAuthData, err := bootstrap.InitCloud(c.Cloud.Login, c.Cloud.Password)
	if err != nil {
		log.Fatalln("Cloud connection:", err)
	}

	// services
	storage := service.NewStorage(conn)
	cloud := service.NewCloud(ctx, httpClient, cloudAuthData.Token, cloudAuthData.OwnerID, logger)
	encode := service.NewEncoder(ctx, f.Name(), c.ThreadFfmpegMax, logger)

	// interactors
	vi := interactor.NewVideoCase(channels, c.ENV, c.Temp, c.RmOriginal, c.SkipNotFull, storage, cloud, encode, logger)
	go vi.Start(ctx)

	// handle signals, channels
	var result resultData
	go listen(ctx, channels, &result)

	select {
	case <-ctx.Done():
		logger.D(fmt.Sprintf("Программа останавливливается по таймауту"))
	case sig := <-shutdown:
		logger.E(fmt.Sprintf("Внеплановое завершение программы по сигналу %d", sig))
	case <-channels[domain.ChDone]:
		logger.D(fmt.Sprintf("Программа штатно завершилась"))
	}

	if err := result.Error(); err != nil {
		logger.E(fmt.Sprintf(`
			Получено видео: %d
			Сконвертировано: %d
			Не сконвертировано: %d
			Загружено на облако: %d
			Не загружено на облако: %d`,
			result.All,
			result.Converted,
			result.NotConverted,
			result.Uploaded,
			result.NotUploaded))
	}
}

type resultData struct {
	All          int
	Converted    int
	NotConverted int
	Uploaded     int
	NotUploaded  int
}

func (r *resultData) Error() error {
	if r.NotConverted > 0 || r.NotUploaded > 0 {
		return errors.New("произошли ошибки обработки")
	}

	return nil
}

func listen(ctx context.Context, ch map[int]chan int, data *resultData) {
	for {
		select {
		case <-ctx.Done():
			return
		case i := <-ch[domain.ChAll]:
			data.All += i
		case i := <-ch[domain.ChConverted]:
			data.Converted += i
		case i := <-ch[domain.ChNotConverted]:
			data.NotConverted += i
		case i := <-ch[domain.ChUploaded]:
			data.Uploaded += i
		case i := <-ch[domain.ChNotUploaded]:
			data.NotUploaded += i
		}
	}
}

func closeChannels(ch map[int]chan int) {
	for _, v := range ch {
		close(v)
	}
}
