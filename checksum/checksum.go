package checksum

import (
	"crypto/md5"
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
)

type checksum struct {
	sum string
}

func NewChecksum(path string) (cs *checksum, err error) {
	cs = new(checksum)
	cs.sum, err = cs.generateCheckSum(path)
	return cs, err
}

func (cs *checksum) GetCheckSum() string {
	return cs.sum
}

func (cs *checksum) generateCheckSum(path string) (string, error) {
	h := md5.New()
	f, err := os.Open(path)
	if err != nil {
		log.WithError(err).Errorf("can't open file %s", path)
		return "", err
	}
	defer f.Close()
	_, err = io.Copy(h, f)
	if err != nil {
		log.WithError(err).Errorf("can't copy file %s", path)
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func (cs *checksum) CompareCheckSum(newPath string) (ok bool) {
	if cs.sum == "" {
		return true
	}

	newSum, _ := cs.generateCheckSum(newPath)
	if cs.sum != newSum {
		cs.sum = newSum
		return false
	}

	return true
}
