package model

import "time"

type User struct {
	ID         string    `json:"id"`
	Key        string    `json:"key"`
	Expiration time.Time `json:"expiration"`
}
