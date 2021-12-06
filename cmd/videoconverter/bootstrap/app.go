package bootstrap

import (
	"fmt"
	"github.com/joho/godotenv"
	"os"
	"strconv"
	"time"
)

// App describe app configuration
type App struct {
	ENV       string
	LogDir    string
	Temp      string
	Timeout   int
	ThreadMax int
	Cloud     Cloud
	DB        DB
}

// Cloud describe cloud configuration
type Cloud struct {
	Login    string
	Password string
}

// DB describe database configuration
type DB struct {
	Scheme   string
	Name     string
	Port     string
	Username string
	Password string
}

// New parses .env file, create a logdir, tempdir and returns a ready for use App config
func New(pathToConfig string) (*App, error) {
	err := godotenv.Load(pathToConfig)
	if err != nil {
		return nil, err
	}

	var c App

	c.ENV = os.Getenv("ENV")
	c.LogDir = os.Getenv("LOG_DIR")
	c.Temp = os.Getenv("TMP_DIR")

	if err = os.Mkdir(c.LogDir, os.FileMode(0766)); err != nil && !os.IsExist(err) {
		return nil, err
	}

	if err = os.Mkdir(c.Temp, os.FileMode(0766)); err != nil && !os.IsExist(err) {
		return nil, err
	}

	timeout := os.Getenv("TIMEOUT")
	c.Timeout, err = strconv.Atoi(timeout)
	if err != nil {
		return nil, err
	}

	threadMax := os.Getenv("THREAD_MAX")
	c.ThreadMax, err = strconv.Atoi(threadMax)
	if err != nil {
		return nil, err
	}

	c.Cloud.Login = os.Getenv("CLOUD_LOGIN")
	c.Cloud.Password = os.Getenv("CLOUD_PASSWORD")

	c.DB.Scheme = os.Getenv("DB_SCHEME")
	c.DB.Port = os.Getenv("DB_PORT")
	c.DB.Name = os.Getenv("DB_NAME")
	c.DB.Username = os.Getenv("DB_USERNAME")
	c.DB.Password = os.Getenv("DB_PASSWORD")

	return &c, nil
}

// NewLog creates a new logfile into with current date into logDir
func NewLog(logDir string) (*os.File, error) {
	timeSting := time.Now().UTC().Format("2006-01-02-15-04-05")
	logPath := fmt.Sprintf("%s/video-converter-%s.log", logDir, timeSting)

	return os.Create(logPath)
}
