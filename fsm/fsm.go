package fsm

import (
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/hashicorp/raft"
)

type FSM struct {
	mutex sync.Mutex
	Value map[string][]byte
}

type Action string

type Event struct {
	Type  Action
	Key   string
	Value []byte
}

const (
	SET Action = "SET"
)

func New() *FSM {
	return &FSM{
		mutex: sync.Mutex{},
		Value: map[string][]byte{
			"": {},
		},
	}
}

func (f *FSM) Apply(log *raft.Log) interface{} {
	var e Event
	if err := json.Unmarshal(log.Data, &e); err != nil {
		panic("failed to unmarshal event")
	}

	switch e.Type {
	case SET:
		f.mutex.Lock()
		defer f.mutex.Unlock()
		f.Value[e.Key] = e.Value
	default:
		panic(fmt.Sprintf("unsupported event %s", e.Type))
	}

	return nil
}

func (f *FSM) Snapshot() (raft.FSMSnapshot, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	snapshot := fsmSnapshot{Value: f.Value}
	return snapshot, nil
}

func (f *FSM) Restore(rc io.ReadCloser) error {
	var snapshot fsmSnapshot
	if err := json.NewDecoder(rc).Decode(&snapshot); err != nil {
		return err
	}
	f.Value = snapshot.Value
	return nil
}
