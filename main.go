package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/leguzman/rss-project/handlers"
	"github.com/leguzman/rss-project/internal/database"
	"github.com/leguzman/rss-project/routes"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	port := os.Getenv("PORT")
	fmt.Println("Port: ", port)
	conn, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		log.Fatal("Can't connect to database: ", err)
	}

	apiCfg := handlers.ApiConfig{
		DB: database.New(conn),
	}
	go startScraping(apiCfg.DB, 10, time.Minute)

	server := &http.Server{
		Handler: routes.GetRouter(apiCfg),
		Addr:    ":" + port,
	}

	err = server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}
