package server

import (
	"hashicorp-raft/config"
	"hashicorp-raft/fsm"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/hashicorp/raft"
)

type Server struct {
	raft *raft.Raft
	fsm  *fsm.FSM
}

func New(raft *raft.Raft, fsm *fsm.FSM) *Server {
	return &Server{
		raft: raft,
		fsm:  fsm,
	}
}

func (s *Server) Start(config *config.StartupConfig) error {
	handler := s.routes()

	return http.ListenAndServe(
		config.HTTPAddr.String(),
		handler,
	)
}

func (s *Server) routes() http.Handler {
	r := mux.NewRouter()

	r.HandleFunc("/key", s.GetKey()).Methods("GET")
	r.HandleFunc("/key", s.SetKey()).Methods("POST")
	r.HandleFunc("/join", s.JoinNode()).Methods("GET")

	return r
}
