package auth

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	lru "github.com/hashicorp/golang-lru"
	log "github.com/sirupsen/logrus"
)

type authCheck func(c *gin.Context) error

type auth struct {
	cache  **lru.ARCCache
	checks *[]authCheck
}

func (a *auth) addCheck(ac authCheck) {
	if ac != nil {
		*a.checks = append(*a.checks, ac)
	}
}

func NewAuth(c *Config) (*auth, error) {
	a := new(auth)
	cache, err := lru.NewARC(c.CacheSize)
	if err != nil {
		log.Println(c.CacheSize, c.Headers.Method)
		return nil, err
	}
	a.cache = &cache
	a.makeChecks(c)

	return a, err
}

func (a *auth) UpdateAuth(c *Config) {
	var err error
	cache, err := lru.NewARC(c.CacheSize)
	if err != nil {
		log.Println(c.CacheSize, c.Headers.Method)
		(*a.cache).Purge()
	}
	a.cache = &cache

	a.makeChecks(c)
}

func (a *auth) getCache() *lru.ARCCache {
	return *a.cache
}

func (a *auth) makeChecks(cfg *Config) {
	var config Config = *cfg
	acl := cfg.getACLCompile()
	checks := make([]authCheck, 0)
	a.checks = &checks
	a.addCheck(a.getHeadersToAttr(&config))
	a.addCheck(a.getAuth())
	a.addCheck(a.getURLValidReg(&config))
	//Last
	//a.addCheck(a.getContentTypeValidReg(cfg))
	a.addCheck(a.getACLVerifier(*acl, a.getCache()))
}

func (a *auth) getChecks() []authCheck {
	return *a.checks
}

func (a *auth) GetMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, f := range a.getChecks() {
			if err := f(c); err != nil {
				log.WithFields(log.Fields{
					"URL":         c.MustGet("AuthURL"),
					"User":        c.MustGet("AuthUser"),
					"ContentType": c.MustGet("ContentType"),
					"IP":          c.MustGet("IP"),
					"Method":      c.MustGet("Method"),
				}).WithError(err).Error()
				c.AbortWithError(http.StatusForbidden, err)
			}
		}
		c.Next()
	}
}

func (a *auth) check(c *gin.Context) {

}

func (a *auth) getAuth() authCheck {
	return func(c *gin.Context) error {
		//
		username, _, authOK := c.Request.BasicAuth()
		if authOK == false {
			username = "anon"
		}
		c.Set("AuthUser", username)

		//if username != "username" || password != "password" {
		//	http.Error(w, "Not authorized", 401)
		//	return
		//}
		return nil
	}
}

func (a *auth) getHeadersToAttr(cfg *Config) authCheck {
	return func(c *gin.Context) error {
		c.Set("AuthURL", strings.TrimPrefix(c.GetHeader(cfg.Headers.AuthURL), cfg.Prefix))
		c.Set("ContentType", c.ContentType())
		c.Set("IP", c.GetHeader(cfg.Headers.IP))
		c.Set("Method", c.GetHeader(cfg.Headers.Method))
		return nil
	}
}

func (a *auth) getURLValidReg(cfg *Config) authCheck {
	if len(cfg.URLValidReg) > 0 {

		url_reg := regexp.MustCompile(cfg.URLValidReg)

		return func(c *gin.Context) error {
			if !url_reg.MatchString(c.MustGet("AuthURL").(string)) {
				return fmt.Errorf("Incorrect URL")
			}
			return nil
		}
	}
	return nil
}

func (a *auth) getContentTypeValidReg(cfg *Config) authCheck {
	if len(cfg.ContentTypeValidReg) > 0 {
		ct_reg := regexp.MustCompile(cfg.ContentTypeValidReg)
		return func(c *gin.Context) error {
			if !c.GetBool("AuthValidContext") {
				// check can be made before
				if !ct_reg.MatchString(c.ContentType()) {
					return fmt.Errorf("Incorrect Content-Type")
				}
			}
			return nil
		}
	}
	return nil
}

func (a *auth) getACLVerifier(acl []ACLCompile, cache *lru.ARCCache) authCheck {
	if len(acl) > 0 {

		return func(c *gin.Context) error {
			var err error
			var cacheKey cacheKey

			cacheKey = *getCacheKey(c)
			log.Info(cacheKey)
			if rerr, ok := cache.Get(cacheKey); ok {
				if rerr == nil {
					return nil
				}
				return rerr.(error)
			}

			for _, acl := range acl {
				if acl.IsMatch(c.MustGet("AuthURL").(string)) {
					lerr := acl.IsAllow(c.MustGet("AuthUser").(string), c.MustGet("Method").(string), c.MustGet("ContentType").(string))
					if lerr == nil {
						// ALC allow!
						cache.Add(cacheKey, nil)
						return nil
					}
					err = lerr
				}
			}
			if err == nil {
				err = fmt.Errorf("Permission denied")
				cache.Add(cacheKey, err)
			}
			return err
		}
	}
	return nil
}
