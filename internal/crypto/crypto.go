package crypto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/wal-g/tracelog"
)

type GpgKeyExportError struct {
	error
}

func NewGpgKeyExportError(text string) GpgKeyExportError {
	return GpgKeyExportError{errors.Errorf("Got error while exporting gpg key: '%s'", text)}
}

func (err GpgKeyExportError) Error() string {
	return fmt.Sprintf(tracelog.GetErrorFormatter(), err.error)
}

const GpgBin = "gpg"

// CachedKey is the data transfer object describing format of key ring cache
type CachedKey struct {
	KeyID string `json:"keyId"`
	Body  []byte `json:"body"`
}

// TODO : unit tests
// Here we read armored version of Key by calling GPG process
func GetPubRingArmor(keyID string) ([]byte, error) {
	var cache CachedKey
	var cacheFilename string

	usr, err := user.Current()
	if err == nil {
		cacheFilename = filepath.Join(usr.HomeDir, ".walg_key_cache")
		file, err := os.ReadFile(cacheFilename)
		// here we ignore whatever error can occur
		if err == nil {
			err = json.Unmarshal(file, &cache)
			tracelog.ErrorLogger.PrintOnError(err)
			if err == nil && cache.KeyID == keyID && len(cache.Body) > 0 { // don't return an empty cached value
				return cache.Body, nil
			}
		}
	}

	cmd := exec.Command(GpgBin, "-a", "--export", keyID)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	if stderr.Len() > 0 { // gpg -a --export <key-id> reports error on stderr and exits == 0 if the key isn't found
		return nil, NewGpgKeyExportError(strings.TrimSpace(stderr.String()))
	}

	cache.KeyID = keyID
	cache.Body = out
	marshal, err := json.Marshal(&cache)
	if err == nil && len(cacheFilename) > 0 {
		err = os.WriteFile(cacheFilename, marshal, 0644)
		tracelog.ErrorLogger.PrintOnError(err)
	}

	return out, nil
}

func GetSecretRingArmor(keyID string) ([]byte, error) {
	out, err := exec.Command(GpgBin, "-a", "--export-secret-key", keyID).Output()
	if err != nil {
		return nil, err
	}
	return out, nil
}
