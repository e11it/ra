package ra

import "github.com/e11it/ra/pkg/auth"

type Config struct {
	APPName  string `default:"app name"`
	Addr     string `default:":8080"`
	LogLevel string `default:""`

	Auth auth.Config

	ShutdownTimeout uint `default:"5"`
}
