package functests

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"time"

	"github.com/apecloud/dataprotection-wal-g/tests_func/helpers"
	"github.com/cucumber/godog"
	"github.com/stretchr/testify/assert"
	"github.com/wal-g/tracelog"
)

func SetupCommonSteps(ctx *godog.ScenarioContext, tctx *TestContext) {
	ctx.Step(`^prepared infrastructure$`, tctx.prepareInfrastructure)
	ctx.Step(`^a configured s3 on ([^\s]*)$`, tctx.configureS3)

	ctx.Step(`we sleep ([^\s]*)$`, tctx.sleep)
}

type TestingfWrap func(format string, args ...interface{})

func (tf TestingfWrap) Errorf(format string, args ...interface{}) {
	tf(format, args)
}

func (tctx *TestContext) sameDataCheck(dataId1, dataId2 string) error {
	if snap1, ok := tctx.AuxData.Snapshots[dataId1]; ok {
		if !assert.NotEmpty(TestingfWrap(tracelog.ErrorLogger.Printf), snap1) {
			return fmt.Errorf("data '%s' snapshot is empty: %+v", dataId1, snap1)
		}
		if snap2, ok := tctx.AuxData.Snapshots[dataId2]; ok {
			if !assert.NotEmpty(TestingfWrap(tracelog.ErrorLogger.Printf), snap2) {
				return fmt.Errorf("data '%s' snapshot is empty: %+v", dataId2, snap2)
			}
			if assert.Equal(TestingfWrap(tracelog.ErrorLogger.Printf), snap1, snap2) {
				return nil
			}
			return fmt.Errorf("same snapshots expected (%s) == (%s)", dataId1, dataId2)
		}
		return fmt.Errorf("no snapshot is saved for with id %s", dataId2)
	}
	return fmt.Errorf("no snapshot is saved for with id %s", dataId1)
}

func (tctx *TestContext) prepareInfrastructure() error {
	tctx.AuxData.CreatedBackupNames = []string{}
	tctx.AuxData.NometaBackupNames = []string{}
	tctx.AuxData.OplogPushEnabled = false
	tctx.AuxData.Timestamps = make(map[string]helpers.OpTimestamp)
	tctx.AuxData.Snapshots = make(map[string][]helpers.NsSnapshot)
	tctx.PreviousBackupTime = time.Unix(0, 0)
	return tctx.Infra.RecreateContainers()
}

func (tctx *TestContext) checkBackupsCount(backupCount int, container string) error {
	walg := WalgUtilFromTestContext(tctx, container)
	backups, err := walg.Backups()
	if err != nil {
		return err
	}
	if len(backups) != backupCount {
		return fmt.Errorf("expected %d number of backups, but found %d", backupCount, len(backups))
	}
	return nil
}

func (tctx *TestContext) purgeBackupRetain(retainCount int, container string) error {
	walg := WalgUtilFromTestContext(tctx, container)
	return walg.PurgeRetain(retainCount)
}

func (tctx *TestContext) deleteMongoBackup(backupNum int, container string) error {
	walg := WalgUtilFromTestContext(tctx, container)
	backups, err := walg.Backups()
	if err != nil {
		return err
	}
	return walg.DeleteBackup(backups[backupNum])
}

func (tctx *TestContext) backupMetadataContains(container string, backupId int, expectedMeta *godog.DocString) error {
	s3client, err := S3StorageFromTestContext(tctx, tctx.S3Host()).Client()
	if err != nil {
		return err
	}

	walg := WalgUtilFromTestContext(tctx, container)
	backups, err := walg.Backups()
	if err != nil {
		return err
	}
	if backupId >= len(backups) {
		return fmt.Errorf("cannot get backup #%d, there are only %d", backupId, len(backups))
	}

	backup := backups[backupId]
	sentinelPath := fmt.Sprintf("%s/basebackups_005/%s_backup_stop_sentinel.json", tctx.Env["WALG_S3_PREFIX"], backup)
	contents, err := s3client.FileContents(sentinelPath)
	if err != nil {
		return err
	}

	expected := helpers.Sentinel{}
	err = json.Unmarshal([]byte(expectedMeta.Content), &expected)
	if err != nil {
		return err
	}

	given := helpers.Sentinel{}
	err = json.Unmarshal(contents, &given)
	if err != nil {
		return err
	}

	if !reflect.DeepEqual(expected, given) {
		return fmt.Errorf("error: expected metadata %v, given %v", expected, given)
	}
	return nil
}

func (tctx *TestContext) configureS3(host string) error {
	return S3StorageFromTestContext(tctx, host).InitMinio()
}

func (tctx *TestContext) getMongoLoadFile(loadId, filename string) string {
	// Mongo configs stored in "mongodb/config"
	return path.Join("mongodb", "config", loadId, filename)
}

func (tctx *TestContext) putEmptyBackupViaMinio(nodeName, filename string) error {
	containerName := fmt.Sprintf("%s.test_net_%s", nodeName, tctx.Env["TEST_ID"])
	backupName := "20010203T040506"
	bucketName := tctx.Env["S3_BUCKET"]
	backupRootDir := tctx.Env["WALG_S3_PREFIX"]
	backupDir := "/export/" + bucketName + "/" + backupRootDir + "/" + backupName
	backupDumpPath := filepath.Join(backupDir, filename)
	tctx.AuxData.NometaBackupNames = append(tctx.AuxData.NometaBackupNames, backupName)
	_, err := helpers.RunCommand(tctx.Context, containerName, []string{"mkdir", "-p", backupDir})
	if err != nil {
		return err
	}
	_, err = helpers.RunCommand(tctx.Context, containerName, []string{"touch", backupDumpPath})
	if err != nil {
		return err
	}
	return nil
}

func (tctx *TestContext) testEmptyBackupsViaMinio(nodeName string) error {
	containerName := fmt.Sprintf("%s.test_net_%s", nodeName, tctx.Env["TEST_ID"])
	bucketName := tctx.Env["S3_BUCKET"]
	backupRootDir := tctx.Env["WALG_S3_PREFIX"]
	backupNames := tctx.AuxData.NometaBackupNames
	for _, backupName := range backupNames {
		backupDir := filepath.Join("/export", bucketName, backupRootDir, backupName)
		_, err := helpers.RunCommand(tctx.Context, containerName, []string{"ls", backupDir})
		if err != nil {
			return err
		}
	}
	return nil
}

func (tctx *TestContext) sleep(duration string) error {
	dur, err := time.ParseDuration(duration)
	if err != nil {
		return err
	}
	time.Sleep(dur)
	return nil
}
