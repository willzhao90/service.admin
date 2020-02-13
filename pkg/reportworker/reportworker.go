package reportworker

import (
	"context"
	"fmt"

	"github.com/robfig/cron"
	log "github.com/sirupsen/logrus"
	"gitlab.com/sdce/exlib/blob"
	exmongo "gitlab.com/sdce/exlib/mongo"
	"gitlab.com/sdce/service/admin/pkg/api"
	"gitlab.com/sdce/service/admin/pkg/reportmaker"
)

type ReportWorker interface {
	Run(ctx context.Context) (err error)
}
type reportManager struct {
	maker reportmaker.ReportMaker
}

func NewReportService(api api.Server, store blob.BlobStore, db *exmongo.Database) ReportWorker {
	maker := reportmaker.NewReportService(api, store, db)
	return &reportManager{
		maker: maker,
	}
}

func (rm *reportManager) Run(ctx context.Context) (err error) {
	c := cron.New()
	c.AddFunc("0 0 0 1 * *", func() {
		//@every 5m
		//0 0 1 * *
		log.Info("This is from the fee report cron job every month.")
		_, err = rm.maker.ConstructMonthlyFeeReport(ctx, 0)
		if err != nil {
			log.Errorf("Fail to construct monthly fee report! : %v", err)
		}
	})
	c.Start()
	<-ctx.Done()
	c.Stop()
	return fmt.Errorf("Cron job of fee report stopped unexpectedly.")
}
