package ra

import (
	"context"

	"github.com/e11it/ra/pkg/filepath"

	"github.com/jinzhu/configor"
)

type Ra struct {
	ctx     context.Context
	cfgPath *filepath.FileWithChecksum
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
	return ra, nil
}

func (ra *Ra) loadConfig() error {
	cfg := new(Config)
	err := configor.New(
		&configor.Config{Verbose: false}).
		Load(cfg, ra.cfgPath.Path())
	if err != nil {
		return err
	}
	return nil
}

func (ra *Ra) ReloadHandler() {
}
