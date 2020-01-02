// version: 3.0
package core

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	// "log"
	ut "PreloadGo/utils"
	// "net/url"
	"bytes"
	"crypto/tls"
	"net"
	"net/http"
	"strings"
	"time"
	// "io/ioutil"
	// "bufio"
	// log "github.com/Sirupsen/logrus"
	// log "github.com/omidnikta/logrus"
	// "gopkg.in/mgo.v2"
	// "gopkg.in/mgo.v2/bson"
	"strconv"
)

// var redisHost string = "172.16.21.198:6379" //223.202.52.82
// var cacheDB int64

//是否发送结果 到边缘
// var sendFlag = ut.SendFlag
var log = ut.Logger

// var uri = ut.URI
var maxConcurrent_num = ut.MaxConcurrent
var maxConcurrent_ch = ut.MaxConcurrent_ch

// var Head_send = ut.Head
var Head_send = "User-Agent:Mozilla/5.0 (Windows NT 6.3; WOW64 PRELOAD)"
var Ip_to = ut.Ip_to

// var Report_address = ut.Report_address
var Read_time_out = ut.Read_time_out
var ssl_ip_to = ut.Ssl_ip_to
var ssl_ip_host = ut.Ssl_ip_host

// var ConnMap = make(map[string]int) // limit channel number

// log.Debugf("maxConcurrent:%s", maxConcurrent)
// type RequestReportBodyT struct {
//     Url     string `json:"url"`
// }

// type ReceiveBody struct {
// 	Channels []UrlId `json:"channels"`
// }

// type ReceiveBodyUrlId struct {
// 	Url string  `json:"url"`
// 	Url_Id string  `json:"url_id"`
// }

type Task struct {
	Url              string
	Conn             int `connect number per channel`
	Id               string
	Rate             int `rate`
	Check_type       string
	Preload_address  string
	Priority         int
	Nest_track_level int
	Limit_rate       int `useless temporarily`
	Md5              string
	Header_list      []map[string]string `json:"header_list"`
}

type DataFromCenter struct {
	Url_list            []Task
	Compressed_url_list []Task
	Is_override         int
	Check_value         string
	Sessionid           string
	Lvs_address         string
	DataType            string
	Action              string
	Report_address      string
}

type ReceiveBody struct {
	Check_result       string              `json:"check_result"`
	Download_mean_rate string              `json:"download_mean_rate"`
	Lvs_address        string              `json:"lvs_address"`
	Preload_status     string              `json:"preload_status"`
	Sessionid          string              `json:"sessionid"`
	Http_status        int                 `json:"http_status"`
	Response_time      string              `json:"response_time"`
	Refresh_status     string              `json:"refresh_status"`
	Check_type         string              `json:"check_type"`
	Data               string              `json:"data"`
	Last_modified      string              `json:"last_modified"`
	Url                string              `json:"url"`
	Cache_status       string              `json:"cache_status"`
	Url_id             string              `json:"url_id"`
	Content_length     int                 `json:"content_length"`
	Report_ip          string              `json:"report_ip"`
	Report_port        string              `json:"report_port"`
	Status             string              `json:"status"`
	Is_compressed      bool                `json:"is_compressed"`
	Rate               int                 `json:"rate"`
	Conn               int                 `json:"Conn"`
	Header_list        []map[string]string `json:"header_list"`
	// Count               int                 `count the number of rate-limit urls received`
}

type ConnInStruct struct {
	ConnChan    chan int `waitChan`
	CountConnIn chan int `safely plus every number to NumIn`
	NumIn       int      `the total number of conn urls receiverd`
}

var ConnInMap = make(map[string]ConnInStruct)
var ConnInChan = make(chan ConnInStruct, maxConcurrent_num)

type ConnOutStruct struct {
	CountConnOut chan int `safely plus every number to NumOut`
	NumOut       int      `the total number of conn urls finished`
}

var ConnOutMap = make(map[string]ConnOutStruct)
var ConnOutChan = make(chan ConnOutStruct, maxConcurrent_num)

// [ ack format as follow:
// {'sessionid': '089790c08e8911e1910800247e10b29b',
// 'pre_ret_list': [{'id': 'sfsdf', 'code': 200}]}
// ]

