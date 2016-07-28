package main

import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "net"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/hashicorp/raft"
    "github.com/hashicorp/raft-boltdb"
)

const (
    raftTimeout = 10 * time.Second
)

var store *Store

func init() {
    store = new(Store)
    store.kv = make(map[string]string)
    store.logger = log.New(os.Stderr, "", log.LstdFlags)
}

type command struct {
    Op    string `json:"op,omitempty"`
    Key   string `json:"key,omitempty"`
    Value string `json:"value,omitempty"`
}

type Store struct {
    mutex    sync.Mutex
    kv       map[string]string
    raft     *raft.Raft
    logger   *log.Logger
}

func (s *Store) Open(raftDir, raftAddr string, singleton bool) error {
    config := raft.DefaultConfig()

    var peers []string
    b, err := ioutil.ReadFile(filepath.Join(raftDir, "peers.json"))
    if err == nil {
        if err := json.NewDecoder(bytes.NewReader(b)).Decode(&peers); err != nil {
            return err
        }
    } else {
        if !os.IsNotExist(err) {
            return err
        }
    }

    if singleton && len(peers) < 2 {
        s.logger.Println("enabling single-node mode")
        config.EnableSingleNode = true
        config.DisableBootstrapAfterElect = false
    }

    addr, err := net.ResolveTCPAddr("tcp", raftAddr)
    if err != nil {
        return err
    }

    transport, err := raft.NewTCPTransport(raftAddr, addr, 3, 10*time.Second, os.Stderr)
    if err != nil {
        return err
    }

    snapshots, err := raft.NewFileSnapshotStore(raftDir, 2, os.Stderr)
    if err != nil {
        return err
    }

    logStore, err := raftboltdb.NewBoltStore(filepath.Join(raftDir, "raft.db"))
    if err != nil {
        return err
    }

    peerStore := raft.NewJSONPeers(raftDir, transport)
    ra, err := raft.NewRaft(config, (*fsm)(s), logStore, logStore, snapshots, peerStore, transport)
    if err != nil {
        return err
    }

    s.raft = ra
    return nil
}

func (s *Store) Get(key string) (string, error) {
    s.mutex.Lock()
    defer s.mutex.Unlock()
    return s.kv[key], nil
}

func (s *Store) Set(key, value string) error {
    if s.raft.State() != raft.Leader {
        return fmt.Errorf("not leader")
    }

    c := &command{
        Op:    "set",
        Key:   key,
        Value: value,
    }

    b, err := json.Marshal(c)
    if err != nil {
        return err
    }

    f := s.raft.Apply(b, raftTimeout)
    if err, ok := f.(error); ok {
        return err
    }

    return nil
}

func (s *Store) Delete(key string) error {
    if s.raft.State() != raft.Leader {
        return fmt.Errorf("not leader")
    }

    c := &command{
        Op:  "delete",
        Key: key,
    }

    b, err := json.Marshal(c)
    if err != nil {
        return err
    }

    f := s.raft.Apply(b, raftTimeout)
    if err, ok := f.(error); ok {
        return err
    }

    return nil
}

func (s *Store) Join(addr string) error {
    s.logger.Printf("joining remote node %s", addr)

    f := s.raft.AddPeer(addr)
    if f.Error() != nil {
        return f.Error()
    }

    s.logger.Printf("successfully join remote node %s", addr)
    return nil
}

////////////////////////////////////////////////////////////////////////////////

type fsm Store

func (f *fsm) Apply(l *raft.Log) interface{} {
    var c command
    if err := json.Unmarshal(l.Data, &c); err != nil {
        panic(err)
    }

    switch c.Op {
    case "set":
        return f.applySet(c.Key, c.Value)
    case "delete":
        return f.applyDelete(c.Key)
    default:
        panic(fmt.Sprintf("unknown command op: %s", c.Op))
    }
}

func (f *fsm) Snapshot() (raft.FSMSnapshot, error) {
    f.mutex.Lock()
    defer f.mutex.Unlock()

    o := make(map[string]string)
    for k, v := range f.kv {
        o[k] = v
    }

    return &fsmSnapshot{store: o}, nil
}

func (f *fsm) Restore(rc io.ReadCloser) error {
    o := make(map[string]string)
    if err := json.NewDecoder(rc).Decode(&o); err != nil {
        return err
    }

    f.kv = o
    return nil
}

func (f *fsm) applySet(key, value string) interface{} {
    f.mutex.Lock()
    defer f.mutex.Unlock()

    f.kv[key] = value
    return nil
}

func (f *fsm) applyDelete(key string) interface{} {
    f.mutex.Lock()
    defer f.mutex.Unlock()

    delete(f.kv, key)
    return nil
}

////////////////////////////////////////////////////////////////////////////////

type fsmSnapshot struct {
    store map[string]string
}

func (f *fsmSnapshot) Persist(sink raft.SnapshotSink) error {
    err := func() error {
        b, err := json.Marshal(f.store)
        if err != nil {
            return err
        }

        if _, err := sink.Write(b); err != nil {
            return err
        }

        if err := sink.Close(); err != nil {
            return err
        }

        return nil
    }()

    if err != nil {
        sink.Cancel()
        return err
    }

    return nil
}

func (f *fsmSnapshot) Release() {}

////////////////////////////////////////////////////////////////////////////////

func readPeersJSON(path string) ([]string, error) {
    b, err := ioutil.ReadFile(path)
    if err != nil && !os.IsNotExist(err) {
        return nil, err
    }

    if len(b) == 0 {
        return nil, nil
    }

    var peers []string
    dec := json.NewDecoder(bytes.NewReader(b))
    if err := dec.Decode(&peers); err != nil {
        return nil, err
    }

    return peers, nil
}
