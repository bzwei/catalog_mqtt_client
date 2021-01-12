package request

import (
	context "context"
	"fmt"
	"net"

	"github.com/RedHatInsights/catalog_mqtt_client/internal/catalogtask"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/common"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/logger"
	"github.com/RedHatInsights/catalog_mqtt_client/internal/towerapiworker"
	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

type catalogServerImpl struct {
	config     *common.CatalogConfig
	wokHandler towerapiworker.WorkHandler
	shutdown   chan struct{}
	counter    int
}

// Run is the service method provided by the GRPC server
func (s *catalogServerImpl) Run(ctx context.Context, in *Message) (*Result, error) {
	log.Printf("Received a catalog request: %s", in.URL)

	s.counter++
	nextCtx := logger.CtxWithLoggerID(ctx, s.counter)
	go processRequest(nextCtx, in.URL, s.config, s.wokHandler, catalogtask.MakeCatalogTask(nextCtx, in.URL), &defaultPageWriterFactory{}, s.shutdown)

	return &Result{Ok: true}, nil
}

type grpcListener struct {
	grpcServer *grpc.Server
}

func (lis grpcListener) stop() {
	lis.grpcServer.GracefulStop()
	log.Info("gRPC server stopped")
}

func startGRPCListener(config *common.CatalogConfig, wh towerapiworker.WorkHandler, shutdown chan struct{}) (listener, error) {
	log.Infof("Starting the gRPC server at port %d", config.GRPCPort)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", config.GRPCPort))
	if err != nil {
		log.Fatalf("failed to listen at %d, error: %v", config.GRPCPort, err)
		return nil, err
	}

	server := catalogServerImpl{config: config, wokHandler: wh, shutdown: shutdown}
	grpcServer := grpc.NewServer()
	RegisterCatalogServiceServer(grpcServer, &server)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to server port %d, error: %v", config.GRPCPort, err)
		}
	}()

	return &grpcListener{grpcServer: grpcServer}, nil
}
