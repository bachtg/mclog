package service

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"time"

	"github.com/hashicorp/raft"

	"github.com/bachtg/mclog/api"
	"github.com/bachtg/mclog/internal/log"
)

type Service struct {
	api.UnimplementedLogServiceServer

	logStore       *log.Log
	raftNode       *raft.Raft
	nodeDiscoverer *NodeDiscoverer
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

	if s.raftNode.State() != raft.Leader {
		leaderAddr := s.raftNode.Leader()
		return nil, fmt.Errorf("not leader, please call %s", leaderAddr)
	}

	future := s.raftNode.Apply(request.Value, 5*time.Second)
	if err := future.Error(); err != nil {
		return nil, err
	}

	offset := future.Response().(uint64)

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

func (s *Service) Apply(log *raft.Log) interface{} {
	offset, err := s.logStore.Append(log.Data)
	if err != nil {
		return err
	}

	return offset
}

func (s *Service) Snapshot() (raft.FSMSnapshot, error) {
	data, err := s.logStore.Dump()
	if err != nil {
		return nil, fmt.Errorf("snapshot dump failed: %w", err)
	}

	fmt.Println("Data đã được snap shot -> ", string(data))

	return &snapshot{data: data}, nil
}

func (s *Service) Restore(rc io.ReadCloser) error {
	defer rc.Close()

	sizeBuf := make([]byte, 8)
	if _, err := io.ReadFull(rc, sizeBuf); err != nil {
		return fmt.Errorf("read size prefix: %w", err)
	}
	dataSize := binary.BigEndian.Uint64(sizeBuf)

	data := make([]byte, dataSize)
	if _, err := io.ReadFull(rc, data); err != nil {
		return fmt.Errorf("read snapshot data: %w", err)
	}

	return s.logStore.Load(data)
}
