package server

import (
	"encoding/json"
	"fmt"
	"hashicorp-raft/fsm"
	"log"
	"net/http"
	"time"

	"github.com/hashicorp/raft"
)

type setRes struct {
	Success bool   `json:"success"`
	Message string `json:"msg,omitempty"`
	Value   []byte `json:"value,omitempty"`
}

func (s *Server) SetKey() http.HandlerFunc {
	type setReq struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	return func(rw http.ResponseWriter, r *http.Request) {
		req := setReq{}
		rw.Header().Add("Content-Type", "application/json")
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			json.NewEncoder(rw).Encode(setRes{
				Success: false,
				Message: "unsupported json format",
			})
			return
		}

		e := fsm.Event{
			Type:  fsm.SET,
			Key:   req.Key,
			Value: []byte(req.Value),
		}

		data, err := json.Marshal(e)
		if err != nil {
			log.Println("failed to marshal data", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		if applyFuture := s.raft.Apply(data, 5*time.Second); applyFuture.Error() != nil {
			json.NewEncoder(rw).Encode(setRes{
				Success: false,
				Message: applyFuture.Error().Error(),
			})
			return
		}

		json.NewEncoder(rw).Encode(setRes{
			Success: true,
			Message: "ok",
		})
	}
}

func (s *Server) GetKey() http.HandlerFunc {
	type GetReq struct {
		Key string `json:"key"`
	}
	return func(rw http.ResponseWriter, r *http.Request) {
		req := GetReq{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		res := setRes{
			Success: true,
			Value:   s.fsm.Value[req.Key],
		}

		data, err := json.Marshal(&res)
		if err != nil {
			log.Println("failed to marshal data", err)
			rw.WriteHeader(http.StatusInternalServerError)
		}

		rw.WriteHeader(http.StatusOK)
		rw.Write(data)
	}
}

func (s *Server) JoinNode() http.HandlerFunc {
	return func(rw http.ResponseWriter, r *http.Request) {
		peerAddr := r.Header.Get("Peer-Addr")
		fmt.Println("peerAddr", peerAddr)
		if peerAddr == "" {
			rw.WriteHeader(http.StatusBadRequest)
			return
		}

		peerFuture := s.raft.AddVoter(raft.ServerID(peerAddr), raft.ServerAddress(peerAddr), 0, 0)
		if err := peerFuture.Error(); err != nil {
			log.Println("failed to join the leader", err)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.WriteHeader(http.StatusOK)
	}
}
