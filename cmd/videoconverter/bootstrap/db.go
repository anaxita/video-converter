package bootstrap

import (
	"fmt"
	"github.com/gocraft/dbr"
)

func Open(c DB) (*dbr.Connection, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", c.Username, c.Password, c.Host, c.Port, c.Name)
	conn, err := dbr.Open(c.Scheme, dsn, nil)
	if err != nil {
		return nil, err
	}

	if err = conn.Ping(); err != nil {
		return nil, err
	}

	conn.SetMaxOpenConns(10)

	return conn, nil
}
