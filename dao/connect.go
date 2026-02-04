package dao

import (
	"database/sql"

	config "github.com/mdhasib01/go-rest-starter/config"

	_ "github.com/lib/pq"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

var DB *sql.DB

func InitDatabase(connString string) error {
	db, err := sql.Open(config.Param.BaseDriver, connString)
	if err != nil {
		return err
	}

	err = db.Ping()
	if err != nil {
		return err
	}
	db.SetMaxIdleConns(50)

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		// log.Fatal(err.Error())
		return err
	}

	// read the migrations from the file system
	mig, err := migrate.NewWithDatabaseInstance("file://./dao/migrations",
		"postgres", driver)
	if err != nil {
		// log.Fatal(err.Error())
		return err
	}

	if err := mig.Up(); err != nil && err != migrate.ErrNoChange {
		return err
	}

	DB = db

	return nil
}
