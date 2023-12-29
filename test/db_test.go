package test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/leguzman/rss-project/handlers"
	"github.com/leguzman/rss-project/internal/database"
	"github.com/leguzman/rss-project/models"
	"github.com/leguzman/rss-project/routes"
	_ "github.com/lib/pq"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var db *sql.DB
var server *http.Server
var apiKey string
var	feed models.Feed
var result handlers.WrappedSlice[models.FeedFollow]

func TestMain(m *testing.M) {

	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("unix:///run/docker.sock")
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

func TestHealthAndRoot(t *testing.T) {
	server = &http.Server{
		Handler: routes.GetRouter(handlers.ApiConfig{DB: database.New(db)}),
	}
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	response := executeRequest(req, server)
	checkResponseCode(t, http.StatusOK, response.Code)

	req, _ = http.NewRequest(http.MethodGet, "/v1/healthz", nil)
	response = executeRequest(req, server)
	checkResponseCode(t, http.StatusOK, response.Code)
}

func createUser(name string)(req *http.Request){
	jsonBody := []byte(`{"name": "`+name+`"}`)
	bodyReader := bytes.NewReader(jsonBody)
	req, _ = http.NewRequest(http.MethodPost, "/v1/users", bodyReader)
    return
}
func TestUserHandler(t *testing.T) {
	queries := database.New(db)
	server = &http.Server{
		Handler: routes.GetRouter(handlers.ApiConfig{DB: queries}),
	}
	response := executeRequest(createUser("Luis"), server)

	checkResponseCode(t, http.StatusCreated, response.Code)
	assert.Contains(t, response.Body.String(), `"name":"Luis"`)

	user := models.User{}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Couldn't read user!")
	}
	err = json.Unmarshal(body, &user)

	if err != nil {
		log.Fatal("Couldn't read user!")
	}
	apiKey = "ApiKey " + user.APIKey
    req, _ := http.NewRequest(http.MethodGet, "/v1/users", nil)
	req.Header.Add("Authorization", apiKey)

	response = executeRequest(req, server)
	checkResponseCode(t, http.StatusOK, response.Code)


}


func TestFeedsHandler(t *testing.T){
	queries := database.New(db)
    server:= &http.Server{
        Handler: routes.GetRouter(handlers.ApiConfig{DB: queries}),
    }
    jsonBody := []byte(`
	{
		"name": "Wags Lane's Blog 2",
		"url":"https://boot.dev/index.xml"
	}`)
    bodyReader := bytes.NewReader(jsonBody)
    req, _ := http.NewRequest(http.MethodPost, "/v1/feeds", bodyReader)
	req.Header.Add("Authorization", apiKey)

    response := executeRequest(req, server)
	checkResponseCode(t, http.StatusCreated, response.Code)
	assert.Contains(t, response.Body.String(), `"name":"Wags Lane's Blog 2"`)

    body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Couldn't read feed!")
	}
	err = json.Unmarshal(body, &feed)

	if err != nil {
		log.Fatal("Couldn't read feed!")
	}

	req, _ = http.NewRequest(http.MethodGet, "/v1/feeds", nil)
	req.Header.Add("Authorization", apiKey)

	response = executeRequest(req, server)
	checkResponseCode(t, http.StatusOK, response.Code)

}

func TestFeedFollowsHandler(t *testing.T){
    jsonBody := []byte(fmt.Sprintf(`
	{
		"feed_id": "%s"
	}`, feed.ID))
    bodyReader := bytes.NewReader(jsonBody)
    req, _ := http.NewRequest(http.MethodPost, "/v1/feed_follows", bodyReader)
	req.Header.Add("Authorization", apiKey)
    response := executeRequest(req, server)


	checkResponseCode(t, http.StatusCreated, response.Code)
	assert.Contains(t, response.Body.String(), feed.ID.String())

	req, _ = http.NewRequest(http.MethodGet, "/v1/feed_follows", nil)
	req.Header.Add("Authorization", apiKey)
	response = executeRequest(req, server)
	checkResponseCode(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), feed.ID.String())
	log.Printf("FeedFollow: %v", response.Body.String())

    body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal("Couldn't read response body!")
	}
	result = handlers.WrappedSlice[models.FeedFollow]{}
	json.Unmarshal(body, &result)

	req, _ = http.NewRequest(http.MethodDelete, fmt.Sprintf("/v1/feed_follows/%s", result.Results[0].ID.String()), nil)
	log.Info(req.URL)
	req.Header.Add("Authorization", apiKey)

	response = executeRequest(req, server)
	assert.Equal(t, response.Body.String(), "{}")
	checkResponseCode(t, http.StatusNoContent, response.Code)
}

func TestUserPosts(t *testing.T){
	queries := database.New(db)
    jsonBody := []byte(fmt.Sprintf(`
	{
		"feed_id": "%s"
	}`, feed.ID))
    bodyReader := bytes.NewReader(jsonBody)
    req, _ := http.NewRequest(http.MethodPost, "/v1/feed_follows", bodyReader)
	req.Header.Add("Authorization", apiKey)
    response := executeRequest(req, server)


	checkResponseCode(t, http.StatusCreated, response.Code)
	assert.Contains(t, response.Body.String(), feed.ID.String())

	post, err := queries.CreatePost(context.Background(), database.CreatePostParams{
		ID:          uuid.New(),
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
		Title:       "Test Post",
		Description: sql.NullString{String: "Test Desc"},
		PublishedAt: time.Now().UTC(),
		Url:         "test link",
		FeedID:      feed.ID,
	})
	if err != nil {
		log.Fatal("Couldn't populate Db with posts!")
	}
    req, _ = http.NewRequest(http.MethodGet, "/v1/posts", nil)
	req.Header.Add("Authorization", apiKey)

    response = executeRequest(req, server)
	checkResponseCode(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), post.ID.String())

	req, _ = http.NewRequest(http.MethodGet, "/v1/post", nil)
	req.Header.Add("Authorization", apiKey)

	response = executeRequest(req, server)
	checkResponseCode(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), post.ID.String())
}


func executeRequest(req *http.Request, s *http.Server) *httptest.ResponseRecorder {
    rr := httptest.NewRecorder()
	s.Handler.ServeHTTP(rr, req)
	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
