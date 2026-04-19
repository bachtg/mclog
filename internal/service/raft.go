package service

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

func (s *Service) EmbedRaft(raftNode *raft.Raft, raftAddr string) {
	s.raftNode = raftNode
	leaderCh := raftNode.LeaderCh()

	go func() {
		for range leaderCh {
			leader := raftNode.Leader()
			if leader != "" {
				fmt.Printf("[Raft] New leader elected: %s\n", leader)
			} else {
				fmt.Println("[Raft] No leader, election in progress...")
			}
		}
	}()
}

type tlsStreamLayer struct {
	listener  net.Listener
	tlsConfig *tls.Config
}

func (t *tlsStreamLayer) Accept() (net.Conn, error) {
	return t.listener.Accept()
}

func (t *tlsStreamLayer) Close() error {
	return t.listener.Close()
}

func (t *tlsStreamLayer) Addr() net.Addr {
	return t.listener.Addr()
}

func (t *tlsStreamLayer) Dial(address raft.ServerAddress, timeout time.Duration) (net.Conn, error) {
	dialer := &net.Dialer{Timeout: timeout}

	return tls.DialWithDialer(dialer, "tcp", string(address), t.tlsConfig)
}

func NewTLSStreamLayer(addr string, tlsConfig *tls.Config) (raft.StreamLayer, error) {
	listener, err := tls.Listen("tcp", addr, tlsConfig)
	if err != nil {
		return nil, err
	}

	return &tlsStreamLayer{
		listener:  listener,
		tlsConfig: tlsConfig,
	}, nil
}

func NewRaftNode(raftAddr string, fsm raft.FSM, isBootstrap bool, tlsStream raft.StreamLayer) (*raft.Raft, error) {
	config := raft.DefaultConfig()

	// store log entries
	logStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("./data/%s-raft-log.db", raftAddr))
	if err != nil {
		return nil, err
	}

	// store stable state (current term, voted for)
	stableStore, err := raftboltdb.NewBoltStore(fmt.Sprintf("./data/%s-raft-stable.db", raftAddr))
	if err != nil {
		return nil, err
	}

	// store snapshots
	snapshotStore, err := raft.NewFileSnapshotStore(fmt.Sprintf("./data/%s-snapshot.db", raftAddr), 1, os.Stderr)
	if err != nil {
		return nil, err
	}

	config.LocalID = raft.ServerID(raftAddr)
	config.SnapshotInterval = 30 * time.Second
	config.SnapshotThreshold = 2

	transport := raft.NewNetworkTransport(tlsStream, 3, 10*time.Second, nil)
	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return nil, err
	}

	if isBootstrap {
		fmt.Println("[Raft] Bootstrapping cluster by node:", raftAddr)
		raftNode.BootstrapCluster(raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(raftAddr),
					Address: transport.LocalAddr(),
				},
			},
		})
	}

	return raftNode, nil
}
