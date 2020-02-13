package reportmaker

import (
	"context"
	"strings"
	"time"

	"os"

	"github.com/jinzhu/now"
	log "github.com/sirupsen/logrus"
	"gitlab.com/sdce/exlib/blob"
	"gitlab.com/sdce/exlib/exutil"
	exmongo "gitlab.com/sdce/exlib/mongo"
	pb "gitlab.com/sdce/protogo"
	"gitlab.com/sdce/service/admin/pkg/api"
	"gitlab.com/sdce/service/admin/pkg/repository"
)

type ReportMaker interface {
	ConstructMonthlyFeeReport(ctx context.Context, timePoint int64) (id *pb.UUID, err error)
}

type reportManager struct {
	apis   api.Server
	store  blob.BlobStore
	repo   repository.FeeReportRepository
	bucket string
}

func NewReportService(api api.Server, store blob.BlobStore, db *exmongo.Database) ReportMaker {
	return &reportManager{
		apis:  api,
		store: store,
		repo:  repository.NewFeeReportRepo(db),
	}
}

func (rm *reportManager) ConstructMonthlyFeeReport(ctx context.Context, timePoint int64) (id *pb.UUID, err error) {
	//1 search fee
	//2 to csv & upload
	//3 rpc create
	var t time.Time
	if timePoint == 0 {
		t = time.Now()
	} else {
		t = time.Unix(0, timePoint)
	}

	env := os.Getenv("ADMIN_ENV")
	folder := "dev_"
	if strings.EqualFold(env, "PROD") {
		folder = "prod_"
	}

	firstDayOfLastMonth := now.New(t).BeginningOfMonth().AddDate(0, -1, 0).UnixNano()
	firstDayOfThisMonth := now.New(t).BeginningOfMonth().UnixNano()
	fileName := folder + now.New(t).BeginningOfMonth().AddDate(0, -1, 0).String()[0:7] + "_fee_report.csv"

	log.Infof("Start get fee records...")
	feeDataMap, err := rm.getFeeRecords(ctx, firstDayOfLastMonth, firstDayOfThisMonth)
	if err != nil {
		return
	}
	log.Infof("Start get trade records...")
	tradeDatas, err := rm.getTradeRecords(ctx, firstDayOfLastMonth, firstDayOfThisMonth)
	if err != nil {
		return
	}
	log.Infof("Finish get records...")

	var outData [][]string
	outData = append(outData, []string{"tradeID", "price", "volume", "value", "time", "asker_id", "asker_email", "ask_fee", "askfee_currency", "bider_id", "bider_email", "bid_fee", "bidfee_currency"})
	var tradeID, price, volume, value, tradeTime, askerId, askerEmail, askFee, askfeeCurrency, biderId, biderEmail, bidFee, bidfeeCurrency string
	imf := new(exutil.ImprFloat)
	emailList := map[*pb.UUID]string{}
	for _, tradeData := range tradeDatas {
		decBase, decQuo := int(tradeData.Instrument.Base.Decimal), int(tradeData.Instrument.Quote.Decimal)
		tradeID = exutil.UUIDtoA(tradeData.Id)
		price = imf.FromFloat(tradeData.Price).Shift(decBase - decQuo).ToString()
		err = imf.GetLastErr()
		if err != nil {
			log.Warnf("Failed to convert price, %v", err)
		}

		volume = imf.FromString(tradeData.Volume).Shift(-decBase).ToString()
		err = imf.GetLastErr()
		if err != nil {
			log.Warnf("Failed to convert vol, %v", err)
		}

		value = imf.FromString(tradeData.Value).Shift(-decQuo).ToString()
		err = imf.GetLastErr()
		if err != nil {
			log.Warnf("Failed to convert val, %v", err)
		}
		tradeTime = time.Unix(0, tradeData.Time).String()
		askerId = exutil.UUIDtoA(tradeData.Ask.Owner.ClientId)
		biderId = exutil.UUIDtoA(tradeData.Bid.Owner.ClientId)
		if feeData, ok := feeDataMap[tradeID]; ok {
			for _, feeRecord := range feeData {
				if feeRecord.ToAcc == nil {
					if askerId == exutil.UUIDtoA(feeRecord.FromAcc.MemberId) && feeRecord.Currency.Symbol == tradeData.Instrument.GetBase().Symbol {
						askerEmail = rm.getEmail(ctx, emailList, feeRecord.FromAcc.MemberId)
						askfeeCurrency = feeRecord.Currency.Symbol
						askFee = imf.FromString(feeRecord.Amount).Shift(-int(feeRecord.Currency.Decimal)).ToString()
						err = imf.GetLastErr()
						if err != nil {
							log.Errorf("Failed to convert ask fee, %v", err)
						}
					}
					if biderId == exutil.UUIDtoA(feeRecord.FromAcc.MemberId) && feeRecord.Currency.Symbol == tradeData.Instrument.GetQuote().Symbol {
						biderEmail = rm.getEmail(ctx, emailList, feeRecord.FromAcc.MemberId)
						bidfeeCurrency = feeRecord.Currency.Symbol
						bidFee = imf.FromString(feeRecord.Amount).Shift(-int(feeRecord.Currency.Decimal)).ToString()
						err = imf.GetLastErr()
						if err != nil {
							log.Errorf("Failed to convert bid fee, %v", err)
						}
					}
				}
			}
		}

		outData = append(outData, []string{tradeID, price, volume, value, tradeTime, askerId, askerEmail, askFee, askfeeCurrency, biderId, biderEmail, bidFee, bidfeeCurrency})

	}
	//prod-backend-exchange
	err = rm.constructCSVandUpload("prod-backend-exchange", "financial-report/"+fileName, &outData)
	if err != nil {
		log.Errorf("Fail to Upload report: %v", err)
		return
	}
	log.Info("Finish uploading...")

	monthlyFeeReport := &pb.MonthlyFeeReport{
		Id:        exutil.NewUUID(),
		Name:      fileName,
		Url:       "financial-report/" + fileName,
		CreatedAt: time.Now().UnixNano(),
	}

	id, err = rm.repo.CreateFeeReport(ctx, monthlyFeeReport)
	if err != nil {
		log.Errorf("Fail to create report record: %v", err)
	}

	return
}

