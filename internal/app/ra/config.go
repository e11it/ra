package ra

import (
	"github.com/e11it/ra/pkg/auth"
	"github.com/e11it/ra/pkg/validate"
)

type Config struct {
	APPName         string `default:"app name"`
	Addr            string `default:":8080"`
	LogLevel        string `default:""`
	ShutdownTimeout uint   `default:"5"`

	// Из каких headers доставать нужные данные
	Headers struct {
		AuthURL string `default:"X-Original-Uri"`
		IP      string `default:"X-Real-Ip"`
		Method  string `default:"X-Original-Method"`
	}
	Cache struct {
		Enabled   bool `default:"true"`
		CacheSize int  `default:"1000"`
	}

	Proxy struct {
		Enabled   bool `default:"false"`
		ProxyHost string
	}
	TrimURLPrefix string

	Auth auth.Config

	// BodyValidation — проверка тела POST /topics/{topic} для Kafka REST v2.
	// Подробнее см. pkg/payloadvalidate и docs/Корпоративный стандарт ...md.
	BodyValidation struct {
		Enabled           bool     `yaml:"enabled" default:"false"`
		AllowedOperations []string `yaml:"allowed_operations" default:"[CREATE,UPDATE,UPSERT,DELETE,SNAPSHOT,EVENT]"`
		Checks            []string `yaml:"checks" default:"[no_partition,is_tombstone,envelope,payload,entity_key]"`
	} `yaml:"body_validation"`

	// AccessLog — фильтрация access-лога (Gin), чтобы технические endpoint'ы не флодили JSON-лог.
	AccessLog struct {
		ExcludePaths []string `yaml:"exclude_paths" default:"[/metrics,/health,/ready,/api/openapi,/api/openapi/ra.yaml]"`
	} `yaml:"access_log"`
}

// validationConfig адаптирует корневой BodyValidation-блок к validate.Config.
func (c *Config) validationConfig() validate.Config {
	return validate.Config{
		Enabled: c.BodyValidation.Enabled,
		Checks:  c.BodyValidation.Checks,
		StringLists: map[string][]string{
			"allowed_operations": c.BodyValidation.AllowedOperations,
		},
	}
}
