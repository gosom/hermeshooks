package rest

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

func Bind(r bunrouter.Request, ans any) error {
	if err := json.NewDecoder(r.Body).Decode(&ans); err != nil {
		if errors.Is(err, io.EOF) {
			return err
		}
		return ValidationError{"invalid json"}
	}
	return nil
}

func JSON(w http.ResponseWriter, statusCode int, value interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if value == nil {
		return nil
	}
	if err := json.NewEncoder(w).Encode(value); err != nil {
		return err
	}
	return nil
}

func logHandler(log zerolog.Logger) func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			rec := NewResponseWriter(w)
			now := time.Now()
			err := next(rec.Wrapped, req)
			realIp, _ := getIP(req)
			dur := time.Since(now)
			statusCode := rec.StatusCode()
			ev := log.Info().
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Int("statusCode", statusCode).
				IPAddr("ip", realIp).
				Dur("duration", dur)
			if err != nil {
				ev.Err(err)
			}
			ev.Msg(http.StatusText(statusCode))
			return err
		}
	}
}

func acceptedContentType(contentType ...string) func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
		return func(w http.ResponseWriter, req bunrouter.Request) error {
			incoming := req.Header.Get("Content-Type")
			valid := false
			for i := range contentType {
				if contentType[i] == incoming {
					valid = true
					break
				}
			}
			if !valid {
				return ValidationError{Message: "Header value Content-Type must be application/json"}
			}
			err := next(w, req)
			return err
		}
	}
}

type ResponseWriter struct {
	Wrapped    http.ResponseWriter
	statusCode int
}

func NewResponseWriter(w http.ResponseWriter) *ResponseWriter {
	var rw ResponseWriter
	rw.Wrapped = httpsnoop.Wrap(w, httpsnoop.Hooks{
		WriteHeader: func(next httpsnoop.WriteHeaderFunc) httpsnoop.WriteHeaderFunc {
			return func(statusCode int) {
				if rw.statusCode == 0 {
					rw.statusCode = statusCode
				}
				next(statusCode)
			}
		},
	})
	return &rw
}

func (w *ResponseWriter) StatusCode() int {
	if w.statusCode != 0 {
		return w.statusCode
	}
	return http.StatusOK
}

func getIP(r bunrouter.Request) (net.IP, error) {
	//Get IP from the X-REAL-IP header
	ip := r.Header.Get("X-REAL-IP")
	netIP := net.ParseIP(ip)
	if len(netIP) > 0 {
		return netIP, nil
	}

	//Get IP from X-FORWARDED-FOR header
	ips := r.Header.Get("X-FORWARDED-FOR")
	splitIps := strings.Split(ips, ",")
	for _, ip := range splitIps {
		netIP := net.ParseIP(ip)
		if len(netIP) > 0 {
			return netIP, nil
		}
	}

	//Get IP from RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return nil, err
	}
	netIP = net.ParseIP(ip)
	if len(netIP) > 0 {
		return netIP, nil
	}
	return nil, fmt.Errorf("No valid ip found")
}

//------------------------------------------------------------------------------
