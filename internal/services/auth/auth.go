package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gosom/hermeshooks/internal/common"
	"github.com/gosom/hermeshooks/internal/cryptoutils"
	"github.com/gosom/hermeshooks/internal/entities"
	"github.com/gosom/hermeshooks/internal/storage"
	"github.com/rs/zerolog"
	"github.com/uptrace/bunrouter"
)

var ErrUnauthorized = errors.New(http.StatusText(http.StatusUnauthorized))

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

func (a *AuthService) AuthMiddleware(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		var (
			user entities.User
			err  error
		)
		if a.isRapidApiReq(req) {
			user, err = a.rapidApiAuth(req)
		} else {
			user, err = a.localAuth(req)
		}

		if err != nil {
			if errors.Is(err, ErrUnauthorized) {
				return toJSON(w, http.StatusUnauthorized, nil)
			}
			return toJSON(w, http.StatusInternalServerError, nil)
		}
		ctx := context.WithValue(req.Context(), common.UserCtxKey, user)
		return next(w, req.WithContext(ctx))
	}
}

func (a *AuthService) isRapidApiReq(req bunrouter.Request) bool {
	incoming := req.Header.Get("X-RapidAPI-Proxy-Secret")
	return len(incoming) > 0
}

func (a *AuthService) rapidApiAuth(req bunrouter.Request) (u entities.User, err error) {
	incoming := req.Header.Get("X-RapidAPI-Proxy-Secret")
	if incoming != a.rapidApiKey {
		err = ErrUnauthorized
		return
	}
	suffix := req.Header.Get("X-Rapidapi-User")
	if len(suffix) == 0 {
		err = ErrUnauthorized
		return
	}
	u, err = func() (entities.User, error) {
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
			user, err = storage.GetUserByUserName(ctx, tx, username)
			if err != nil {
				return user, err
			}
			return user, tx.Commit()
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
	return u, err
}

func (a *AuthService) localAuth(req bunrouter.Request) (u entities.User, err error) {
	incoming := req.Header.Get("X-API-KEY")
	if len(incoming) == 0 {
		err = ErrUnauthorized
		return
	}
	u, err = storage.GetUserByApiKey(req.Context(), a.db, incoming)
	if err != nil {
		err = ErrUnauthorized
	}
	return
}

func (a *AuthService) InternalApi(next bunrouter.HandlerFunc) bunrouter.HandlerFunc {
	return func(w http.ResponseWriter, req bunrouter.Request) error {
		incoming := req.Header.Get("X-API-KEY")
		if incoming != a.internalApiKey {
			return toJSON(w, http.StatusUnauthorized, nil)
		}
		err := next(w, req)
		return err
	}
}

func (a *AuthService) Signup(ctx context.Context, username string) (string, error) {
	apiKey := cryptoutils.XApiKey()
	u := entities.User{
		Username:  username,
		ApiKey:    apiKey,
		CreatedAt: time.Now(),
	}
	_, err := storage.InsertUser(ctx, a.db, u)
	return u.ApiKey, err
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
