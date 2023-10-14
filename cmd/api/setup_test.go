package main

import (
	"log"
	"os"
	"testing"
	"thelsblog-server/internal/data"

	"github.com/DATA-DOG/go-sqlmock"
)

var testApp application
var mockDB sqlmock.Sqlmock

func TestMain(m *testing.M) {
	testDB, myMock, _ := sqlmock.New()
	mockDB = myMock

	defer testDB.Close()

	testApp = application{
		config:      config{},
		infoLog:     log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime),
		errorLog:    log.New(os.Stdout, "Error\t", log.Ldate|log.Ltime),
		models:      data.New(testDB),
		environment: "developement",
	}

	os.Exit(m.Run())

}
