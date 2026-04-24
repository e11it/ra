package filepath

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"
)

// Хранит путь до файла и последнюю известную контрольную сумму
type FileWithChecksum struct {
	configPath     string
	configChecksum string
}

func NewFileWithChecksum(path string) (*FileWithChecksum, error) {
	var (
		cm  *FileWithChecksum
		err error
	)
	cm = new(FileWithChecksum)
	cm.configPath = path
	cm.configChecksum, err = fileChecksum(path)
	if err != nil {
		return nil, err
	}
	return cm, err
}

func (fc *FileWithChecksum) Path() string {
	return fc.configPath
}

// Контрольная сумма обновляется, если она успешно
// посчитана и отличается от хранимой.
// В случае ошибки возвращается true и ошибка
func (cm *FileWithChecksum) IsConfigFileChanged() (bool, error) {
	var (
		newChecksum string
		err         error
	)

	newChecksum, err = fileChecksum(cm.configPath)
	if err != nil {
		return true, err
	}
	if newChecksum != cm.configChecksum {
		cm.configChecksum = newChecksum
		return true, nil
	}
	return false, nil
}

// fileChecksum computes content checksum for change detection.
func fileChecksum(filePath string) (string, error) {
	var (
		fileHash hash.Hash
		file     *os.File
		err      error
	)
	fileHash = sha256.New()
	cleanPath := filepath.Clean(filePath)
	// #nosec G304 -- path comes from configured RA_CONFIG_FILE and is expected to be user-provided.
	file, err = os.Open(cleanPath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	_, err = io.Copy(fileHash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(fileHash.Sum(nil)), nil
}
