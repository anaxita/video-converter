package bootstrap

import (
	"fmt"
	"github.com/gocraft/dbr"
)

func Open(scheme, username, password, port, name string) (*dbr.Connection, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(localhost:%s)/%s", username, password, port, name)
	conn, err := dbr.Open(scheme, dsn, nil)
	if err != nil {
		return nil, err
	}

	if err = conn.Ping(); err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(10)

	return conn, nil
}
