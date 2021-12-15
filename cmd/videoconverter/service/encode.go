package service

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"os/exec"
	"path"
	"videoconverter/bootstrap"
	"videoconverter/domain"
)

type cmdError struct {
	out []byte
	err error
}

func (e cmdError) Error() string {
	return fmt.Sprintf("%v\n%s\n", e.err, e.out)
}

type VideoEncoder struct {
	ctx context.Context
	l   *bootstrap.Logger
}

func NewEncoder(ctx context.Context, l *bootstrap.Logger) *VideoEncoder {
	return &VideoEncoder{
		ctx: ctx,
		l:   l,
	}
}

// Convert a video from src to dst with q quality, return path to new video.
func (e *VideoEncoder) Convert(tmp string, filePath string, quality domain.VQ) (string, error) {
	fmt.Sprintf("Начинаю конвертировать файл %s в качество %d", filePath, quality)

	_, fName := path.Split(filePath)
	outVideo := fmt.Sprintf("%s/v-%d-%s", tmp, quality, fName)

	cmd := exec.CommandContext(e.ctx,
		"ffmpeg",
		"-y",
		"-i",
		filePath,
		"-profile:v",
		"baseline",
		"-movflags",
		"+faststart",
		"-vcodec",
		"libx264",
		"-crf",
		"28",
		"-preset",
		"faster",
		"-acodec",
		"aac",
		"-filter:v",
		fmt.Sprintf("scale=trunc(oh*a/2)*2:%d", quality),
		outVideo,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(cmdError{out, err})
	}

	fmt.Sprintf("Успешно сконвертировали файл %s в качество %d", filePath, quality)

	return outVideo, nil
}

func (e *VideoEncoder) CreatePreview(tmp, filePath string) (string, error) {
	fmt.Sprintf("Создается превью файла %s", filePath)

	_, fName := path.Split(filePath)
	outVideo := fmt.Sprintf("%s/v-preview-%s", tmp, fName)

	cmd := exec.CommandContext(e.ctx,
		"ffmpeg",
		"-y",
		"-ss",
		"00:00:00",
		"-to",
		"00:03:00",
		"-i",
		filePath,
		"-c",
		"copy",
		outVideo,
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", errors.WithStack(cmdError{out, err})
	}

	fmt.Sprintf("Успешно создано превью для файла %s", filePath)

	return outVideo, nil
}
