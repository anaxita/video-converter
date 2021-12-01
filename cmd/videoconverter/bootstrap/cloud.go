package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const authURL = "https://auth.platformcraft.ru/token"

type AuthData struct {
	Token   string `json:"access_token"`
	OwnerID string `json:"owner_id"`
}

func InitCloud(login, password string) (*http.Client, *AuthData, error) {
	var authData AuthData

	var client http.Client

	dsn := fmt.Sprintf("%s?login=%s&password=%s", authURL, login, password)
	r, err := client.Post(dsn, "application/x-www-form-urlencoded", nil)
	if err != nil {
		return nil, nil, err
	}
	defer r.Body.Close()

	if err = json.NewDecoder(r.Body).Decode(&authData); err != nil {
		return nil, nil, err
	}

	return &client, &authData, nil
}