func NewReceiveBody(data DataFromCenter) ([]ReceiveBody, map[string]interface{}) {
	var pre_ret = make(map[string]interface{})
	var pre_ret_list []map[string]interface{}
	var ack = make(map[string]interface{})
	var result []ReceiveBody
	var is_compressed bool
	var body ReceiveBody
	var url_list []Task
	if len(data.Compressed_url_list) > 0 {
		is_compressed = true
		url_list = data.Compressed_url_list
	}
	if len(data.Url_list) > 0 {
		is_compressed = false
		url_list = data.Url_list
	}
	log.Debugf("NewReceiveBody url_list: %T|| url_list: %+v", url_list, url_list)
	for i := 0; i < len(url_list); i++ {
		body.Last_modified = "-"
		body.Response_time = "-"
		body.Download_mean_rate = "0"
		body.Http_status = 0
		body.Preload_status = "200"
		body.Refresh_status = "failed"
		body.Data = "-"
		body.Cache_status = "HIT"
		body.Content_length = 0
		body.Is_compressed = is_compressed

		body.Lvs_address = data.Lvs_address
		body.Sessionid = data.Sessionid
		body.Report_ip, body.Report_port = splitReportAddress(data.Report_address)
		body.Url_id = url_list[i].Id
		body.Url = url_list[i].Url
		body.Check_type = url_list[i].Check_type
		body.Check_result = url_list[i].Md5
		body.Rate = url_list[i].Rate
		body.Conn = url_list[i].Conn
		body.Header_list = url_list[i].Header_list
		result = append(result, body)

		pre_ret["id"] = url_list[i].Id
		pre_ret["code"] = 200
		pre_ret_list = append(pre_ret_list, pre_ret)
	}
	ack["sessionid"] = data.Sessionid
	ack["pre_ret_list"] = pre_ret_list
	return result, ack
}

func splitReportAddress(report_address string) (string, string) {
	report_arr := strings.SplitN(report_address, ":", 2)
	return report_arr[0], report_arr[1]
}

func GetHost(str string) (host string, len_str int) {
	array_str := strings.SplitN(str, "/", 4)
	if len(array_str) < 3 {
		log.Debugf("parse str error:", str)
		host = "127.0.0.1"

	} else {
		host = array_str[2]
	}

	len_str = len(array_str)
	return host, len_str
}

func is_http(str string) (is_http_t bool) {
	array_str := strings.SplitN(str, "/", 4)
	is_http_t = true
	if len(array_str) > 0 {
		// log.Debugf("is_http url: %s|| array_str[0]: %s", str, array_str[0])
		if array_str[0] == "https:" {
			is_http_t = false
		}

	}
	return is_http_t
}

func getChannelName(url string) string {
	url_arr := strings.Split(url, "/")
	channelName := url_arr[0] + "//" + url_arr[2]
	return channelName
}

func get_head_length(str string) (string, int, int, int) {
	var theContentLength int
	var err error
	req_arr := strings.SplitN(str, "\r\n\r\n", 2)
	thehead := req_arr[0]
	first_line := strings.SplitN(thehead, "\r\n", 2)[0]
	every_line_arr := strings.Split(thehead, "\r\n")
	for _, s_line := range every_line_arr {
		if strings.Contains(s_line, "Content-Length") {
			theContentLength, err = strconv.Atoi(strings.Replace(strings.SplitN(s_line, ":", 2)[1], " ", "", -1))
			if err != nil {
				log.Debugf("get_head_length theContentLength convert to int err: %s", err)
			}
		}
	}
	log.Debugf("get_head_length theContentLength: %d", theContentLength)
	http_status, err := strconv.Atoi(strings.SplitN(first_line, " ", 3)[1])
	if err != nil {
		log.Debugf("get_head_length http_status convert to int error.http_status: %s", err)
	}
	head_length := len(thehead)
	return thehead, head_length, http_status, theContentLength
}

