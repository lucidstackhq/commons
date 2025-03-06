package users

import (
	"encoding/json"
	"fmt"
	"github.com/hashicorp/raft"
	"github.com/hashicorp/raft-boltdb"
	"io"
	"os"
	"sync"
	"time"
)

const raftTimeout = 10 * time.Second

type Command struct {
	Op       string `json:"op"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Store struct {
	mu   sync.Mutex
	data map[string]string
	raft *raft.Raft
}

func (s *Store) Snapshot() (raft.FSMSnapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(s.data)
	if err != nil {
		return nil, err
	}

	return &storeSnapshot{data: data}, nil
}

func (s *Store) Restore(snapshot io.ReadCloser) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data := make(map[string]string)
	if err := json.NewDecoder(snapshot).Decode(&data); err != nil {
		return err
	}

	s.data = data
	return nil
}

func NewStore() *Store {
	return &Store{
		data: make(map[string]string),
	}
}

func (s *Store) Apply(log *raft.Log) interface{} {
	var cmd Command
	if err := json.Unmarshal(log.Data, &cmd); err != nil {
		return fmt.Errorf("failed to unmarshal command: %v", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	switch cmd.Op {
	case "set":
		s.data[cmd.Username] = cmd.Password
		return nil
	case "get":
		password, exists := s.data[cmd.Username]
		if !exists {
			return fmt.Errorf("user not found")
		}
		return password
	default:
		return fmt.Errorf("unknown command")
	}
}

func (s *Store) Set(username, password string) error {
	cmd := Command{Op: "set", Username: username, Password: password}
	return s.execute(cmd)
}

func (s *Store) Get(username string) (string, error) {
	cmd := Command{Op: "get", Username: username}
	return s.query(cmd)
}
func (s *Store) execute(cmd Command) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not the leader")
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	future := s.raft.Apply(data, raftTimeout)
	return future.Error()
}

func (s *Store) query(cmd Command) (string, error) {
	data, err := json.Marshal(cmd)
	if err != nil {
		return "", err
	}

	future := s.raft.Apply(data, raftTimeout)
	if err := future.Error(); err != nil {
		return "", err
	}

	result, ok := future.Response().(string)
	if !ok {
		return "", fmt.Errorf("invalid response type")
	}

	return result, nil
}

func (s *Store) setupRaft(nodeID, raftAddr string) error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(nodeID)

	snapshotStore, err := raft.NewFileSnapshotStore("./raft", 2, os.Stdout)
	if err != nil {
		return err
	}

	logStore, err := raftboltdb.NewBoltStore("./raft/logs.bolt")
	if err != nil {
		return err
	}

	transport, err := raft.NewTCPTransport(raftAddr, nil, 3, raftTimeout, os.Stdout)
	if err != nil {
		return err
	}

	raftStore := raft.NewInmemStore()

	ra, err := raft.NewRaft(config, s, raftStore, logStore, snapshotStore, transport)
	if err != nil {
		return err
	}

	s.raft = ra
	return nil
}
