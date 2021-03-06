package fsm

import (
	"encoding/json"

	"github.com/hashicorp/raft"
)

type fsmSnapshot struct {
	Value map[string][]byte `json:"value"`
}

func (f fsmSnapshot) Persist(sink raft.SnapshotSink) error {
	err := func() error {
		snapshot, err := json.Marshal(f)
		if err != nil {
			return err
		}

		if _, err := sink.Write(snapshot); err != nil {
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

func (f fsmSnapshot) Release() {
}