func check_conn_out(body ReceiveBody) {
	var CountConnOut = make(chan int, body.Conn)
	connKey := getChannelName(body.Url) + "__" + fmt.Sprintf("%d", body.Conn)
	log.Debugf("check_conn_out connKey: %s", connKey)
	<-ConnInMap[connKey].ConnChan
	connOut, is_existed := ConnOutMap[connKey]
	if !is_existed {
		connOut.CountConnOut = CountConnOut
	}
	connOut.CountConnOut <- 1
	connOut.NumOut += <-connOut.CountConnOut
	ConnOutMap[connKey] = connOut
	log.Debugf("check_conn_out ConnOutMap: %+v", ConnOutMap)
	if ConnInMap[connKey].NumIn == ConnOutMap[connKey].NumOut {
		log.Debugf("check_conn_out [ConnInMap[connKey].NumIn == ConnOutMap[connKey].NumOut]|| ConnInMap: %+v|| ConnOutMap: %+v", ConnInMap, ConnOutMap)
		delete(ConnInMap, connKey)
		delete(ConnOutMap, connKey)
		log.Debugf("check_conn_out [NumIn==NumOut,delete the Two keys successfully.]|| connKey: %s|| ConnInMap: %+v|| ConnOutMap: %+v|| Url_id: %s|| Url: %s", connKey, ConnInMap, ConnOutMap, body.Url_id, body.Url)
	}
	log.Debugf("check_conn_out ConnInMap: %+v|| ConnOutMap: %+v|| Url_id: %s|| Url: %s", ConnInMap, ConnOutMap, body.Url_id, body.Url)
}

