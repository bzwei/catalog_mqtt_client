package request

import (
	context "context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/RedHatInsights/rhc-worker-catalog/internal/catalogtask"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/common"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/logger"
	"github.com/RedHatInsights/rhc-worker-catalog/internal/towerapiworker"
	pb "github.com/redhatinsights/yggdrasil/protocol"
	log "github.com/sirupsen/logrus"
	grpc "google.golang.org/grpc"
)

type catalogServerImpl struct {
	pb.UnimplementedWorkerServer
	config     *common.CatalogConfig
	wokHandler towerapiworker.WorkHandler
	shutdown   chan struct{}
}

// Run is the service method provided by the GRPC server
func (s *catalogServerImpl) Send(ctx context.Context, in *pb.Data) (*pb.Receipt, error) {
	log.Printf("Received a catalog request with ID: %s", in.MessageId)

	payload := make(map[string]interface{})
	if err := json.Unmarshal(in.Payload, &payload); err != nil {
		log.Errorf("Failed to parse the payload to a map, error %v", err)
		return nil, err
	}
	urlObj := payload["URL"]
	if urlObj == nil {
		log.Errorf("Payload does not contain an URL")
		return nil, fmt.Errorf("Payload does not contain an URL")
	}
	url := fmt.Sprintf("%v", urlObj)
	nextCtx := logger.CtxWithLoggerID(ctx, in.MessageId)
	logger.GetLogger(nextCtx).Infof("Request payload: %v", payload)
	go processRequest(nextCtx, url, s.config, s.wokHandler, catalogtask.MakeCatalogTask(nextCtx, url), &defaultPageWriterFactory{}, s.shutdown)

	return &pb.Receipt{}, nil
}

type grpcListener struct {
	grpcServer *grpc.Server
}

func (lis grpcListener) stop() {
	lis.grpcServer.GracefulStop()
	log.Info("gRPC server stopped")
}

func registerRHCWorker() string {
	log.Info("Registering catalog worker to RHC")
	// Get initialization values from the environment.
	yggdDispatchSocketAddr, ok := os.LookupEnv("YGG_SOCKET_ADDR")
	if !ok {
		log.Fatal("Missing YGG_SOCKET_ADDR environment variable")
	}

	// Dial the dispatcher on its well-known address.
	conn, err := grpc.Dial(yggdDispatchSocketAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	// Create a dispatcher client
	c := pb.NewDispatcherClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Register as a handler of the "catalog" type.
	r, err := c.Register(ctx, &pb.RegistrationRequest{Handler: "catalog", Pid: int64(os.Getpid())})
	if err != nil {
		log.Fatal(err)
	}
	if !r.GetRegistered() {
		log.Fatalf("handler registration failed: %v", err)
	}
	return r.Address
}

func startGRPCListener(config *common.CatalogConfig, wh towerapiworker.WorkHandler, shutdown chan struct{}) (listener, error) {
	socketAddr := registerRHCWorker()

	log.Infof("Starting the gRPC server at socket %v", socketAddr)
	lis, err := net.Listen("unix", socketAddr)
	if err != nil {
		log.Fatalf("failed to listen at %v, error: %v", socketAddr, err)
	}

	server := catalogServerImpl{config: config, wokHandler: wh, shutdown: shutdown}
	grpcServer := grpc.NewServer()
	pb.RegisterWorkerServer(grpcServer, &server)
	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve socket %v, error: %v", socketAddr, err)
		}
	}()

	return &grpcListener{grpcServer: grpcServer}, nil
}
