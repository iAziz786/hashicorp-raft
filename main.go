package main

import (
	"fmt"
	"hashicorp-raft/config"
	"hashicorp-raft/fsm"
	"hashicorp-raft/server"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
	"github.com/rs/zerolog"
)

func main() {
	config, err := config.ReadStartupConfig().Sanitize()
	if err != nil {
		panic(fmt.Sprintf("start up config read failed %s", err))
	}

	logger := zerolog.New(os.Stdout)
	if err != nil {
		panic(fmt.Sprintf("logger creation failed %s", err))
	}

	raftNode, fsm, err := NewRaftNode(config, &logger)
	if err != nil {
		log.Fatalln(err)
	}

	if config.Join != "" {
		go joinPeer(config)
	}

	srv := server.New(raftNode, fsm)
	if err := srv.Start(config); err != nil {
		log.Fatalln(err)
	}
}

func NewRaftNode(cfg *config.StartupConfig, log *zerolog.Logger) (*raft.Raft, *fsm.FSM, error) {
	config := raft.DefaultConfig()
	fsm := fsm.New()

	logStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.DataDir, "raft-log.bolt"))
	if err != nil {
		return nil, nil, err
	}

	stableStore, err := raftboltdb.NewBoltStore(filepath.Join(cfg.DataDir, "raft.stable.bolt"))
	if err != nil {
		return nil, nil, err
	}

	snapshotStore, err := raft.NewFileSnapshotStore(cfg.DataDir, 1, log)
	if err != nil {
		panic(fmt.Sprintf("failed to create snap store %s", err))
	}

	transporter, err := createRaftTransporter(cfg.RaftAddr, log)
	if err != nil {
		panic(fmt.Sprintf("failed to create transporter %s", err))
	}

	config.LocalID = raft.ServerID(cfg.RaftAddr.String())
	raftNode, err := raft.NewRaft(config, fsm, logStore, stableStore, snapshotStore, transporter)
	if err != nil {
		return nil, nil, err
	}

	if cfg.Bootstrap {
		bootstrapCfg := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      config.LocalID,
					Address: transporter.LocalAddr(),
				},
			},
		}

		raftNode.BootstrapCluster(bootstrapCfg)
	}

	return raftNode, fsm, nil
}

func createRaftTransporter(raftAddr net.Addr, log io.Writer) (*raft.NetworkTransport, error) {
	addr, err := net.ResolveTCPAddr("tcp", raftAddr.String())
	if err != nil {
		panic(fmt.Sprintf("failed to created tcp address %s", err))
	}

	transport, err := raft.NewTCPTransport(addr.String(), addr, 3, 5*time.Second, log)
	if err != nil {
		return nil, err
	}
	return transport, nil
}

func joinPeer(cfg *config.StartupConfig) {
	tryJoining := func() error {
		url := url.URL{
			Scheme: "http",
			Host:   cfg.Join,
			Path:   "join",
		}

		req, err := http.NewRequest(http.MethodGet, url.String(), nil)
		if err != nil {
			return err
		}

		req.Header.Add("Peer-Addr", cfg.RaftAddr.String())

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			return err
		}

		if res.StatusCode != http.StatusOK {
			return fmt.Errorf("non ok status %d", res.StatusCode)
		}

		return nil
	}

	for {
		if err := tryJoining(); err != nil {
			fmt.Println(err)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
}