func GetProxy_http(ip_to string, head string, receiveBody_t ReceiveBody) {
	// channel_name := getChannelName(receiveBody_t.Url)
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ip_to)
	if err != nil {
		log.Debugf("GetProxy_http err: %s|| url_id: %s|| url: %s", err, receiveBody_t.Url_id, receiveBody_t.Url)
	}

	tcpconn, err2 := net.DialTCP("tcp", nil, tcpAddr)
	if err2 != nil {
		log.Debugf("GetProxy_http err2: %s|| url_id: %s|| url: %s", err2, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err2)
		go PostRealBody(receiveBody_t)
		if receiveBody_t.Conn != 0 {
			check_conn_out(receiveBody_t)
		}
		<-maxConcurrent_ch
		return
	}
	host, _ := GetHost(receiveBody_t.Url)
	var headers_str string
	for _, d := range receiveBody_t.Header_list {
		for k, v := range d {
			headers_str += k + ":" + v + "\r\n"
		}
	}
	log.Debugf("GetProxy_http headers_str: %s|| url_id: %s|| url: %s", headers_str, receiveBody_t.Url_id, receiveBody_t.Url)

	var str_t string
	var conn_header string
	if receiveBody_t.Conn != 0 {
		conn_header = fmt.Sprintf("\r\nCC_PRELOAD_SPEED:%dKB", receiveBody_t.Rate)
	}
	if receiveBody_t.Is_compressed {
		str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + "Accept-Encoding:gzip\r\n" + head + conn_header + "\r\n\r\n"
	} else {
		str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + head + conn_header + "\r\n\r\n"
	}

	log.Debugf("GetProxy_http str_t: %s|| url_id: %s|| url: %s", str_t, receiveBody_t.Url_id, receiveBody_t.Url)
	//向tcpconn中写入数据
	_, err3 := tcpconn.Write([]byte(str_t))
	if err3 != nil {
		log.Debugf("GetProxy_http err3: %s|| url_id: %s|| url: %s", err3, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err3)
		go PostRealBody(receiveBody_t)
		if receiveBody_t.Conn != 0 {
			check_conn_out(receiveBody_t)
		}
		<-maxConcurrent_ch
		return

	}
	buf := make([]byte, 8192)
	log.Debugf("GetProxy_http Read_time_out: %d", Read_time_out)
	tcpconn.SetDeadline(time.Now().Add(time.Duration(Read_time_out) * time.Second))

	total_length := 0
	head_length := 0
	head_end := false
	var o_content_length int
	st := float64(time.Now().UnixNano() / 1e6)
	for {
		length, errRead := tcpconn.Read(buf)
		if errRead != nil {
			tcpconn.Close()
			log.Debugf("GetProxy_http errRead: %s|| url_id: %s|| url: %s", errRead, receiveBody_t.Url_id, receiveBody_t.Url)
			// get Content_length and Download_mean_rate
			receiveBody_t.Content_length = total_length - head_length - 4
			delta_t := (float64(time.Now().UnixNano()/1e6) - st) / 1000
			var d_rate float64
			if receiveBody_t.Content_length < 0 {
				d_rate = 0
			} else {
				if delta_t <= 0 {
					d_rate = float64(receiveBody_t.Content_length) / 1024
				} else {
					d_rate = float64(receiveBody_t.Content_length) / 1024 / float64(delta_t)
				}
			}
			receiveBody_t.Download_mean_rate = strconv.FormatFloat(d_rate, 'f', 6, 64)
			log.Debugf("GetProxy_http total_length: %d|| delta_t: %f|| url_id: %s|| url: %s", total_length, delta_t, receiveBody_t.Url_id, receiveBody_t.Url)
			if errRead == io.EOF {
				receiveBody_t.Status = "Done"
			} else if strings.Contains(fmt.Sprintf("%s", errRead), "timeout") {
				receiveBody_t.Status = fmt.Sprintf("Timeout(%ds)", Read_time_out)
				if receiveBody_t.Content_length <= 0 {
					receiveBody_t.Content_length = 0
				}
			} else {
				receiveBody_t.Status = fmt.Sprintf("%s", errRead)
			}
			go PostRealBody(receiveBody_t)
			if receiveBody_t.Conn != 0 {
				check_conn_out(receiveBody_t)
			}
			<-maxConcurrent_ch
			break
		}

		if length > 0 {
			total_length += length
			tcpconn.SetDeadline(time.Now().Add(time.Duration(Read_time_out) * time.Second))
		}
		if !head_end {
			recvStr := string(buf[0:length])
			is_head_end := strings.Contains(recvStr, "\r\n\r\n")
			if !is_head_end {
				head_length += length
			} else {
				thehead, h_length, http_status, original_content_length := get_head_length(recvStr)
				o_content_length = original_content_length
				head_length += h_length
				head_end = true
				receiveBody_t.Http_status = http_status
				log.Debugf("GetProxy_http o_content_length: %d|| head_length: %d|| http_status: %d|| thehead: %s|| url_id: %s|| url: %s", o_content_length, head_length, http_status, thehead, receiveBody_t.Url_id, receiveBody_t.Url)
			}
		}
		// the length of receiving satisfied with Content_Length, close connection initiatively.
		if (o_content_length > 0) && ((total_length - head_length - 4) >= o_content_length) {
			tcpconn.Close()
			log.Debugf("GetProxy_http [the length of receiving satisfied with Content_Length, close connection initiatively.] total_length: %d|| head_length: %d|| o_content_length: %d|| url_id: %s|| url: %s", total_length, head_length, o_content_length, receiveBody_t.Url_id, receiveBody_t.Url)
			// get Content_length and Download_mean_rate
			receiveBody_t.Content_length = total_length - head_length - 4
			delta_t := (float64(time.Now().UnixNano()/1e6) - st) / 1000
			var d_rate float64
			if receiveBody_t.Content_length < 0 {
				d_rate = 0
			} else {
				if delta_t <= 0 {
					d_rate = float64(receiveBody_t.Content_length) / 1024
				} else {
					d_rate = float64(receiveBody_t.Content_length) / 1024 / float64(delta_t)
				}
			}
			receiveBody_t.Download_mean_rate = strconv.FormatFloat(d_rate, 'f', 6, 64)
			log.Debugf("GetProxy_http total_length: %d|| delta_t: %f|| url_id: %s|| url: %s", total_length, delta_t, receiveBody_t.Url_id, receiveBody_t.Url)
			receiveBody_t.Status = "Done."
			if receiveBody_t.Conn != 0 {
				check_conn_out(receiveBody_t)
			}
			go PostRealBody(receiveBody_t)
			<-maxConcurrent_ch
			break
		}
		//fmt.Println("Rec[",conn.RemoteAddr().String(),"] Say :" ,string(buf[0:lenght]))
		// reciveStr :=string(buf[0:lenght])
		// if reciveStr != nil {}
	}
	// receiveBody_t.Status = "success"
	// go PostRealBody(receiveBody_t)
	// <-maxConcurrent_ch
}

func GetHost_url(str string) (host string) {
	array_str := strings.SplitN(str, "/", 4)
	if len(array_str) < 3 {
		log.Debugf("GetHost_url len(array_str) < 3 str:", str)
		host = "127.0.0.1"

	} else {
		host = array_str[2]
	}

	// len_str = len(array_str)
	// if len_str <= 3 {
	// 	url_t = str
	// } else {
	// 	url_t = array_str[0] + "//" + ssl_ip_host + "/" + array_str[3]
	// }
	return host
}

