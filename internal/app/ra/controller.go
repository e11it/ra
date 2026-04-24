package ra

import (
	"context"

	"github.com/e11it/ra/pkg/auth"
	"github.com/e11it/ra/pkg/auth/cache"
	"github.com/e11it/ra/pkg/filepath"
	"github.com/e11it/ra/pkg/validate"

	"github.com/jinzhu/configor"
)

type Ra struct {
	ctx context.Context

	cfgPath *filepath.FileWithChecksum
	config  *Config

	auth          auth.AccessController
	bodyValidator validate.BodyValidator
}

func NewRA(configPath string) (*Ra, error) {
	var (
		ra  *Ra
		err error
	)
	ra = new(Ra)
	ra.cfgPath, err = filepath.NewFileWithChecksum(configPath)
	if err != nil {
		return nil, err
	}
	ra.ctx = context.TODO()
	if ra.config, err = ra.loadConfig(); err != nil {
		return nil, err
	}

	if err = ra.createAccessController(ra.config); err != nil {
		return nil, err
	}
	if ra.bodyValidator, err = createBodyValidator(ra.config.validationConfig()); err != nil {
		return nil, err
	}
	return ra, nil
}

func (ra *Ra) GetServerAddr() string {
	return ra.config.Addr
}

func (ra *Ra) ProxyEnabled() bool {
	return ra.config.Proxy.Enabled
}

func (ra *Ra) GetShutdownTimeout() uint {
	return ra.config.ShutdownTimeout
}

// AccessLogExcludePaths возвращает пути (без query), для которых [AccessLogMiddleware] не пишет строку access-лога.
// Значения берутся из актуального конфига, в том числе после [Ra.ReloadHandler].
func (ra *Ra) AccessLogExcludePaths() []string {
	return ra.config.AccessLog.ExcludePaths
}

func (ra *Ra) loadConfig() (*Config, error) {
	cfg := new(Config)
	err := configor.New(
		&configor.Config{Verbose: false}).
		Load(cfg, ra.cfgPath.Path())
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (ra *Ra) createAccessController(config *Config) (err error) {
	if config.Cache.Enabled {
		// Инициализируем контроллер с кешом
		ra.auth, err = cache.NewAclWichCache(&config.Auth, config.Cache.CacheSize)
		return
	}
	ra.auth, err = auth.NewSimpleAccessController(&ra.config.Auth)
	return
}

// Возращает флаг был ли перезагружен конфиг(или он не изменился)
// и ошибку
func (ra *Ra) ReloadHandler() (bool, error) {
	var config *Config
	isChanged, err := ra.cfgPath.IsConfigFileChanged()
	if err != nil {
		return false, err
	}
	if !isChanged {
		return false, nil
	}
	if config, err = ra.loadConfig(); err != nil {
		return false, err
	}
	if err := ra.createAccessController(config); err != nil {
		return false, err
	}
	validator, err := createBodyValidator(config.validationConfig())
	if err != nil {
		return false, err
	}
	ra.bodyValidator = validator
	ra.config = config
	return true, nil
}

// BodyValidator возвращает активный валидатор тела (или nil, если он выключен
// в конфигурации). Используется middleware-слоями (gin/fiber).
func (ra *Ra) BodyValidator() validate.BodyValidator {
	return ra.bodyValidator
}
