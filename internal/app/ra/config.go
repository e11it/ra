package ra

import "github.com/e11it/ra/pkg/auth"

type Config struct {
	APPName  string `default:"app name"`
	Addr     string `default:":8080"`
	LogLevel string `default:""`

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

	Auth auth.Config

	ShutdownTimeout uint `default:"5"`
}
