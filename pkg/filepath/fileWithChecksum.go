package filepath

import (
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sync"
)

// Хранит путь до файла и последнюю известную контрольную сумму
type FileWithChecksum struct {
	mu              sync.Mutex
	configPath      string
	configChecksum  string
	pendingChecksum string
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

// IsConfigFileChanged calculates a candidate checksum without publishing it.
// Call CommitConfigFileChecksum only after the candidate config is active.
func (cm *FileWithChecksum) IsConfigFileChanged() (bool, error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	newChecksum, err := fileChecksum(cm.configPath)
	if err != nil {
		cm.pendingChecksum = ""
		return true, err
	}
	cm.pendingChecksum = newChecksum
	return newChecksum != cm.configChecksum, nil
}

// CommitConfigFileChecksum publishes the last successfully observed checksum.
func (cm *FileWithChecksum) CommitConfigFileChecksum() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.pendingChecksum == "" {
		return
	}
	cm.configChecksum = cm.pendingChecksum
	cm.pendingChecksum = ""
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
	defer func() { _ = file.Close() }()
	_, err = io.Copy(fileHash, file)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(fileHash.Sum(nil)), nil
}