// https
func GetProxy_https(ip_to string, head string, receiveBody_t ReceiveBody) {
	// channel_name := getChannelName(receiveBody_t.Url)
	host := GetHost_url(receiveBody_t.Url)
	var headers_str string
	for _, d := range receiveBody_t.Header_list {
		for k, v := range d {
			headers_str += k + ":" + v + "\r\n"
		}
	}
	log.Debugf("GetProxy_https headers_str: %s|| url_id: %s|| url: %s", headers_str, receiveBody_t.Url_id, receiveBody_t.Url)

	var str_t string
	var conn_header string
	if receiveBody_t.Conn != 0 {
		conn_header = fmt.Sprintf("\r\nCC_PRELOAD_SPEED:%dKB", receiveBody_t.Rate)
	}
	if receiveBody_t.Is_compressed {
		str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + "Accept-Encoding:gzip\r\n" + head + conn_header + "\r\n\r\n"
	} else {
		str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + head + conn_header + "\r\n\r\n"
	}
	log.Debugf("GetProxy_https str_t: %s|| url_id: %s|| url: %s", str_t, receiveBody_t.Url_id, receiveBody_t.Url)
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	tcpconn, err2 := tls.Dial("tcp", ssl_ip_to, conf)
	if err2 != nil {
		log.Debugf("GetProxy_https err2: %s|| url_id: %s|| url: %s", err2, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err2)
		go PostRealBody(receiveBody_t)
		if receiveBody_t.Conn != 0 {
			check_conn_out(receiveBody_t)
		}
		<-maxConcurrent_ch
		return
	}

	//     log.Debugf("str_t:%s", str_t)
	//向tcpconn中写入数据
	_, err3 := tcpconn.Write([]byte(str_t))
	if err3 != nil {
		log.Debugf("GetProxy_https err3: %s|| url_id: %s|| url: %s", err3, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err3)
		go PostRealBody(receiveBody_t)
		if receiveBody_t.Conn != 0 {
			check_conn_out(receiveBody_t)
		}
		<-maxConcurrent_ch
		return
	}
	buf := make([]byte, 8192)
	log.Debugf("GetProxy_https Read_time_out: %s|| url_id: %s|| url: %s", Read_time_out, receiveBody_t.Url_id, receiveBody_t.Url)
	tcpconn.SetDeadline(time.Now().Add(time.Duration(Read_time_out) * time.Second))

	total_length := 0
	head_end := false
	head_length := 0
	var o_content_length int
	st := float64(time.Now().UnixNano() / 1e6)
	for {
		length, err := tcpconn.Read(buf)
		if err != nil {
			tcpconn.Close()
			log.Debugf("GetProxy_https err: %s|| url_id: %s|| url: %s", err, receiveBody_t.Url_id, receiveBody_t.Url)
			receiveBody_t.Content_length = total_length - head_length - 4
			delta_t := (float64(time.Now().UnixNano()/1e6) - st) / 1000
			var d_rate float64
			if receiveBody_t.Content_length < 0 {
				d_rate = 0
			} else {
				if delta_t <= 0 {
					d_rate = float64(receiveBody_t.Content_length) / 1024
				} else {
					d_rate = float64(receiveBody_t.Content_length) / 1024 / float64(delta_t)
				}
			}
			receiveBody_t.Download_mean_rate = strconv.FormatFloat(d_rate, 'f', 6, 64)
			log.Debugf("GetProxy_https total_length: %s|| delta_t: %s|| url_id: %s|| url: %s", total_length, delta_t, receiveBody_t.Url_id, receiveBody_t.Url)
			if err == io.EOF {
				receiveBody_t.Status = "Done"
			} else if strings.Contains(fmt.Sprintf("%s", err), "timeout") {
				receiveBody_t.Status = fmt.Sprintf("Timeout(%ds)", Read_time_out)
				if receiveBody_t.Content_length <= 0 {
					receiveBody_t.Content_length = 0
				}
			} else {
				receiveBody_t.Status = fmt.Sprintf("%s", err)
			}
			go PostRealBody(receiveBody_t)
			if receiveBody_t.Conn != 0 {
				check_conn_out(receiveBody_t)
			}
			<-maxConcurrent_ch
			break
		}
		if length > 0 {
			total_length += length
			tcpconn.SetDeadline(time.Now().Add(time.Duration(Read_time_out) * time.Second))
		}
		if !head_end {
			recvStr := string(buf[0:length])
			is_head_end := strings.Contains(recvStr, "\r\n\r\n")
			if !is_head_end {
				head_length += length
			} else {
				thehead, h_length, http_status, original_content_length := get_head_length(recvStr)
				o_content_length = original_content_length
				receiveBody_t.Http_status = http_status
				head_length += h_length
				head_end = true
				log.Debugf("GetProxy_https o_content_length: %d|| head_length: %s|| http_status: %s|| thehead: %s|| url_id: %s|| url: %s", o_content_length, head_length, http_status, thehead, receiveBody_t.Url_id, receiveBody_t.Url)
			}
		}
		// the length of receiving satisfied with Content_Length, close connection initiatively.
		if (o_content_length > 0) && ((total_length - head_length - 4) >= o_content_length) {
			tcpconn.Close()
			log.Debugf("GetProxy_https the length of receiving satisfied with Content_Length, close connection initiatively. total_length: %d|| head_length: %d|| o_content_length: %d|| url_id: %s|| url: %s", total_length, head_length, o_content_length, receiveBody_t.Url_id, receiveBody_t.Url)
			// get Content_length and Download_mean_rate
			receiveBody_t.Content_length = total_length - head_length - 4
			delta_t := (float64(time.Now().UnixNano()/1e6) - st) / 1000
			var d_rate float64
			if receiveBody_t.Content_length < 0 {
				d_rate = 0
			} else {
				if delta_t <= 0 {
					d_rate = float64(receiveBody_t.Content_length) / 1024
				} else {
					d_rate = float64(receiveBody_t.Content_length) / 1024 / float64(delta_t)
				}
			}
			receiveBody_t.Download_mean_rate = strconv.FormatFloat(d_rate, 'f', 6, 64)
			log.Debugf("GetProxy_https total_length: %d|| delta_t: %f|| url_id: %s|| url: %s", total_length, delta_t, receiveBody_t.Url_id, receiveBody_t.Url)
			receiveBody_t.Status = "Done."
			go PostRealBody(receiveBody_t)
			if receiveBody_t.Conn != 0 {
				check_conn_out(receiveBody_t)
			}
			<-maxConcurrent_ch
			break
		}
		//fmt.Println("Rec[",conn.RemoteAddr().String(),"] Say :" ,string(buf[0:lenght]))
		// reciveStr :=string(buf[0:lenght])
		// if reciveStr != nil {}
	}
	// receiveBody_t.Status = "success"
	// go PostRealBody(receiveBody_t)
	// <-maxConcurrent_ch
}

