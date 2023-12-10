package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/leguzman/rss-project/internal/database"
)

func (apiCfg *apiConfig) handlerCreateUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Name string `json:"name"`
	}
	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Error parsing json: %v", err))
		return
	}
	user, err := apiCfg.DB.CreateUser(r.Context(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
		Name:      params.Name,
	})
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Create user err: %v", err))
		return
	}
	respondWithJson(w, 201, DBUserToUser(user))
}

func (apiCfg *apiConfig) handlerGetUser(w http.ResponseWriter, r *http.Request, user database.User) {
	respondWithJson(w, 200, DBUserToUser(user))
}

func (apiCfg *apiConfig) handlerGetUserPosts(w http.ResponseWriter, r *http.Request, user database.User) {
	limitStr := r.URL.Query().Get("limit")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}
	posts, err := apiCfg.DB.GetUserPosts(r.Context(), database.GetUserPostsParams{
		UserID: user.ID,
		Limit:  int32(limit),
	})
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Couldn't get posts: %v", err))
		return
	}
	response := WrappedSlice{Results: DBPostsToPosts(posts), Size: len(posts)}
	respondWithJson(w, 200, response)
}

func (apiCfg *apiConfig) handlerFilterUserPosts(w http.ResponseWriter, r *http.Request, user database.User) {
	limitStr := r.URL.Query().Get("limit")
	description := r.URL.Query().Get("description")
	title := r.URL.Query().Get("title")
	before, err := time.Parse(time.DateOnly, r.URL.Query().Get("before"))
	if err != nil {
		log.Printf("Error parsing before date: %s", err)
		before = time.Time{}
	}
	after, err := time.Parse(time.DateOnly, r.URL.Query().Get("after"))
	if err != nil {
		log.Printf("Error parsing after date: %s", err)
		after = time.Time{}
	}
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}
	posts, err := apiCfg.DB.FilterUserPosts(r.Context(), database.FilterUserPostsParams{
		UserID:      user.ID,
		Description: description,
		Title:       title,
		Before:      before,
		After:       after,
		Limit:       int32(limit),
	})
	if err != nil {
		respondWithError(w, 400, fmt.Sprintf("Couldn't get posts: %v", err))
		return
	}
	response := WrappedSlice{Results: DBPostsToPosts(posts), Size: len(posts)}
	respondWithJson(w, 200, response)
}
