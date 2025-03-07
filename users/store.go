package users

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"io"
	"os"
	"path"
	"strings"
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
	mu       sync.Mutex
	data     map[string]string
	raft     *raft.Raft
	dataPath string
}

func NewStore(dataPath string) *Store {
	return &Store{
		data:     make(map[string]string),
		dataPath: dataPath,
	}
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
	default:
		return fmt.Errorf("unknown command")
	}
}

func (s *Store) Set(username, password string) error {
	cmd := Command{Op: "set", Username: username, Password: password}
	return s.execute(cmd)
}

func (s *Store) Get(username string) (string, error) {
	user, ok := s.data[username]
	if !ok {
		return "", fmt.Errorf("user not found")
	}
	return user, nil
}

func (s *Store) GetInfo() gin.H {
	return gin.H{
		"leader": s.raft.State() == raft.Leader,
	}
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

func (s *Store) setupRaft(raftAddr string, bootstrapServers string) error {
	config := raft.DefaultConfig()
	config.LocalID = raft.ServerID(raftAddr)

	snapshotStore, err := raft.NewFileSnapshotStore(path.Join(s.dataPath, "raft"), 2, os.Stdout)
	if err != nil {
		return err
	}

	logStore, err := raftboltdb.NewBoltStore(path.Join(s.dataPath, "raft", "logs.bolt"))
	if err != nil {
		return err
	}

	var bootstrapServersList []raft.Server
	if len(bootstrapServers) != 0 {
		bootstrapServerStringList := strings.Split(bootstrapServers, ",")
		for _, bootstrapServerString := range bootstrapServerStringList {
			bootstrapServerString = strings.TrimSpace(bootstrapServerString)
			bootstrapServersList = append(bootstrapServersList, raft.Server{
				ID:      raft.ServerID(bootstrapServerString),
				Address: raft.ServerAddress(bootstrapServerString),
			})
		}
	}

	bootstrapServersList = append(bootstrapServersList, raft.Server{
		ID:      config.LocalID,
		Address: raft.ServerAddress(raftAddr),
	})

	transport, err := raft.NewTCPTransport(raftAddr, nil, 3, raftTimeout, os.Stdout)
	if err != nil {
		return err
	}

	ra, err := raft.NewRaft(config, s, logStore, logStore, snapshotStore, transport)
	if err != nil {
		return err
	}

	ra.BootstrapCluster(raft.Configuration{
		Servers: bootstrapServersList,
	})

	s.raft = ra
	return nil
}
