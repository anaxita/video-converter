package bootstrap

import (
	"github.com/pkg/errors"
	"os/exec"
	"strings"
)

func CheckFfmpegVersion(v string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-version",
	)

	out, err := cmd.Output()
	if err != nil {
		return err
	}

	if !strings.Contains(string(out), v) {
		return errors.New("not compare versions")
	}

	return nil
}
