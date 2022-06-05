package common

import (
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewLogger() zerolog.Logger {
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	return logger
}