func Process_go(body ReceiveBody) {
	maxConcurrent_ch <- 1
	is_http_t := is_http(body.Url)
	log.Debugf("Process_go Url_id: %s|| url: %s|| Ip_to: %s|| Head_send: %s|| is_http_t: %t", body.Url_id, body.Url, Ip_to, Head_send, is_http_t)
	if is_http_t == true {
		GetProxy_http(Ip_to, Head_send, body)
	} else {
		GetProxy_https(Ip_to, Head_send, body)
	}
	log.Debugf("Process_go [finished] Url_id: %s|| status: %s|| Url: %s", body.Url_id, body.Status, body.Url)
}

func transmitIn(connKey string) {
	connIn := <-ConnInChan
	ConnInMap[connKey] = connIn
}

func ProcessConnTask(body ReceiveBody) {
	// var ConnMap = make(map[string]int) // limit channel number
	var waitChan = make(chan int, body.Conn)
	var CountConnIn = make(chan int, body.Conn)
	channel_name := getChannelName(body.Url)
	conn := body.Conn
	connKey := channel_name + "__" + fmt.Sprintf("%d", conn)
	connIn, is_existed := ConnInMap[connKey]
	if !is_existed {
		connIn.ConnChan = waitChan
		connIn.CountConnIn = CountConnIn
		log.Debugf("ProcessConnTask [New Conn-domain comes in.] connKey: %s|| body: %+v", connKey, body)
	}
	connIn.ConnChan <- 1
	connIn.CountConnIn <- 1
	connIn.NumIn += <-connIn.CountConnIn
	ConnInChan <- connIn
	go transmitIn(connKey)
	log.Debugf("ProcessConnTask ConnInMap: %+v|| body: %+v", ConnInMap, body)
	Process_go(body)
	// ---------------------------------------
	// var waitChan = make(chan int, body.Conn)
	// var CountConnIn = make(chan int, body.Conn)
	// channel_name := getChannelName(body.Url)
	// conn := body.Conn
	// connKey := channel_name + "__" + fmt.Sprintf("%d", conn)
	// connIn, is_existed := ConnInMap[connKey]
	// if !is_existed {
	// 	connIn.ConnChan = waitChan
	// 	connIn.CountConnIn = CountConnIn
	// 	log.Debugf("ProcessConnTask [New Conn-domain comes in.] connKey: %s|| body: %+v", connKey, body)
	// }
	// connIn.ConnChan <- 1
	// connIn.CountConnIn <- 1
	// connIn.NumIn += <-connIn.CountConnIn
	// ConnInMap[connKey] = connIn
	// log.Debugf("ProcessConnTask ConnInMap: %+v|| body: %+v", ConnInMap, body)
	// Process_go(body)
}

