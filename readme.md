run node bootstrap:
- go run cmd/main.go server 8001 9001 nbootstrap 7001 7002
    - server: command that run server
    - 8001: grpc port that server serve request read/write log
    - 9001: raft address to replicate data to peers
    - bootstrap/nbootstrap: if bootstrap -> this is raft bootstrap node. otherwise, cluster already inited, node will request join cluster by discovering peers (with serf)
    - 7001: serf address
    - 7002: serf seed port, this is port of node already running, current node will connect to seed port to join to serf cluster