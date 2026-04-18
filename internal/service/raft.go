package service

import (
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

func NewRaftNode(raftAddr string, fsm raft.FSM, isBootstrap bool) (*raft.Raft, error) {
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

	add, _ := net.ResolveTCPAddr("tcp", raftAddr)

	transport, err := raft.NewTCPTransport(raftAddr, add, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return nil, err
	}

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
