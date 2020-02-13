package repository

import (
	"context"

	exmongo "gitlab.com/sdce/exlib/mongo"
	pb "gitlab.com/sdce/protogo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	FeeReportCollection = "fee_report"
)

type FeeReportRepository interface {
	CreateFeeReport(ctx context.Context, data *pb.MonthlyFeeReport) (*pb.UUID, error)
	GetFeeReport(ctx context.Context, id *pb.UUID) (out *pb.MonthlyFeeReport, err error)
	SearchFeeReport(ctx context.Context, filter *FeeReportFilter) (out []*pb.MonthlyFeeReport, count int64, err error)
}

type feeReportRepo struct {
	FeeReport *mongo.Collection
}

type FeeReportFilter struct {
	AfterTime  int64
	BeforeTime int64
	PageIdx    int64
	PageSize   int64
}

func NewFeeReportRepo(db *exmongo.Database) FeeReportRepository {
	return &feeReportRepo{
		FeeReport: db.CreateCollection(FeeReportCollection),
	}
}

func (m *feeReportRepo) CreateFeeReport(ctx context.Context, data *pb.MonthlyFeeReport) (*pb.UUID, error) {
	//data.Id = exutil.NewUUID()

	//upsert
	_, err := m.FeeReport.UpdateOne(ctx, bson.M{"name": data.Name}, bson.M{"$setOnInsert": data}, options.Update().SetUpsert(true))
	//res, err := m.FeeReport.InsertOne(ctx, data)
	if err != nil {
		return nil, err
	}
	//insertedID := res.InsertedID.(primitive.ObjectID)
	//return &pb.UUID{Bytes: insertedID[:]}, nil
	return nil, nil
}

func (m *feeReportRepo) GetFeeReport(ctx context.Context, id *pb.UUID) (out *pb.MonthlyFeeReport, err error) {
	err = m.FeeReport.FindOne(ctx, exmongo.IDFilter(id)).Decode(&out)
	return
}

func (m *feeReportRepo) SearchFeeReport(ctx context.Context, filter *FeeReportFilter) (out []*pb.MonthlyFeeReport, count int64, err error) {
	opts := &options.FindOptions{}
	if filter.PageSize > 0 {
		opts = exmongo.NewPaginationOptions(filter.PageIdx, filter.PageSize)
	}
	fobj := bson.M{}
	opts.SetSort(bson.M{"createdAt": -1}) // Time descending order
	if filter.AfterTime != 0 {
		fobj["createdAt"] = bson.M{"$gte": filter.AfterTime}
	}
	if filter.BeforeTime != 0 {
		fobj["createdAt"] = bson.M{"$lt": filter.BeforeTime}
	}
	cur, err := m.FeeReport.Find(ctx, fobj, opts)
	if err != nil {
		return nil, 0, err
	}
	defer cur.Close(ctx)
	count, err = m.FeeReport.CountDocuments(ctx, fobj)
	if err != nil {
		return nil, 0, err
	}
	err = exmongo.DecodeCursorToSlice(ctx, cur, &out)
	return
}
