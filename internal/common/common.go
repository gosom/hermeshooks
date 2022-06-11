package common

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog"
)

type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
}

func NewLogger(debug bool) zerolog.Logger {
	zerolog.TimestampFunc = func() time.Time {
		return time.Now().UTC()
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	logger := zerolog.New(os.Stderr).With().Timestamp().Logger()
	return logger
}

func RetryDo(client HTTPClient, req *http.Request, maxRetries int) (*http.Response, error) {
	var (
		body []byte
		err  error
		resp *http.Response
	)
	if req.Body != nil {
		body, err = io.ReadAll(req.Body)
		if err != nil {
			return resp, err
		}
	}
	if maxRetries <= 0 {
		maxRetries = 1
	}
	backoff := func(i int) time.Duration {
		return time.Duration(ExpSquaring(2, i))*time.Second + 5*time.Microsecond

	}
	for i := 1; i <= maxRetries; i++ {
		if len(body) > 0 {
			req.Body = io.NopCloser(bytes.NewReader(body))
		}
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if resp != nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
		if i != maxRetries {
			time.Sleep(backoff(i))
		}
	}
	return resp, err
}

func ExpSquaring(x, n int) int {
	if n < 0 {
		x = 1 / x
		n = -n
	}
	if n == 0 {
		return 1
	}
	y := 1
	for n > 1 {
		if n%2 == 0 {
			x = x * x
			n = n / 2
		} else {
			y = x * y
			x = x * x
			n = (n - 1) / 2
		}
	}
	return x * y
}

func RandomString(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	s := fmt.Sprintf("%X", b)
	return s
}
