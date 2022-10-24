package filepath

import (
	"crypto/md5"
	"encoding/hex"
	"hash"
	"io"
	"os"
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
	cm.configChecksum, err = fileMd5Checksum(path)
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

	newChecksum, err = fileMd5Checksum(cm.configPath)
	if err != nil {
		return true, err
	}
	if newChecksum != cm.configChecksum {
		cm.configChecksum = newChecksum
		return true, nil
	}
	return false, nil
}

// Функтция вычисляет md5 сумму файла или возвращает ошибку
func fileMd5Checksum(filePath string) (string, error) {
	var (
		fileHash hash.Hash
		file     *os.File
		err      error
	)
	// #nosec
	fileHash = md5.New()
	file, err = os.Open(filePath)
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
