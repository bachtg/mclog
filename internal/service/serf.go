package service

import (
	"fmt"
	"net"

	"github.com/hashicorp/raft"
	"github.com/hashicorp/serf/serf"
)

type NodeDiscoverer struct {
	serfNode   *serf.Serf
	serfEvents chan serf.Event
	tags       map[string]string
}

func NewNodeDiscoverer(serfAddr string, tags map[string]string, serfSeedPorts []string) (*NodeDiscoverer, error) {
	addr, err := net.ResolveTCPAddr("tcp", serfAddr)
	if err != nil {
		return nil, err
	}

	config := serf.DefaultConfig()
	config.Init()

	config.MemberlistConfig.BindAddr = addr.IP.String()
	config.MemberlistConfig.BindPort = addr.Port
	eventsChan := make(chan serf.Event, 16)
	config.NodeName = serfAddr
	config.EventCh = eventsChan
	config.Tags = tags

	serfNode, err := serf.Create(config)
	if err != nil {
		return nil, err
	}

	if len(serfSeedPorts) > 0 {
		serfSeedNodes := make([]string, len(serfSeedPorts))
		for i, port := range serfSeedPorts {
			serfSeedNodes[i] = fmt.Sprintf("localhost:%s", port)
		}

		_, err := serfNode.Join(serfSeedNodes, true)
		if err != nil {
			return nil, err
		}
	}

	return &NodeDiscoverer{
		serfNode:   serfNode,
		serfEvents: eventsChan,
		tags:       tags,
	}, nil
}

func (s *Service) EmbedNodeDiscoverer(nodeDiscoverer *NodeDiscoverer) {
	s.nodeDiscoverer = nodeDiscoverer
	go s.discoverPeers()
}

func (s *Service) discoverPeers() {
	for e := range s.nodeDiscoverer.serfEvents {
		switch e.EventType() {
		case serf.EventMemberJoin:
			for _, member := range e.(serf.MemberEvent).Members {
				s.handleJoinEvent(member)
			}
		case serf.EventMemberLeave, serf.EventMemberFailed:
			for _, member := range e.(serf.MemberEvent).Members {
				s.handleLeaveEvent(member)
			}
		}
	}
}

func (s *Service) handleJoinEvent(member serf.Member) {
	raftAddr := member.Tags["raft_addr"]
	fmt.Println("[Serf] Node joined:", member.Name, "with Raft address:", raftAddr)

	if s.raftNode.State() != raft.Leader {
		return
	}

	future := s.raftNode.AddVoter(raft.ServerID(raftAddr), raft.ServerAddress(raftAddr), 0, 0)

	if err := future.Error(); err != nil {
		fmt.Println("[Raft] AddVoter failed:", err)
	} else {
		fmt.Println("[Raft] Added node:", raftAddr)
	}
}

func (s *Service) handleLeaveEvent(member serf.Member) {
	raftAddr := member.Tags["raft_addr"]
	fmt.Println("[Serf] Node left:", member.Name, "with Raft address:", raftAddr)
	if s.raftNode.State() != raft.Leader {
		return
	}

	future := s.raftNode.RemoveServer(raft.ServerID(raftAddr), 0, 0)

	if err := future.Error(); err != nil {
		fmt.Println("[Raft] RemoveServer failed:", err)
	} else {
		fmt.Println("[Raft] Removed node:", raftAddr)
	}
}