/*TaskRequestPost 处理实时接收汇报的请求，并异步判断请求 */
func TaskRequestPost(writer http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		nonPostData, _ := ioutil.ReadAll(r.Body)
		log.Debugf("TaskRequestPost Your request method %s is forbidden.(POST recommanded)|| nonPostData: %s", r.Method, nonPostData)
		return
	} else {
		var data DataFromCenter
		result, _ := ioutil.ReadAll(r.Body)
		r.Body.Close()
		log.Debugf("TaskRequestPost type(request_body): %T|| request_body: %s", result, result)
		errUnmarshal := json.Unmarshal(result, &data)
		if errUnmarshal != nil {
			log.Error("TaskRequestPost errUnmarshal: ", errUnmarshal)
		}
		body, ack := NewReceiveBody(data) // originally from lua
		log.Debugf("TaskRequestPost RemoteAddr: %s|| URL.Path: %s|| data: %+v|| body: %+v|| ack: %+v", r.RemoteAddr, r.URL.Path, data, body, ack)
		// ack reply center
		writer.WriteHeader(200)
		msg, _ := json.Marshal(ack)
		_, errWrite := writer.Write([]byte(msg))
		if errWrite != nil {
			http.Error(writer, "Interal ERROR: ", 500)
		}

		for i := 0; i < len(body); i++ {
			// limit channel number for limiting rate
			if body[i].Conn != 0 {
				go ProcessConnTask(body[i])
			} else {
				go Process_go(body[i])
			}
		}

	}

	// -------------
	// var body []ReceiveBody
	// result, _ := ioutil.ReadAll(r.Body)
	// r.Body.Close()
	// log.Debugf("TaskRequestPost request_body: %s", result)
	// errUnmarshal := json.Unmarshal(result, &body)
	// if errUnmarshal != nil {
	// 	log.Error("TaskRequestPost errUnmarshal: ", errUnmarshal)
	// }
	// log.Debugf("TaskRequestPost RemoteAddr: %s|| URL.Path: %s|| body: %+v", r.RemoteAddr, r.URL.Path, body)
	// time.Sleep(1)
	// for i := 0; i < len(body); i++ {
	// 	go Process_go(body[i])
	// }
	// writer.WriteHeader(200)
	// var param = make(map[string]interface{})
	// param["msg"] = "ok"
	// msg, _ := json.Marshal(param)
	// _, errWrite := writer.Write([]byte(msg))
	// if errWrite != nil {
	// 	http.Error(writer, "Interal ERROR: ", 500)
	// }
}

