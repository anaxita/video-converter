package main

import (
	"context"
	"flag"
	"fmt"
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

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	log.Println("Program is started")

	pathToConfig := flag.String("c", "./.env", "path to .env config")
	flag.Parse()
	now := time.Now()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdown := make(chan os.Signal)
	signal.Notify(shutdown, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGSTOP)
	defer close(shutdown)

	// configs
	c, err := bootstrap.New(*pathToConfig)
	if err != nil {
		log.Fatalln("Config load:", err)
	}

	logger, err := bootstrap.NewLog(c.ENV, c.LogDir)
	if err != nil {
		log.Fatalln("Logfile error: ", err)
	}

	defer func() {
		timeFinish := time.Since(now)
		log.Println("Program is finished", timeFinish)

		if err := logger.Close(); err != nil {
			log.Println("Logfile close error: ", err)
		}
	}()

	threadsCount := runtime.NumCPU()
	if c.ThreadMax > threadsCount {
		log.Fatalln("Current maximum threads are", threadsCount)
	}

	runtime.GOMAXPROCS(c.ThreadMax)

	conn, err := bootstrap.Open(c.DB.Scheme, c.DB.Username, c.DB.Password, c.DB.Port, c.DB.Name)
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
	encode := service.NewEncoder(ctx, logger)

	// interactors
	channels := map[int]chan int{
		domain.ChDone:         make(chan int),
		domain.ChAll:          make(chan int),
		domain.ChConverted:    make(chan int),
		domain.ChNotConverted: make(chan int),
		domain.ChUploaded:     make(chan int),
	}
	defer closeChannels(channels)

	deadline := now.Add(time.Hour * time.Duration(c.Timeout))

	vi := interactor.NewVideoCase(channels, c.ENV, c.Temp, c.RmOriginal, c.SkipNotFull, storage, cloud, encode, logger)
	go vi.Start(deadline)

	// handle signals, channels
	var result resultData
	go listen(ctx, channels, &result)

	select {
	case sig := <-shutdown:
		logger.E(fmt.Sprintf("Внеплановое завершение программы по сигналу %d", sig))
	case <-channels[domain.ChDone]:
		logger.E(fmt.Sprintf("Все обработчики завершили работу"))
	}

	logger.E(fmt.Sprintf(`
	Получено видео: %d
	Сконвертировано: %d
	Ошибок конвертирования: %d
	Загружено на облако: %d
	Ошибок загрузки на облако: %d`,
		result.All,
		result.Converted,
		result.NotConverted,
		result.Uploaded,
		result.NotUploaded))

	os.RemoveAll(c.Temp)
}

type resultData struct {
	All          int
	Converted    int
	NotConverted int
	Uploaded     int
	NotUploaded  int
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
