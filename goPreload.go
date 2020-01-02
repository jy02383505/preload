// Copyright 201７
//Created by longjun.zhao on 5/19/2016
// 预加载

package main

import (
	rec "PreloadGo/process"
	ut "PreloadGo/utils"
	"fmt"
	"net/http"
	_ "net/http/pprof"

	// "gopkg.in/ini.v1"
	// log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

var serverPort = ut.ServerPort
var log = ut.Logger

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe(":7070", nil))
	// }()
	router := mux.NewRouter()
	router.HandleFunc("/", rec.TaskRequestPost).Methods("POST")
	log.Printf("Starting on port: %d", serverPort)
	// log.Printf("maxConcurrent_ch:%s", maxConcurrent_ch)
	if errServer := http.ListenAndServe(fmt.Sprintf(":%d", serverPort), router); errServer != nil {
		log.Printf("FatalError: main errServer: %s", errServer)
	}

}
