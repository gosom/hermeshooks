package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

type Config struct {
	Log            zerolog.Logger
	DB             *storage.DB
	RapidApiKey    string
	InternalApiKey string
}

type AuthService struct {
	log            zerolog.Logger
	db             *storage.DB
	rapidApiKey    string
	internalApiKey string
}

func New(cfg Config) (*AuthService, error) {
	ans := AuthService{
		log:            cfg.Log,
		db:             cfg.DB,
		rapidApiKey:    cfg.RapidApiKey,
		internalApiKey: cfg.InternalApiKey,
	}
	return &ans, nil
}

func (a *AuthService) RapidApi(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		incoming := req.Header.Get("X-API-KEY")
		a.log.Info().Str("api", incoming).Str("vs", a.rapidApiKey).Msg("")
		if incoming != a.rapidApiKey {
			return toJSON(w, http.StatusUnauthorized, nil)
		}
		for k, v := range req.Header {
			fmt.Println(k, v)
		}
		suffix := req.Header.Get("X-Rapidapi-User")
		if len(suffix) == 0 {
			return toJSON(w, http.StatusUnauthorized, nil)
		}
		user, err := func() (entities.User, error) {
			ctx := req.Context()
			var user entities.User
			username := "rapid_api_" + suffix
			tx, err := a.db.Begin()
			if err != nil {
				return user, err
			}
			defer tx.Rollback()
			exists, err := storage.UserExists(ctx, tx, username)
			if err != nil {
				return user, err
			}
			switch exists {
			case true:
				return storage.GetUserByUserName(ctx, tx, username)
			default:
				user.Username = username
				user.CreatedAt = time.Now().UTC()
				user, err = storage.InsertUser(ctx, tx, user)
				if err != nil {
					return user, err
				}
				return user, tx.Commit()
			}
		}()
		if err != nil {
			return err
		}
		ctx := context.WithValue(req.Context(), common.UserCtxKey, user)
		next(w, req.WithContext(ctx))
		return nil
	}
}

func (a *AuthService) InternalApi(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		incoming := req.Header.Get("X-API-KEY")
		if incoming != a.internalApiKey {
			return toJSON(w, http.StatusUnauthorized, nil)
		}
		next(w, req)
		return nil
	}
}

func toJSON(w http.ResponseWriter, statusCode int, value interface{}) error {
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
