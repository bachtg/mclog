package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/bachtg/mclog/api"
	"github.com/bachtg/mclog/internal/service"
	"github.com/bachtg/mclog/internal/tls"
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

// args: [program, "server", grpcPort, raftPort, "bootstrap"/"not-bootstrap", serfPort, serfPortBootstrap]
func startServer(args []string) error {
	grpcAddress := fmt.Sprintf(":%s", args[2])

	// init new service
	sv, err := service.NewService(fmt.Sprintf("./data/%s.log", args[2]))
	if err != nil {
		return err
	}

	// init Raft node
	raftAddr := fmt.Sprintf("localhost:%s", args[3])
	tlsConfig, err := tls.NewTLS(&tls.TLSConfig{
		CertFile:   "./authority/certs/server.pem",
		KeyFile:    "./authority/certs/server-key.pem",
		CAFile:     "./authority/certs/ca.pem",
		ServerAddr: "localhost",
		IsServer:   true,
	})
	if err != nil {
		return err
	}

	tlsStream, err := service.NewTLSStreamLayer(raftAddr, tlsConfig)
	if err != nil {
		return err
	}
	defer tlsStream.Close()

	raftNode, err := service.NewRaftNode(raftAddr, sv, args[4] == "bootstrap", tlsStream)
	if err != nil {
		return err
	}
	sv.EmbedRaft(raftNode, raftAddr)

	// init Serf node
	serfAddr := fmt.Sprintf("localhost:%s", args[5])
	tags := map[string]string{
		"raft_addr": raftAddr,
		"grpc_port": grpcAddress,
	}

	nodeDiscoverer, err := service.NewNodeDiscoverer(serfAddr, tags, args[6:])
	if err != nil {
		return err
	}
	sv.EmbedNodeDiscoverer(nodeDiscoverer)

	// start gRPC server
	conn, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		return err
	}
	defer conn.Close()

	grpcServer := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(tlsConfig)),
	)

	api.RegisterLogServiceServer(grpcServer, sv)
	fmt.Println("gRPC server listening on", grpcAddress)

	if err := grpcServer.Serve(conn); err != nil {
		return err
	}
	return nil
}

func startClient(args []string) error {
	var (
		serverPort = args[2]
		command    = args[3]
		serverAddr = fmt.Sprintf("localhost:%s", serverPort)
	)

	tlsConfig, err := tls.NewTLS(&tls.TLSConfig{
		CertFile:   "./authority/certs/client.pem",
		KeyFile:    "./authority/certs/client-key.pem",
		CAFile:     "./authority/certs/ca.pem",
		ServerAddr: serverAddr,
		IsServer:   false,
	})
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))
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
