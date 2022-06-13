package entities

import "time"

type User struct {
	ID        int64
	Username  string
	ApiKey    string
	CreatedAt time.Time
}
