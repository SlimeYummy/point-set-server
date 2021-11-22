package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"point-set/base"
	"point-set/core"
	"time"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

func startHttp(mgr *core.RoomManager) {
	h := handler{mgr}
	r := mux.NewRouter()
	r.HandleFunc("/create-room", h.createRoom).Methods("POST")
	r.HandleFunc("/delete-room", h.deleteRoom).Methods("POST")
	http.Handle("/", r)

	addr := "127.0.0.1:8080"
	base.LogPrint(base.LevelInfo, nil, fmt.Sprintf("start HTTP %s", addr))
	http.ListenAndServe(addr, nil)
}

type handler struct {
	mgr *core.RoomManager
}

const success = "{\"success\":true}"
const failure = "{\"success\":false}"

type createArgs struct {
	RoomId   string             `json:"room_id"`
	Duration time.Duration      `json:"duration"`
	Configs  []core.PlayerBasic `json:"configs"`
}

type createRet struct {
	Success  bool                 `json:"success"`
	RoomId   string               `json:"room_id"`
	Duration time.Duration        `json:"duration"`
	Configs  []*core.PlayerConfig `json:"configs"`
}

func (h handler) createRoom(w http.ResponseWriter, r *http.Request) {
	var args createArgs
	err := json.NewDecoder(r.Body).Decode(&args)
	if err != nil {
		http.Error(w, failure, http.StatusBadRequest)
		base.LogPrint(base.LevelError, nil, errors.WithStack(err))
		return
	}

	cfgs, err := h.mgr.CreateRoom(args.RoomId, args.Duration, args.Configs)
	if err != nil {
		http.Error(w, failure, http.StatusInternalServerError)
		base.LogPrint(base.LevelError, nil, err)
		return
	}

	ret := createRet{
		Success:  true,
		RoomId:   args.RoomId,
		Duration: args.Duration,
		Configs:  cfgs,
	}
	err = json.NewEncoder(w).Encode(ret)
	if err != nil {
		http.Error(w, failure, http.StatusInternalServerError)
		base.LogPrint(base.LevelError, nil, errors.WithStack(err))
	}
}

type deleteArgs struct {
	RoomId string `json:"room_id"`
}

func (h handler) deleteRoom(w http.ResponseWriter, r *http.Request) {
	var args deleteArgs
	err := json.NewDecoder(r.Body).Decode(&args)
	if err != nil {
		http.Error(w, failure, http.StatusBadRequest)
		base.LogPrint(base.LevelError, nil, errors.WithStack(err))
		return
	}

	err = h.mgr.DeleteRoom(args.RoomId)
	if err != nil {
		http.Error(w, failure, http.StatusInternalServerError)
		base.LogPrint(base.LevelError, nil, err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(success))
}