func (rm *reportManager) getFeeRecords(ctx context.Context, startTime, endTime int64) (recordMap map[string][]*pb.FeeTransactionRecord, err error) {
	feeReq := &pb.SearchFeeTxRecordsRequest{
		AfterTime:  startTime,
		BeforeTime: endTime,
	}

	pageIdx := 0
	pageSize := 8000
	records := []*pb.FeeTransactionRecord{}
	for {
		feeReq.PageIdx = int64(pageIdx)
		feeReq.PageSize = int64(pageSize)
		feeRes, err := rm.apis.Member.DoSearchTxFeeRecords(ctx, feeReq)
		if err != nil {
			log.Errorf("Fail to get fee records: %v", err)
			return nil, err
		}
		records = append(records, feeRes.Records...)
		if len(feeRes.Records) < 8000 {
			break
		}
		pageIdx++
		time.Sleep(20 * time.Millisecond)
	}
	log.Infof("There are %v fee results found.", len(records))
	feemap := map[string][]*pb.FeeTransactionRecord{}
	for _, record := range records {
		if record.TradeId == nil {
			continue
		}
		tid := exutil.UUIDtoA(record.TradeId)
		if _, ok := feemap[tid]; ok {
			feemap[tid] = append(feemap[tid], record)
		} else {
			feemap[tid] = []*pb.FeeTransactionRecord{record}
		}
	}
	return feemap, nil
}

func (rm *reportManager) getTradeRecords(ctx context.Context, startTime, endTime int64) (records []*pb.TradeDefined, err error) {
	tradeReq := &pb.FindTradesRequest{
		Start: startTime,
		End:   endTime,
	}

	pageIdx := 0
	pageSize := 8000
	for {
		tradeReq.Paging = &pb.PaginationRequest{
			PageIndex: int64(pageIdx),
			PageSize:  int64(pageSize),
		}
		tradeRes, err := rm.apis.Trading.DoSearchTrades(ctx, tradeReq)
		if err != nil {
			log.Errorf("Fail to get trade records: %v", err)
			return records, err
		}
		records = append(records, tradeRes.Result...)
		if len(tradeRes.Result) < 8000 {
			break
		}
		pageIdx++
		time.Sleep(20 * time.Millisecond)
	}
	log.Infof("There are totally: %v trade results found.", len(records))
	return
}

func (rm *reportManager) getEmailFromMember(ctx context.Context, memberID *pb.UUID) (email string, err error) {
	res, err := rm.apis.Member.DoFindMember(ctx, &pb.FindMemberRequest{
		MemberId: memberID,
	})
	if res != nil {
		return res.GetMemberDefined().GetContact().GetEmail(), err
	}
	return
}

func (rm *reportManager) getEmail(ctx context.Context, emailList map[*pb.UUID]string, memberID *pb.UUID) string {
	if email, ok := emailList[memberID]; ok {
		return email
	} else {
		email, err := rm.getEmailFromMember(ctx, memberID)
		if err != nil {
			log.Errorf("find member error, %v", err)
		}
		emailList[memberID] = email
		return email
	}
}
