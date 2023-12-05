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

func DBUserToUser(Dbuser database.User) User {
	return User{
		ID:        Dbuser.ID,
		CreatedAt: Dbuser.CreatedAt,
		UpdatedAt: Dbuser.UpdatedAt,
		Name:      Dbuser.Name,
		APIKey:    Dbuser.ApiKey,
	}
}
