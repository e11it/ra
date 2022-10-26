package cache

import (
	"github.com/e11it/ra/pkg/auth"
	lru "github.com/hashicorp/golang-lru"
)

type AccessControllerWithCache struct {
	cache            *lru.ARCCache
	accessController auth.AccessController
}

func NewAclWichCache(config *auth.Config, cacheSize int) (auth.AccessController, error) {
	var err error
	ac := new(AccessControllerWithCache)
	if ac.cache, err = lru.NewARC(cacheSize); err != nil {
		return nil, err
	}
	if ac.accessController, err = auth.NewSimpleAccessController(config); err != nil {
		return nil, err
	}
	return ac, nil
}

func (a *AccessControllerWithCache) Validate(authRequest *auth.AuthRequest) error {
	// TODO: проверить, что сам объект попал в кеш
	if err, ok := a.cache.Get(*authRequest); ok {
		if err == nil {
			return nil
		}
		return err.(error)
	}

	err := a.accessController.Validate(authRequest)
	a.cache.Add(*authRequest, err)

	return err
}
