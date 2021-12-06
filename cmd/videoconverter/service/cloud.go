package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"videoconverter/domain"
)

const (
	apiURL = "https://filespot.platformcraft.ru/2/fs/container"
)

var (
	ErrFreeSpace    = errors.New("на облаке кончилось место")
	ErrNotFullWrite = errors.New("файл был загружен не полностью")
)

// Cloud describe a remote file cloud
type Cloud struct {
	ctx     context.Context
	client  *http.Client
	token   string
	ownerID string
}

// NewCloud returns ready for use *Cloud instance
func NewCloud(ctx context.Context, client *http.Client, token, ownerID string) *Cloud {
	return &Cloud{
		ctx:     ctx,
		client:  client,
		token:   token,
		ownerID: ownerID,
	}
}

// DownloadFile downloads a file from url u into file f
// Use for downloading an original file for next converting
func (c *Cloud) DownloadFile(u string, f *os.File) error {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	req = req.WithContext(c.ctx)

	r, err := c.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return errors.WithMessagef(err, "reponse code is %d", r.StatusCode)
	}

	n, err := io.Copy(f, r.Body)
	if err != nil {
		return errors.WithStack(err)
	}

	if n < r.ContentLength {
		return errors.WithStack(ErrNotFullWrite)
	}

	return nil
}

// UploadFile uploads a converted file to the cloud
func (c *Cloud) UploadFile(path string, f *os.File) (string, error) {
	var apiResponse struct {
		DownloadUrl string `json:"download_url"`
	}

	var body bytes.Buffer

	w := multipart.NewWriter(&body)

	wr, err := w.CreateFormFile("file", f.Name())
	if err != nil {
		return "", err
	}

	n, err := io.Copy(wr, f)
	if err != nil {
		return "", err
	}

	w.Close()

	uri := fmt.Sprintf("%s/%s/object/videoconverter/%s", apiURL, c.ownerID, path)

	req, err := http.NewRequest(http.MethodPost, uri, bytes.NewReader(body.Bytes()))
	if err != nil {
		return "", errors.WithStack(err)
	}

	req = req.WithContext(c.ctx)

	req.Header.Add("Authorization", "Bearer "+c.token)
	req.Header.Add("Content-Type", w.FormDataContentType())
	req.Header.Add("Content-Length", fmt.Sprintf("%d", n))

	res, err := c.client.Do(req)
	if err != nil {
		return "", errors.WithStack(err)
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Println("can't close response body ", err)
		}
	}()

	if res.StatusCode == http.StatusConflict {
		return domain.CacheURL + path, nil
	}

	if res.StatusCode == http.StatusInsufficientStorage {
		return "", ErrFreeSpace
	}

	if res.StatusCode != http.StatusOK {
		return "", errors.WithMessagef(err, "reponse code is %d", res.StatusCode)

	}

	err = json.NewDecoder(res.Body).Decode(&apiResponse)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return apiResponse.DownloadUrl, nil
}

// Delete deletes a converted file from the cloud
// Use for delete large original files after converting to all required formats
func (c *Cloud) Delete(filepath string) error {
	uri := fmt.Sprintf("%s/%s/object/%s", apiURL, c.ownerID, filepath)

	req, err := http.NewRequest(http.MethodDelete, uri, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Add("Content-Type", "multipart/form-data")
	req.Header.Add("Authorization", "Bearer "+c.token)

	req = req.WithContext(c.ctx)

	res, err := c.client.Do(req)
	if err != nil {
		return errors.WithStack(err)
	}

	if res.StatusCode != http.StatusOK {
		return errors.WithStack(err)
	}

	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Println("can't close response body ", err)
		}
	}()

	return nil
}
