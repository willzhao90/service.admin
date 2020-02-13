package main

import (
	"context"
	"net"
	"os"
	"sync"

	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/sdce/exlib/alicloud"
	"gitlab.com/sdce/exlib/blob"
	"gitlab.com/sdce/exlib/config"
	"gitlab.com/sdce/exlib/mongo"
	"gitlab.com/sdce/exlib/service"
	pb "gitlab.com/sdce/protogo"
	"gitlab.com/sdce/service/admin/pkg/api"
	"gitlab.com/sdce/service/admin/pkg/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	port        = "0.0.0.0:8030"
	serviceName = "service.admin"
)

type Service interface {
	Run(ctx context.Context)
}

type AdminService struct {
	rpc    *rpc.AdminServer
	health *health.Server
	db     *mongo.Database
}

func getConfigs() (aliCfg alicloud.AliConfig, svcConf service.Config, mongoConf mongo.Config, err error) {
	v, err := config.LoadConfig(serviceName)
	if err != nil {
		log.Errorf("Failed to load configs: %v", err)
		return
	}

	aliCfg, err = alicloud.GetConfig(v)
	if err != nil {
		return
	}

	svcConf, err = service.GetConfig(v)
	if err != nil {
		return
	}

	mongoConf, err = mongo.GetConfig(v)
	if err != nil {
		return
	}
	return
}

func envOrDefaultString(envVar string, defaultValue string) string {
	value := os.Getenv(envVar)
	if value == "" {
		return defaultValue
	}

	return value
}

func (s *AdminService) Run(ctx context.Context) {
	lis, err := net.Listen("tcp", envOrDefaultString("admin_rpc:server:port", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	gs := grpc.NewServer()
	pb.RegisterAdminServiceServer(gs, s.rpc)
	grpc_health_v1.RegisterHealthServer(gs, s.health)
	s.health.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)
	// Register reflection service on gRPC server.
	reflection.Register(gs)

	go func() {
		select {
		case <-ctx.Done():
			gs.GracefulStop()
		}
	}()

	log.Infof("Listening at %v...\n", port)
	if err := gs.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func main() {
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})

	config.LoadConfig("service.admin")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	aliCfg, svcConf, mgoConf, err := getConfigs()
	if err != nil {
		log.Fatalf("Cannot read config: %v", err)
	}
	store, err := blob.NewOssBlobStore(&aliCfg, 1*time.Hour)
	if err != nil {
		log.Fatalf("Cannot connect ali cloud: %v", err)
	}

	apiClient, err := api.New(&svcConf)
	if err != nil {
		log.Fatal("failed to create api ")
	}
	db := mongo.Connect(ctx, mgoConf)
	defer db.Close(ctx)

	adminServer := rpc.NewAdminServer(*apiClient, store, db)
	admin := &AdminService{
		rpc:    adminServer,
		db:     db,
		health: health.NewServer(),
	}

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		admin.Run(ctx)
	}()
	wg.Wait()
}
