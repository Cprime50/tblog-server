package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"thelsblog-server/internal/data"
	"thelsblog-server/internal/driver"
)

// Our port for where our server will listen from
type config struct {
	port string
}

type application struct {
	config      config
	infoLog     *log.Logger
	errorLog    *log.Logger
	db          *driver.DB
	models      data.Models
	environment string
}

func main() {

	//Local server will listen at port 8082 bc we already have our vue server at 8081
	var cfg config
	cfg.port = "localhost:8082"

	//declaring our log to get useful information form our cli
	infoLog := log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	errorLog := log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)

	//data source name for our database
	dsn := os.Getenv("DSN")
	environment := os.Getenv("ENV")

	db, err := driver.ConnectPostgres(dsn)
	if err != nil {
		log.Fatal("Cannot connect to database")
	}

	//some connections may refuse to close even after exiting, this should handle that
	defer db.SQL.Close()

	// initializing our application struct
	app := &application{
		config:      cfg,
		infoLog:     infoLog,
		errorLog:    errorLog,
		models:      data.New(db.SQL),
		environment: environment,
	}

	// start the webserver
	err = app.serve()
	if err != nil {
		log.Fatal(err)
	}

}

// Serve function starts the webserver
func (app *application) serve() error {
	app.infoLog.Printf("API is now listening on port %s .....", app.config.port)

	srv := &http.Server{
		Addr:    fmt.Sprintf("%s", app.config.port),
		Handler: app.routes(),
	}
	return srv.ListenAndServe()
}
