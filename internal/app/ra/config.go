package ra

import (
	"github.com/e11it/ra/pkg/auth"
	"github.com/e11it/ra/pkg/kafkarest"
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
	// Подробнее см. pkg/kafkarest и docs/Корпоративный стандарт ...md.
	BodyValidation struct {
		Enabled           bool     `yaml:"enabled" default:"false"`
		AllowedOperations []string `yaml:"allowed_operations" default:"[CREATE,UPDATE,UPSERT,DELETE,SNAPSHOT,EVENT]"`
		Checks            []string `yaml:"checks" default:"[entity_key_match,operation_allowed,event_time_zone_valid]"`
	} `yaml:"body_validation"`
}

// kafkaRestConfig адаптирует корневой BodyValidation-блок к kafkarest.Config.
func (c *Config) kafkaRestConfig() kafkarest.Config {
	return kafkarest.Config{
		Enabled:           c.BodyValidation.Enabled,
		AllowedOperations: c.BodyValidation.AllowedOperations,
		Checks:            c.BodyValidation.Checks,
	}
}
