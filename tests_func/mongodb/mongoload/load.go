package mongoload

import (
	"context"
	"fmt"
	"io"
	"sync"

	"go.mongodb.org/mongo-driver/mongo"

	"github.com/apecloud/dataprotection-wal-g/tests_func/mongodb/mongoload/internal"
	"github.com/apecloud/dataprotection-wal-g/tests_func/mongodb/mongoload/models"
)

func HandleLoad(ctx context.Context, reader io.Reader, mc *mongo.Client, workers int) (models.LoadStat, error) {
	if workers < 1 {
		return models.LoadStat{}, fmt.Errorf("positive workers count expected")
	}

	ctx, cancel := context.WithCancel(ctx)
	wg := &sync.WaitGroup{}
	defer wg.Wait()
	defer cancel()

	var errs []<-chan error

	rawOpc, errc0, err := internal.ReadRawStage(ctx, reader, 1, wg)
	if err != nil {
		return models.LoadStat{}, err
	}
	errs = append(errs, errc0)

	opsc, errc1 := internal.BuildStage(ctx, mc, rawOpc, wg)
	errs = append(errs, errc1)

	runc := internal.ExecStage(ctx, opsc, workers, workers, wg)
	stat := internal.CollectStat(runc)

	if err := WaitForPipeline(errs...); err != nil {
		return models.LoadStat{}, err
	}

	return stat, nil
}

func WaitForPipeline(errs ...<-chan error) error {
	errc := MergeErrors(errs...)
	for err := range errc {
		if err != nil {
			return err
		}
	}
	return nil
}

func MergeErrors(cs ...<-chan error) <-chan error {
	var wg sync.WaitGroup
	out := make(chan error, len(cs))
	output := func(c <-chan error) {
		for n := range c {
			out <- n
		}
		wg.Done()
	}
	wg.Add(len(cs))
	for _, c := range cs {
		go output(c)
	}
	go func() {
		wg.Wait()
		close(out)
	}()
	return out
}
