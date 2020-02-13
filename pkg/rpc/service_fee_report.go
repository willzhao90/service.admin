package rpc

import (
	log "github.com/sirupsen/logrus"
	exmongo "gitlab.com/sdce/exlib/mongo"
	pb "gitlab.com/sdce/protogo"
	"gitlab.com/sdce/service/admin/pkg/repository"
	"golang.org/x/net/context"
)

func (a AdminServer) DoCreateMonthlyFeeReport(ctx context.Context, in *pb.CreateMonthlyFeeReportRequest) (out *pb.CreateMonthlyFeeReportResponse, err error) {
	go func() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		a.maker.ConstructMonthlyFeeReport(ctx, in.TimePoint)
	}()
	// if err != nil {
	// 	log.Errorf("Fail to create monthly fee report: %v", err)
	// 	err = exmongo.ErrorToRpcError(err)
	// 	return
	// }
	out = &pb.CreateMonthlyFeeReportResponse{
		Id: nil,
	}
	return
}

func (a AdminServer) DoGetMonthlyFeeReport(ctx context.Context, in *pb.GetMonthlyFeeReportRequest) (out *pb.GetMonthlyFeeReportResponse, err error) {
	monthlyfeeReport, err := a.feeReport.GetFeeReport(ctx, in.Id)
	if err != nil {
		log.Errorf("Fail to get monthly fee report: %v", err)
		err = exmongo.ErrorToRpcError(err)
		return
	}
	out = &pb.GetMonthlyFeeReportResponse{
		MonthlyFeeReport: monthlyfeeReport,
	}
	return
}

func (a AdminServer) DoSearchMonthlyFeeReport(ctx context.Context, in *pb.SearchMonthlyFeeReportRequest) (out *pb.SearchMonthlyFeeReportResponse, err error) {
	filter := &repository.FeeReportFilter{
		BeforeTime: in.BeforeTime,
		AfterTime:  in.AfterTime,
	}
	if in.GetPaging() != nil {
		filter.PageIdx = in.GetPaging().GetPageIndex()
		filter.PageSize = in.GetPaging().GetPageSize()
	}
	reports, count, err := a.feeReport.SearchFeeReport(ctx, filter)
	if err != nil {
		log.Errorf("", err)
		err = exmongo.ErrorToRpcError(err)
		return
	}
	out = &pb.SearchMonthlyFeeReportResponse{
		MonthlyFeeReports: reports,
		ResultCount:       count,
	}
	return
}
