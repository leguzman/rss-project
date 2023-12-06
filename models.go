package main

import (
	"time"

	"github.com/google/uuid"
	"github.com/leguzman/rss-project/internal/database"
)

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	APIKey    string    `json:"api_key"`
}

type Feed struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Name      string    `json:"name"`
	Url       string    `json:"url"`
	UserId    uuid.UUID `json:"user_id"`
}

func DBUserToUser(Dbuser database.User) User {
	return User{
		ID:        Dbuser.ID,
		CreatedAt: Dbuser.CreatedAt,
		UpdatedAt: Dbuser.UpdatedAt,
		Name:      Dbuser.Name,
		APIKey:    Dbuser.ApiKey,
	}
}

func DBFeedToFeed(DbFeed database.Feed) Feed {
	return Feed{
		ID:        DbFeed.ID,
		CreatedAt: DbFeed.CreatedAt,
		UpdatedAt: DbFeed.UpdatedAt,
		Name:      DbFeed.Name,
		Url:       DbFeed.Url,
		UserId:    DbFeed.UserID,
	}
}
