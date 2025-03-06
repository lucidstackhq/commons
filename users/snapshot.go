package users

import (
	"github.com/hashicorp/raft"
)

type storeSnapshot struct {
	data []byte
}

func (s *storeSnapshot) Persist(sink raft.SnapshotSink) error {
	if _, err := sink.Write(s.data); err != nil {
		err := sink.Cancel()
		if err != nil {
			return err
		}
		return err
	}
	return sink.Close()
}

func (s *storeSnapshot) Release() {}
