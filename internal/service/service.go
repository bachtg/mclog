package service

import (
	"context"
	"fmt"

	"github.com/hashicorp/raft"

	"github.com/bachtg/mclog/api"
	"github.com/bachtg/mclog/internal/log"
)

type Service struct {
	api.UnimplementedLogServiceServer

	logStore *log.Log
	raftNode *raft.Raft
	raftAddr string
}

func NewService(dataDir string) (*Service, error) {

	logStore, err := log.NewLog(dataDir)
	if err != nil {
		return nil, err
	}

	return &Service{
		logStore: logStore,
	}, nil
}

func (s *Service) Produce(ctx context.Context, request *api.ProduceRequest) (*api.ProduceResponse, error) {
	fmt.Printf("Received Produce request with value: %s\n", string(request.Value))

	offset, err := s.logStore.Append(request.Value)
	if err != nil {
		return nil, err
	}

	return &api.ProduceResponse{
		Offset: offset,
	}, nil
}

func (s *Service) Consume(ctx context.Context, request *api.ConsumeRequest) (*api.ConsumeResponse, error) {
	fmt.Printf("Received Consume request with offset: %d\n", request.Offset)

	value, err := s.logStore.Read(request.Offset)
	if err != nil {
		return nil, err
	}

	return &api.ConsumeResponse{
		Value: value,
	}, nil
}
