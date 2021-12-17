package bootstrap

import (
	"github.com/pkg/errors"
	"os"
)

func ExtractFfmpeg(d []byte) (*os.File, error) {
	f, err := os.Create("ffmpeg")
	if err != nil {
		return nil, err
	}

	if err := f.Chmod(os.FileMode(0766)); err != nil {
		return nil, err
	}

	n, err := f.Write(d)
	if err != nil {
		return nil, err
	}

	if n != len(d) {
		return nil, errors.Errorf("Распаковалось только %d байт из %d", n, len(d))
	}

	return f, nil
}
