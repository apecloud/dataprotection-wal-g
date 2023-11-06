package functests

import (
	"fmt"
	"math/rand"
	"os"
	"path"
	"strconv"

	"github.com/apecloud/dataprotection-wal-g/tests_func/utils"
)

const (
	EnvDirPerm  os.FileMode = 0755
	EnvFilePerv os.FileMode = 0644
)

func EnvExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func SetupNewEnv(fromEnv map[string]string, envFilePath, stagingDir string) (map[string]string, error) {
	if _, err := os.Stat(stagingDir); err == nil {
		err = os.Chmod(stagingDir, EnvDirPerm)
		if err != nil {
			return nil, fmt.Errorf("can not chmod staging dir: %v", err)
		}
	} else if err := os.Mkdir(stagingDir, EnvDirPerm); err != nil {
		return nil, fmt.Errorf("can not create staging dir: %v", err)
	}
	env := utils.MergeEnvs(fromEnv, DynConf(fromEnv))
	file, err := os.OpenFile(envFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, EnvFilePerv)
	if err != nil {
		return nil, fmt.Errorf("can not open database file for writing: %v", err)
	}
	defer func() { _ = file.Close() }()

	if err := utils.WriteEnv(env, file); err != nil {
		return nil, fmt.Errorf("can not write to database file: %v", err)
	}

	return env, nil
}

func ReadEnv(path string) (map[string]string, error) {
	file, err := os.OpenFile(path, os.O_RDONLY, EnvFilePerv)
	if err != nil {
		return nil, fmt.Errorf("can not open database file: %v", err)
	}
	defer func() { _ = file.Close() }()
	envLines, err := utils.ReadLines(file)
	if err != nil {
		return nil, err
	}
	return utils.ParseEnvLines(envLines), nil
}

func SetupStaging(imagesDir, stagingDir string) error {
	if err := utils.CopyDirectory(imagesDir, path.Join(stagingDir, imagesDir), ""); err != nil {
		return fmt.Errorf("can not copy images into staging: %v", err)
	}
	walgDir := path.Join(stagingDir, "wal-g")

	if err := utils.CreateDir(walgDir, 0755); err != nil {
		return err
	}

	if err := utils.CopyDirectory("..", walgDir, "tests_func"); err != nil {
		return fmt.Errorf("can not copy wal-g into staging: %v", err)
	}

	return nil
}

func DynConf(env map[string]string) map[string]string {
	portFactor := env["TEST_ID"]
	netName := fmt.Sprintf("test_net_%s", portFactor)

	return map[string]string{
		"DOCKER_BRIDGE_ID": strconv.Itoa(rand.Intn(65535)),
		"PORT_FACTOR":      portFactor,
		"NETWORK_NAME":     netName,
	}
}
