package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/leguzman/rss-project/handlers"
	"github.com/leguzman/rss-project/internal/database"
	"github.com/leguzman/rss-project/routes"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var db *sql.DB

func TestMain(m *testing.M) {
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not construct pool: %s", err)
	}

	err = pool.Client.Ping()
	if err != nil {
		log.Fatalf("Could not connect to Docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "postgres",
		Tag:        "11",
		Env: []string{
			"POSTGRES_PASSWORD=secret",
			"POSTGRES_USER=user_name",
			"POSTGRES_DB=dbname",
			"listen_addresses = '*'",
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	hostAndPort := resource.GetHostPort("5432/tcp")
	databaseUrl := fmt.Sprintf("postgres://user_name:secret@%s/dbname?sslmode=disable", hostAndPort)

	log.Println("Connecting to database on url: ", databaseUrl)

	resource.Expire(120) // Tell docker to hard kill the container in 120 seconds

	// exponential backoff-retry, because the application in the container might not be ready to accept connections yet
	pool.MaxWait = 120 * time.Second
	if err = pool.Retry(func() error {
		db, err = sql.Open("postgres", databaseUrl)
		if err != nil {
			return err
		}
		return db.Ping()
	}); err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}
	// Run migrations
	var args = []string{
		"postgres",
		databaseUrl,
		"up",
	}

	exe := exec.Command("goose", args...)
	log.Print(exe)
	dir, _ := os.Getwd()
	exe.Dir = strings.Split(dir, "db_test.go")[0] + "/../sql/schema"
	log.Print(exe.Dir)

	output := exe.Run()
	fmt.Println(output)

	//Run tests
	code := m.Run()

	// You can't defer this because os.Exit doesn't care for defer
	if err := pool.Purge(resource); err != nil {
		log.Fatalf("Could not purge resource: %s", err)
	}

	os.Exit(code)
}

func TestHelloWorld(t *testing.T) {

	// Create a New Server Struct
	s := &http.Server{
		Handler: routes.GetRouter(handlers.ApiConfig{DB: database.New(db)}),
	}

	// Create a New Request
	req, _ := http.NewRequest(http.MethodGet, "/", nil)

	// Execute Request
	response := executeRequest(req, s)

	// Check the response code
	checkResponseCode(t, http.StatusOK, response.Code)

	// Create a New Request
	req, _ = http.NewRequest(http.MethodGet, "/v1/healthz", nil)

	// Execute Request
	response = executeRequest(req, s)

	// Check the response code
	checkResponseCode(t, http.StatusOK, response.Code)

	// Create a New Request
	jsonBody := []byte(`{"name": "Luis"}`)
	bodyReader := bytes.NewReader(jsonBody)
	req, _ = http.NewRequest(http.MethodPost, "/v1/users", bodyReader)

	// Execute Request
	response = executeRequest(req, s)

	// Check the response code
	checkResponseCode(t, http.StatusCreated, response.Code)
	assert.Contains(t, string(response.Body.Bytes()), `"name":"Luis"`)
	// We can use testify/require to assert values, as it is more convenient
}

// executeRequest, creates a new ResponseRecorder
// then executes the request by calling ServeHTTP in the router
// after which the handler writes the response to the response recorder
// which we can then inspect.
func executeRequest(req *http.Request, s *http.Server) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.Handler.ServeHTTP(rr, req)
	return rr
}

// checkResponseCode is a simple utility to check the response code
// of the response
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
