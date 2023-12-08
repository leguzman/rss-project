package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/leguzman/rss-project/internal/database"
	_ "github.com/lib/pq"
)

type apiConfig struct {
	DB *database.Queries
}

func main() {
	//	feed, err := urlToFeed("https://wagslane.dev/index.xml")
	//	if err != nil {
	//		log.Fatal(err)
	//	}
	//	fmt.Println(feed)

	godotenv.Load()
	port := os.Getenv("PORT")
	fmt.Println("Port: ", port)
	conn, err := sql.Open("postgres", os.Getenv("DB_URL"))
	handleError(err, "Can't connect to database: %v")

	apiCfg := apiConfig{
		DB: database.New(conn),
	}
	go startScraping(apiCfg.DB, 10, time.Minute)

	server := &http.Server{
		Handler: getRouter(apiCfg),
		Addr:    ":" + port,
	}

	err = server.ListenAndServe()
	handleError(err, "Could not start server: %v")
}
