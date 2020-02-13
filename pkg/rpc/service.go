package rpc

import (
	exmongo "gitlab.com/sdce/exlib/mongo"
	"gitlab.com/sdce/service/admin/pkg/repository"
	"gitlab.com/sdce/service/admin/pkg/reportmaker"
	"gitlab.com/sdce/service/admin/pkg/api"
	"gitlab.com/sdce/exlib/blob"
)

type AdminServer struct {
	feeReport repository.FeeReportRepository
	maker reportmaker.ReportMaker
}

const (
	port = ":8030"
)

func NewAdminServer(api api.Server, store blob.BlobStore, db *exmongo.Database) *AdminServer {
	maker := reportmaker.NewReportService(api, store, db)
	return &AdminServer{
		feeReport: repository.NewFeeReportRepo(db),
		maker: maker,
	}
}
