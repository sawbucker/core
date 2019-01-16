package auth

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/ShoshinNikita/log"
	"github.com/pkg/errors"
	"github.com/tags-drive/core/internal/params"
)

const maxTokenSize = 30

type Auth struct {
	tokens []tokenStruct // we can use array instead of map because number of tokens is small and O(n) is enough
	mutex  *sync.RWMutex

	logger *log.Logger
}

type tokenStruct struct {
	Token   string    `json:"token"`
	Expires time.Time `json:"expire"`
}

// NewAuthService create new Auth and inits tokens
func NewAuthService(lg *log.Logger) (*Auth, error) {
	service := &Auth{
		mutex:  new(sync.RWMutex),
		logger: lg,
	}

	f, err := os.Open(params.TokensFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, errors.Wrapf(err, "can't open file %s", params.TokensFile)
		}

		// Have to create a new file
		lg.Infof("file %s doesn't exist. Need to create a new file\n", params.TokensFile)
		f, err = os.OpenFile(params.TokensFile, os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			return nil, errors.Wrap(err, "can't create a new file")
		}
		// Write empty structure
		json.NewEncoder(f).Encode(service.tokens)
		// Can exit because we don't need to decode files from the file
		f.Close()
	} else {
		err = json.NewDecoder(f).Decode(&service.tokens)
		if err != nil {
			return nil, errors.Wrap(err, "can't decode allToken.tokens")
		}

		f.Close()
	}

	// Expiration function
	go func() {
		ticker := time.NewTicker(time.Hour * 6)
		for ; true; <-ticker.C {
			lg.Infoln("check expired tokens")
			service.expire()
		}
	}()

	return service, nil
}

// GenerateToken generates a new token
func (a Auth) GenerateToken() string {
	return generate(maxTokenSize)
}

// AddToken adds new generated token
func (a *Auth) AddToken(token string) {
	a.add(token)
}

// DeleteToken deletes token
func (a *Auth) DeleteToken(token string) {
	a.delete(token)
}

// CheckToken return true, if there's a passed token
func (a Auth) CheckToken(token string) bool {
	return a.check(token)
}

func (a Auth) Shutdown() error {
	return nil
}
