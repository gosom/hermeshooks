package client

import (
	"fmt"

	"github.com/google/uuid"
)

type HttpError struct {
	StatusCode int    `json:"statusCode"`
	Message    string `json:"mesage"`
}

func (e HttpError) Error() string {
	return fmt.Sprintf("StatusCode: %d Message: %s", e.StatusCode, e.Message)
}

type Worker struct {
	Name      uuid.UUID
	Partition int
}
