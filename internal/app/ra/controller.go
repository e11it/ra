package ra

import (
	"fmt"
	"maps"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/e11it/ra/pkg/auth"
	"github.com/e11it/ra/pkg/auth/cache"
	"github.com/e11it/ra/pkg/filepath"
	"github.com/e11it/ra/pkg/validate"

	"github.com/jinzhu/configor"
)

type Ra struct {
	cfgPath *filepath.FileWithChecksum
	reload  sync.Mutex
	state   atomic.Pointer[runtimeState]
}

type runtimeState struct {
	config        *Config
	auth          auth.AccessController
	bodyValidator validate.BodyValidator
	identity      *identitySource
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
	config, err := ra.loadConfig()
	if err != nil {
		return nil, err
	}
	state, err := ra.buildRuntimeState(config)
	if err != nil {
		return nil, err
	}
	ra.state.Store(state)
	return ra, nil
}

func (ra *Ra) GetServerAddr() string {
	return ra.currentState().config.Addr
}

func (ra *Ra) ProxyEnabled() bool {
	return ra.currentState().config.Proxy.Enabled
}

func (ra *Ra) GetShutdownTimeout() uint {
	return ra.currentState().config.ShutdownTimeout
}

// AccessLogExcludePaths возвращает пути (без query), для которых [AccessLogMiddleware] не пишет строку access-лога.
// Значения берутся из актуального конфига, в том числе после [Ra.ReloadHandler].
func (ra *Ra) AccessLogExcludePaths() []string {
	return slices.Clone(ra.currentState().config.AccessLog.ExcludePaths)
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

func (ra *Ra) createAccessController(config *Config) (auth.AccessController, error) {
	if config.Cache.Enabled {
		return cache.NewAclWichCache(&config.Auth, config.Cache.CacheSize)
	}
	return auth.NewSimpleAccessController(&config.Auth)
}

func (ra *Ra) buildRuntimeState(config *Config) (*runtimeState, error) {
	identity, err := newIdentitySource(config)
	if err != nil {
		return nil, err
	}
	accessController, err := ra.createAccessController(config)
	if err != nil {
		return nil, fmt.Errorf("create access controller: %w", err)
	}
	bodyValidator, err := createBodyValidator(config.validationConfig())
	if err != nil {
		return nil, fmt.Errorf("create body validator: %w", err)
	}
	return &runtimeState{
		config:        config,
		auth:          accessController,
		bodyValidator: bodyValidator,
		identity:      identity,
	}, nil
}

func (ra *Ra) currentState() *runtimeState {
	if ra == nil {
		return nil
	}
	return ra.state.Load()
}

func startupOnlyConfigChanges(active, candidate *Config) []string {
	changes := make([]string, 0, 4)
	if active.Addr != candidate.Addr {
		changes = append(changes, "addr")
	}
	if active.Proxy.Enabled != candidate.Proxy.Enabled {
		changes = append(changes, "proxy.enabled")
	}
	if active.Proxy.ProxyHost != candidate.Proxy.ProxyHost {
		changes = append(changes, "proxy.proxyhost")
	}
	if !maps.Equal(
		buildAccessLogExcludeSet(active.AccessLog.ExcludePaths),
		buildAccessLogExcludeSet(candidate.AccessLog.ExcludePaths),
	) {
		changes = append(changes, "access_log.exclude_paths")
	}
	return changes
}

// Возращает флаг был ли перезагружен конфиг(или он не изменился)
// и ошибку
func (ra *Ra) ReloadHandler() (bool, error) {
	return ra.reloadConfig(ra.buildRuntimeState)
}

func (ra *Ra) reloadConfig(buildState func(*Config) (*runtimeState, error)) (bool, error) {
	ra.reload.Lock()
	defer ra.reload.Unlock()

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
	if changes := startupOnlyConfigChanges(ra.currentState().config, config); len(changes) > 0 {
		return false, fmt.Errorf(
			"startup-only config fields changed: %s; restart required",
			strings.Join(changes, ", "),
		)
	}
	state, err := buildState(config)
	if err != nil {
		return false, err
	}
	ra.state.Store(state)
	ra.cfgPath.CommitConfigFileChecksum()
	return true, nil
}

// BodyValidator возвращает активный валидатор тела (или nil, если он выключен
// в конфигурации). Используется middleware-слоями Gin.
func (ra *Ra) BodyValidator() validate.BodyValidator {
	return ra.currentState().bodyValidator
}