func PostRealBody(data ReceiveBody) {
	b, err := json.Marshal(data)
	if err != nil {
		log.Debugf("PostRealBody json err:%s", err)
	}
	body := bytes.NewBuffer([]byte(b))
	log.Debugf("PostRealBody body: %+v", body)
	Report_address := "http://" + data.Report_ip + ":" + data.Report_port + "/content/preload_report"
	for i := 0; i < 4; i++ {
		//可以通过client中transport的Dial函数,在自定义Dial函数里面设置建立连接超时时长和发送接受数据超时
		client := &http.Client{
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					conn, err := net.DialTimeout(netw, addr, time.Second*3) //设置建立连接超时
					if err != nil {
						return nil, err
					}
					conn.SetDeadline(time.Now().Add(time.Second * 5)) //设置发送接受数据超时
					return conn, nil
				},
				ResponseHeaderTimeout: time.Second * 2,
			},
		}
		if i > 0 {
			log.Debugf("PostRealBody retried No.%d times|| body: %s|| url_id: %s|| url: %s", i, body, data.Url_id, data.Url)
		}
		request, errNewRequest := http.NewRequest("POST", Report_address, body) //提交请求;用指定的方法，网址，可选的主体放回一个新的*Request
		if errNewRequest != nil {
			log.Debugf("PostRealBody retried No.%d times|| errNewRequest: %s|| url_id: %s|| url: %s", i, errNewRequest, data.Url_id, data.Url)
			time.Sleep(1e9)
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Host", "www.report.com")
		request.Header.Set("Accept", "*/*")
		request.Header.Set("User-Agent", "ChinaCache")
		request.Header.Set("X-CC-Preload-Report", "ChinaCache")
		request.Header.Set("Content-Length", fmt.Sprintf("%d", len(b)))
		request.Header.Set("Connection", "close")
		response, errDo := client.Do(request) //前面预处理一些参数，状态，Do执行发送；处理返回结果;Do:发送请求,
		if errDo != nil {
			log.Debugf("PostRealBody retried No.%d times, errDo: %s|| request: %+v, Report_address: %s|| url_id: %s|| url: %s", i, errDo, request, Report_address, data.Url_id, data.Url)
			time.Sleep(1e9)
			continue
		}
		r_status := response.StatusCode //获取返回状态码，正常是200
		if r_status != 200 {
			log.Debugf("PostRealBody retried No.%d times, r_status: %d, url_id: %s|| url: %s", i, r_status, data.Url_id, data.Url)
			time.Sleep(1e9)
			continue
		}
		defer response.Body.Close()
		r_body, errReadAll := ioutil.ReadAll(response.Body)
		if errReadAll != nil {
			log.Debugf("PostRealBody retried No.%d times, errReadAll: %s|| url_id: %s|| url: %s", i, errReadAll, data.Url_id, data.Url)
			return
		}
		log.Debugf("PostRealBody retried No.%d times, r_status: %d, r_body: %s|| url_id: %s|| url: %s", i, r_status, r_body, data.Url_id, data.Url)
		break
	}

	// ---------------------
	// original http.post
	// rb.IP = fmt.Sprintf("192.168.1.%d", i)
	// b, err := json.Marshal(data)
	// if err != nil {
	// 	log.Debugf("PostRealBody json err:%s", err)
	// }
	// body := bytes.NewBuffer([]byte(b))
	// log.Debugf("PostRealBody body: %+v", body)
	// Report_address := data.Report_ip + ":" + data.Report_port
	// for i := 0; i < 4; i++ {
	// 	resp, err := http.Post(Report_address, "APPLICATION/x-www-form-urlencoded", body)
	// 	log.Debugf("PostRealBody Report_address: %s", Report_address)
	// 	// resp, err := http.Post("http://127.0.0.1:8888/authentication", "APPLICATION/x-www-form-urlencoded", body)
	// 	if err != nil {
	// 		log.Debugf("PostRealBody[Post] retried No.%d times, error: %s|| url_id: %s|| url: %s", i, err, data.Url_id, data.Url)
	// 		time.Sleep(1e9)
	// 		continue
	// 	}
	// 	r_body, err := ioutil.ReadAll(resp.Body)
	// 	if err != nil {
	// 		// handle error
	// 		log.Debugf("PostRealBody[ReadAll] retried No.%d times, error: %s|| url_id: %s|| url: %s", i, err, data.Url_id, data.Url)
	// 	}
	// 	log.Debugf("PostRealBody resp retried No.%d times, r_body: %s|| url_id: %s|| url: %s", i, string(r_body), data.Url_id, data.Url)
	// 	defer resp.Body.Close()
	// 	break
	// }

}
