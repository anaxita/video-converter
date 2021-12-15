package bootstrap

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
	"videoconverter/domain"
)

type Logger struct {
	env string
	f   *os.File
	log *log.Logger
}

// NewLog creates a new logfile into with current date into logDir
func NewLog(env, logDir string) (*Logger, error) {
	if err := os.Mkdir(logDir, os.FileMode(0766)); err != nil && !os.IsExist(err) {
		return nil, err
	}

	timeSting := time.Now().UTC().Format("2006-02-01-15-04-05")
	logPath := fmt.Sprintf("%s/video-converter-%s.log", logDir, timeSting)

	f, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}

	l := log.New(os.Stdout, "", 0)

	return &Logger{
		env: env,
		f:   f,
		log: l,
	}, err
}

func (l *Logger) Close() error {
	return l.f.Close()
}

func (l *Logger) E(s ...string) {
	b := &strings.Builder{}
	b.WriteString(timeFormat())
	b.WriteString(" [ERROR] ")
	b.WriteString(strings.Join(s, " "))

	l.f.WriteString(b.String())

	if l.env == domain.EnvDebug {
		l.log.Println(b.String())
	}
}

func (l *Logger) D(s ...string) {
	if l.env == domain.EnvDebug {
		b := &strings.Builder{}
		b.WriteString(timeFormat())
		b.WriteString(" [DEBUG] ")
		b.WriteString(strings.Join(s, " "))

		l.log.Println(b.String())
	}
}
