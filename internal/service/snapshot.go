package service

import (
	"encoding/binary"
	"fmt"

	"github.com/hashicorp/raft"
)

type snapshot struct {
	data []byte
}

func (s *snapshot) Persist(sink raft.SnapshotSink) error {

	fmt.Println("Persisting snapshot data -> ", string(s.data))

	err := func() error {
		sizeBuf := make([]byte, 8)
		binary.BigEndian.PutUint64(sizeBuf, uint64(len(s.data)))
		if _, err := sink.Write(sizeBuf); err != nil {
			return fmt.Errorf("write size prefix: %w", err)
		}

		if _, err := sink.Write(s.data); err != nil {
			return fmt.Errorf("write snapshot data: %w", err)
		}

		return sink.Close()
	}()

	if err != nil {
		_ = sink.Cancel()
		return err
	}

	return nil
}

func (s *snapshot) Release() {}
