package openpgp

import (
	"bytes"
	"io"
	"os"

	"github.com/ProtonMail/go-crypto/openpgp"
)

func readKey(path string) (io.Reader, error) {
	byteData, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	return bytes.NewReader(byteData), nil
}

func readPGPKey(path string) (openpgp.EntityList, error) {
	gpgKeyReader, err := readKey(path)

	if err != nil {
		return nil, err
	}

	entityList, err := openpgp.ReadArmoredKeyRing(gpgKeyReader)

	if err != nil {
		return nil, err
	}

	return entityList, nil
}

func decryptSecretKey(entityList openpgp.EntityList, passphrase string) error {
	passphraseBytes := []byte(passphrase)

	for _, entity := range entityList {
		err := entity.PrivateKey.Decrypt(passphraseBytes)

		if err != nil {
			return err
		}

		for _, subKey := range entity.Subkeys {
			err := subKey.PrivateKey.Decrypt(passphraseBytes)

			if err != nil {
				return err
			}
		}
	}

	return nil
}
