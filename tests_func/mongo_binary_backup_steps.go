package functests

import (
	"fmt"

	"github.com/apecloud/dataprotection-wal-g/tests_func/helpers"
	"github.com/cucumber/godog"
	"github.com/wal-g/tracelog"
)

func SetupMongodbBinaryBackupSteps(ctx *godog.ScenarioContext, tctx *TestContext) {
	ctx.Step(`^we create binary mongo-backup on ([^\s]*)$`, tctx.createMongoBinaryBackup)
	ctx.Step(`^we restore binary mongo-backup #(\d+) to ([^\s]+)`, tctx.restoreMongoBinaryBackupAsNonInitialized)
	ctx.Step(`^we restore initialized binary mongo-backup #(\d+) to ([^\s]+)`,
		tctx.restoreMongoBinaryBackupAsInitialized)
}

func (tctx *TestContext) createMongoBinaryBackup(container string) error {
	host := tctx.ContainerFQDN(container)

	walg := WalgUtilFromTestContext(tctx, container)
	err := walg.PushBinaryBackup()
	if err != nil {
		return err
	}
	tracelog.DebugLogger.Println("Backup created")

	tctx.PreviousBackupTime, err = helpers.TimeInContainer(tctx.Context, host)
	if err != nil {
		return err
	}

	return nil
}

func (tctx *TestContext) restoreMongoBinaryBackupAsNonInitialized(backupNumber int, container string) error {
	return tctx.restoreMongoBinaryBackup(backupNumber, container, false)
}

func (tctx *TestContext) restoreMongoBinaryBackupAsInitialized(backupNumber int, container string) error {
	return tctx.restoreMongoBinaryBackup(backupNumber, container, true)
}

func (tctx *TestContext) restoreMongoBinaryBackup(backupNumber int, container string, initialized bool) error {
	walg := WalgUtilFromTestContext(tctx, container)

	backup, err := walg.GetBackupByNumber(backupNumber)
	if err != nil {
		return err
	}

	mc, err := MongoCtlFromTestContext(tctx, container)
	if err != nil {
		return err
	}

	mongodbVersion, err := mc.GetVersion()
	if err != nil {
		return err
	}

	configPath, err := mc.GetConfigPath()
	if err != nil {
		return err
	}

	err = mc.StopMongod()
	if err != nil {
		return err
	}

	rsName := ""
	rsMembers := ""
	if initialized {
		rsName = container
		rsMembers = fmt.Sprintf("%s:%d", container, mc.GetMongodPort())
	}
	err = walg.FetchBinaryBackup(backup, configPath, mongodbVersion, rsName, rsMembers)
	if err != nil {
		return err
	}

	if err := mc.ChownDBPath(); err != nil {
		return err
	}

	if err := mc.StartMongod(); err != nil {
		return err
	}

	if !initialized {
		if err := tctx.initiateReplSet(container); err != nil {
			return err
		}
	} else {
		tracelog.DebugLogger.Println("Skip initiateReplSet")
	}

	return nil
}
