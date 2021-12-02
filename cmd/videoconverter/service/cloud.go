package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"videoconverter/domain"
)

const (
	apiURL = "https://filespot.platformcraft.ru/2/fs/container"
)

var (
	ErrNotStatusOK  = errors.New("download file failed, code is not 200")
	ErrNotFullWrite = errors.New("file was not recorded completely")
)

type Cloud struct {
	client  *http.Client
	token   string
	ownerID string
}

func NewCloud(client *http.Client, token, ownerID string) *Cloud {
	return &Cloud{
		client:  client,
		token:   token,
		ownerID: ownerID,
	}
}

// DownloadFile downloads a file from url u into file f
func (c *Cloud) DownloadFile(u string, f *os.File) error {
	r, err := http.Get(u)
	if err != nil {
		return errors.WithStack(err)
	}

	defer r.Body.Close()

	if r.StatusCode != http.StatusOK {
		return ErrNotStatusOK
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

	if res.StatusCode != http.StatusOK {
		var resp map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
			return "", ErrNotStatusOK
		}

		if strings.Contains(resp["msg"].(string), "duplication") {
			return domain.CacheURL + path, nil
		}

		return "", errors.New(resp["msg"].(string))
	}

	err = json.NewDecoder(res.Body).Decode(&apiResponse)
	if err != nil {
		return "", errors.WithStack(err)
	}

	return apiResponse.DownloadUrl, nil
}

func (c *Cloud) Delete(filepath string) error {
	uri := fmt.Sprintf("%s/%s/object/%s", apiURL, c.ownerID, filepath)

	req, err := http.NewRequest(http.MethodDelete, uri, nil)
	if err != nil {
		return errors.WithStack(err)
	}

	req.Header.Add("Content-Type", "multipart/form-data")
	req.Header.Add("Authorization", "Bearer "+c.token)

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
