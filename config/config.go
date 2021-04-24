package config

import (
	"errors"
	"flag"
	"net"

	template "github.com/hashicorp/go-sockaddr/template"
)

type RawConfig struct {
	Port      int
	RaftPort  int
	Bootstrap bool
	Join      string
	DataDir   string
	BindAddr  string
}

type StartupConfig struct {
	HTTPAddr  net.Addr
	RaftAddr  net.Addr
	Bootstrap bool
	Join      string
	DataDir   string
}

func ReadStartupConfig() *RawConfig {
	config := RawConfig{}
	flag.IntVar(&config.Port, "port", 3000, "start server on the port specified")
	flag.IntVar(&config.RaftPort, "raft-port", 4000, "start receiving the connection from peers")
	flag.BoolVar(&config.Bootstrap, "bootstrap", false, "bootstrap server means it is the first node in the cluster")
	flag.StringVar(&config.Join, "join", "", "join the peers on this port")
	flag.StringVar(&config.DataDir, "data-dir", "raftDB", "directory where the data is stored")
	flag.StringVar(&config.BindAddr, "bind-addr", "127.0.0.1", "directory where the data is stored")

	flag.Parse()

	return &config
}

func (s *RawConfig) Sanitize() (*StartupConfig, error) {
	if s.Port < 1 || s.Port > 65536 {
		return nil, errors.New("port should be range 1 - 65536")
	}

	if s.RaftPort < 1 || s.RaftPort > 65536 {
		return nil, errors.New("raft port should be range 1 - 65536")
	}

	if s.RaftPort == s.Port {
		return nil, errors.New("raft port and join port can not be the same")
	}

	var bindAddr net.IP
	resolvedBindAddr, err := template.Parse(s.BindAddr)
	if err != nil {
		return nil, err
	}

	bindAddr = net.ParseIP(resolvedBindAddr)

	httpAddr := &net.TCPAddr{
		IP:   bindAddr,
		Port: s.Port,
	}

	raftAddr := &net.TCPAddr{
		IP:   bindAddr,
		Port: s.RaftPort,
	}

	return &StartupConfig{
		DataDir:   s.DataDir,
		Bootstrap: s.Bootstrap,
		Join:      s.Join,
		HTTPAddr:  httpAddr,
		RaftAddr:  raftAddr,
	}, nil
}
