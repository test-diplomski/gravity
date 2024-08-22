package servers

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/c12s/agent_queue/pkg/api"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type agentQueueServer struct {
	api.UnimplementedAgentQueueServer
	natsConn *nats.Conn
}

func (s *agentQueueServer) DisseminateAppConfig(ctx context.Context, in *api.DeseminateConfigRequest) (*api.DeseminateConfigResponse, error) {
	log.Println("[DeseminateAppConfig]: Endpoint execution.")
	nodeId, err := uuid.Parse(in.NodeId)
	if err != nil {
		log.Printf("[DeseminateAppConfig] Error: %s is not a valid UUID", in.NodeId)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	err = s.natsConn.Publish(fmt.Sprintf("%s.app_config", &nodeId), in.Config)
	if err != nil {
		log.Printf("[DeseminateAppConfig] Error while publishing msg to nats. %v", err)
		return nil, status.Errorf(codes.Aborted, "Could not publish message to nats queue")
	}
	return &api.DeseminateConfigResponse{}, nil
}

func (s *agentQueueServer) JoinCluster(ctx context.Context, in *api.JoinClusterRequest) (*api.JoinClusterResponse, error) {
	log.Println("[JoinCluster]: Endpoint execution.")
	nodeId, err := uuid.Parse(in.NodeId)
	if err != nil {
		log.Printf("[JoinCluster] Error: %s is not a valid UUID", in.NodeId)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	err = s.natsConn.Publish(fmt.Sprintf("%s.join", &nodeId), []byte(fmt.Sprintf("%s|%s", in.JoinAddress, in.ClusterId)))
	if err != nil {
		log.Printf("[JoinCluster] Error while publishing msg to nats. %v", err)
		return nil, status.Errorf(codes.Aborted, "Could not publish message to nats queue")
	}
	return &api.JoinClusterResponse{}, nil
}

func (s *agentQueueServer) DeseminateConfig(ctx context.Context, in *api.DeseminateConfigRequest) (*api.DeseminateConfigResponse, error) {
	log.Println("[DeseminateConfig]: Endpoint execution.")
	nodeId, err := uuid.Parse(in.NodeId)
	if err != nil {
		log.Printf("[DeseminateConfig] Error: %s is not a valid UUID", in.NodeId)
		return nil, status.Errorf(codes.InvalidArgument, err.Error())
	}
	replySubject := fmt.Sprintf("%s/%s", in.NodeId, uuid.New().String())
	subsctiption, err := s.natsConn.Subscribe(replySubject, func(msg *nats.Msg) {
		_, err := http.Post(in.Webhook, "application/protobuf", bytes.NewBuffer(msg.Data))
		if err != nil {
			log.Printf("[DeseminateConfig] Error: Response from webhook %s: %s", in.Webhook, err.Error())
		}
	})
	if err != nil {
		log.Printf("[DeseminateConfig] Error: Could not subscribe to the reply subject %s", replySubject)
		return nil, status.Errorf(codes.Internal, err.Error())
	}
	go func() {
		time.Sleep(10 * time.Second)
		if subsctiption.IsValid() {
			subsctiption.Unsubscribe()
		}
	}()

	err = s.natsConn.PublishRequest(fmt.Sprintf("%s.configs", &nodeId), replySubject, in.Config)
	if err != nil {
		log.Printf("[DeseminateConfig] Error while publishing config to nats. %v", err)
		return nil, status.Errorf(codes.Aborted, "Could not publish message to nats queue")
	}
	return &api.DeseminateConfigResponse{}, nil
}

func Serve(natsConn *nats.Conn, grpcPort int) {
	log.Println("Starting grpc server...")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", grpcPort))
	if err != nil {
		panic(err)
	}

	server := grpc.NewServer()
	api.RegisterAgentQueueServer(server, &agentQueueServer{natsConn: natsConn})
	reflection.Register(server)
	if err := server.Serve(listener); err != nil {
		log.Fatalf("Failed to start grpc server: %v", err)
	}
}
