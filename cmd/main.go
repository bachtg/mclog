package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"

	"github.com/bachtg/mclog/api"
	"github.com/bachtg/mclog/internal/service"
)

func main() {
	err := start(os.Args)
	if err != nil {
		panic(err)
	}
}

func start(args []string) error {
	switch args[1] {
	case "server":
		return startServer(args)
	case "client":
		return startClient(args)
	default:
		return nil
	}
}

func startServer(args []string) error {
	conn, err := net.Listen("tcp", fmt.Sprintf(":%s", args[2]))
	if err != nil {
		return err
	}
	defer conn.Close()

	grpcServer := grpc.NewServer()

	service, err := service.NewService("./data/data.log")
	if err != nil {
		return err
	}

	api.RegisterLogServiceServer(grpcServer, service)
	fmt.Println("gRPC server listening on", fmt.Sprintf(":%s", args[2]))

	if err := grpcServer.Serve(conn); err != nil {
		return err
	}
	return nil
}

func startClient(args []string) error {
	var (
		serverPort = args[2]
		command    = args[3]
	)

	conn, err := grpc.Dial("localhost:"+serverPort, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := api.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch command {
	case "produce":
		if args[4] == "" {
			return fmt.Errorf("value is required for produce command")
		}
		produceReq := &api.ProduceRequest{
			Value: []byte(args[4]),
		}
		produceRes, err := client.Produce(ctx, produceReq)
		if err != nil {
			return err
		}
		println("Produced record at offset", produceRes.Offset)

	case "consume":
		if args[4] == "" {
			return fmt.Errorf("offset is required for consume command")
		}

		offset, err := strconv.ParseUint(args[4], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid offset: %v", err)
		}

		consumeReq := &api.ConsumeRequest{
			Offset: offset,
		}
		consumeRes, err := client.Consume(ctx, consumeReq)
		if err != nil {
			return err
		}
		println("Consumed record value:", string(consumeRes.Value))

	default:
		return fmt.Errorf("unknown command: %s", command)
	}

	return nil
}
