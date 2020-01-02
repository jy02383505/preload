package utils

import (
	"fmt"
	"time"

	r "gopkg.in/ini.v1"
	// "gopkg.in/mgo.v2"
	// "gopkg.in/redis.v3"
)

// var cacheDB int64
// var URI = ""
// var DatabaseName = ""
var SendFlag = true
var ServerPort = 9999
var MaxConcurrent = 200
var MaxConcurrent_ch = make(chan int, 200)
var TsConcurrent = 200
var TsConcurrent_ch = make(chan int, 200)
var ConnConcurrent = 200
var ConnConcurrent_ch = make(chan int, 200)
var RangeConcurrent = 100
var RangeConcurrent_ch = make(chan int, 100)
var PartialSize = 10 * 1024 * 1024
var Head = "User-Agent:Mozilla/5.0 (Windows NT 6.3; WOW64 PRELOAD)"
var Ip_to = "127.0.0.1:80"
var Read_time_out = 30

// var ParseTS = true

// connect to
var Ssl_ip_to = "127.0.0.1:443"

// build url
var Ssl_ip_host = "127.0.0.1"

// var (
// 	Session     *mgo.Session
// 	RedisClient *redis.Client
// )

func init() {
	cfg, err := r.Load("/usr/local/preload/config.ini") // online env
	// cfg, err := r.Load("/usr/local/preload/src/go/preloadgo/config.ini") // online env old
	// cfg, err := r.Load("config.ini") // for local
	if err != nil {
		panic(err)
	}
	loglevel := cfg.Section("log").Key("level").String()
	logfile := cfg.Section("log").Key("filename").String()
	// logfile = fmt.Sprintf("%s/%s", "logs", logfile)
	SetLog(loglevel, logfile)
	ServerPort, err = cfg.Section("server").Key("port").Int()
	SendFlag, err = cfg.Section("server").Key("send").Bool()
	MaxConcurrent, err = cfg.Section("server").Key("maxConcurrent").Int()
	MaxConcurrent_ch = make(chan int, MaxConcurrent)
	TsConcurrent, err = cfg.Section("server").Key("TsConcurrent").Int()
	TsConcurrent_ch = make(chan int, TsConcurrent)
	ConnConcurrent, err = cfg.Section("server").Key("ConnConcurrent").Int()
	ConnConcurrent_ch = make(chan int, ConnConcurrent)
	RangeConcurrent, err = cfg.Section("server").Key("RangeConcurrent").Int()
	RangeConcurrent_ch = make(chan int, RangeConcurrent)
	PartialSize, err = cfg.Section("server").Key("PartialSize").Int()
	Head = cfg.Section("server").Key("head").String()
	Ip_to = cfg.Section("server").Key("ip_to").String()
	Read_time_out, err = cfg.Section("server").Key("read_time_out").Int()
	Ssl_ip_to = cfg.Section("server").Key("ssl_ip_to").String()
	Ssl_ip_host = cfg.Section("server").Key("ssl_ip_host").String()
	fmt.Printf("(%s)MaxConcurrent: %d|| TsConcurrent: %d|| ConnConcurrent: %d|| ServerPort: %d|| Head: %s|| Ip_to: %s|| Read_time_out: %d|| Ssl_ip_to: %s|| Ssl_ip_host: %s|| RangeConcurrent: %d|| PartialSize: %d|| InitializationWithNothingWrong->server is running...\n\n", time.Now().Format("2006-01-02 15:04:05"), MaxConcurrent, TsConcurrent, ConnConcurrent, ServerPort, Head, Ip_to, Read_time_out, Ssl_ip_to, Ssl_ip_host, RangeConcurrent, PartialSize)

}
