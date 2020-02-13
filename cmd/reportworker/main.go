package main

import (
	"context"
	"sync"

	"gitlab.com/sdce/exlib/alicloud"
	"gitlab.com/sdce/exlib/blob"
	"gitlab.com/sdce/exlib/mongo"
	"gitlab.com/sdce/exlib/service"
	"gitlab.com/sdce/service/admin/pkg/api"
	"gitlab.com/sdce/service/admin/pkg/reportworker"

	"time"

	log "github.com/sirupsen/logrus"
	"gitlab.com/sdce/exlib/config"
)

const (
	serviceName = "service.admin"
)

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

func main() {
	log.SetReportCaller(true)
	log.SetFormatter(&log.JSONFormatter{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel() // Just in case

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

	reportWorker := reportworker.NewReportService(*apiClient, store, db)
	//expireWorker := report.NewExpireCheckService(otcapi, db)

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		err := reportWorker.Run(ctx)
		log.Errorf("service exited: %v", err)
	}()

	wg.Wait()
	log.Warning("Server has exited.")
}
