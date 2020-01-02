// version: 3.2.1
package core

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"sync"
	// "log"
	ut "PreloadGo/utils"
	// "net/url"
	"bytes"
	"crypto/tls"
	"net"
	"net/http"
	// "reflect"
	"strings"
	"time"
	// "bufio"
	// log "github.com/Sirupsen/logrus"
	// log "github.com/omidnikta/logrus"
	// "gopkg.in/mgo.v2"
	// "gopkg.in/mgo.v2/bson"
	"strconv"
)

var log = ut.Logger
var maxConcurrent_num = ut.MaxConcurrent
var maxConcurrent_ch = ut.MaxConcurrent_ch
var TsConcurrent_num = ut.TsConcurrent
var TsConcurrent_ch = ut.TsConcurrent_ch
var ConnConcurrent_num = ut.ConnConcurrent
var ConnConcurrent_ch = ut.ConnConcurrent_ch
var Head_send = "User-Agent:Mozilla/5.0 (Windows NT 6.3; WOW64 PRELOAD)"
var Ip_to = ut.Ip_to
var Read_time_out = ut.Read_time_out
var ssl_ip_to = ut.Ssl_ip_to
var ssl_ip_host = ut.Ssl_ip_host

// var TsOutputChan = make(chan map[string]interface{}, TsConcurrent_num)

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
	Switch_m3u8         bool `whether do little ts in m3u8 file`
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
	Conn               int                 `json:"conn"`
	Header_list        []map[string]string `json:"header_list"`
	Switch_m3u8        bool                `json:"switch_m3u8" whether do little ts in m3u8 file`
}

type ConnInStruct struct {
	Url    string
	Url_id string
	Conn   int
	Status string `UNPROCESSED`
}

type ConnOutStruct struct {
	Url    string
	Url_id string
	Conn   int
	Status string `record the status whether finished.`
}

type ConnStruct struct {
	TasksIn  []ConnInStruct
	InNum    int
	ConnChan chan int
	TasksOut []ConnOutStruct
	OutNum   int
	// Otype    string
}

type ConnTasks struct {
	mu    sync.Mutex
	Tasks map[string]ConnStruct
}

func (ct *ConnTasks) write(conn_map map[string]ConnStruct) {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	var conn_struct ConnStruct
	if ct.Tasks == nil {
		log.Debugf("ConnTasks write [ct.Tasks == nil] ct: %+v", ct)
		ct.Tasks = make(map[string]ConnStruct)
	}
	for k, v := range conn_map {
		task, is_existed := ct.Tasks[k]
		if !is_existed {
			conn_struct.TasksIn = append(conn_struct.TasksIn, v.TasksIn...)
			conn_struct.InNum += v.InNum
			conn_struct.ConnChan = v.ConnChan
			conn_struct.TasksOut = append(conn_struct.TasksOut, v.TasksOut...)
			conn_struct.OutNum += v.OutNum
			ct.Tasks[k] = conn_struct
		} else {
			task.TasksIn = append(task.TasksIn, v.TasksIn...)
			task.InNum += v.InNum
			task.TasksOut = append(task.TasksOut, v.TasksOut...)
			task.OutNum += v.OutNum
			ct.Tasks[k] = task
		}
	}
}

var ConnTaskStruct ConnTasks

type M3U8_struct struct {
	uid          string
	fetch_num    int
	url_filtered []string
}

type Ts_struct struct {
	Ts_uid      string
	Url         string
	Http_status int
	Body_length int
	Status      string
}

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
		body.Switch_m3u8 = data.Switch_m3u8
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

func RandInt(n int) int {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	x := r.Intn(n)
	if x == 0 {
		return RandInt(n)
	}
	return x
}

func splitReportAddress(report_address string) (string, string) {
	report_arr := strings.SplitN(report_address, ":", 2)
	return report_arr[0], report_arr[1]
}

func GetHost(str string) string {
	var host string
	array_str := strings.SplitN(str, "/", 4)
	if len(array_str) < 3 {
		log.Debugf("parse str error:", str)
		host = "127.0.0.1"

	} else {
		host = array_str[2]
	}

	return host
}

// judge url http or https
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
		if strings.Contains(s_line, "Content-Length:") {
			theContentLength, err = strconv.Atoi(strings.Replace(strings.SplitN(s_line, ":", 2)[1], " ", "", -1))
			if err != nil {
				log.Debugf("get_head_length theContentLength convert to int err: %s|| s_line: %s", err, s_line)
			}
		}
	}
	http_status, err := strconv.Atoi(strings.SplitN(first_line, " ", 3)[1])
	if err != nil {
		log.Debugf("get_head_length http_status convert to int error.http_status: %s", err)
	}
	head_length := len(thehead)
	return thehead, head_length, http_status, theContentLength
}

func check_conn_out(body ReceiveBody) {
	connKey := getChannelName(body.Url) + "__" + fmt.Sprintf("%d", body.Conn)

	var conn_out_struct ConnOutStruct
	conn_out_struct.Url_id = body.Url_id
	conn_out_struct.Url = body.Url
	conn_out_struct.Conn = body.Conn
	conn_out_struct.Status = body.Status

	var conn_struct ConnStruct
	conn_struct.OutNum += 1
	conn_struct.TasksOut = append(conn_struct.TasksOut, conn_out_struct)

	var conn_map = make(map[string]ConnStruct)
	conn_map[connKey] = conn_struct

	ConnTaskStruct.write(conn_map)
	log.Debugf("check_conn_out ConnTaskStruct: %+v|| Url_id: %s|| Url: %s", ConnTaskStruct, body.Url_id, body.Url)

	if ConnTaskStruct.Tasks[connKey].InNum == ConnTaskStruct.Tasks[connKey].OutNum {
		delete(ConnTaskStruct.Tasks, connKey)
		log.Debugf("check_conn_out [NumIn==NumOut,delete the key successfully.]|| connKey: %s|| ConnTaskStruct: %+v|| Url_id: %s|| Url: %s", connKey, ConnTaskStruct, body.Url_id, body.Url)
	}

	<-ConnConcurrent_ch
	<-ConnTaskStruct.Tasks[connKey].ConnChan
}

func ProxyRequest(ip_to string, head string, url_id string, url string, read_flag bool) string {
	channel_name := getChannelName(url)
	tcpAddr, errResolveTCPAddr := net.ResolveTCPAddr("tcp4", ip_to)
	if errResolveTCPAddr != nil {
		log.Debugf("ProxyRequest errResolveTCPAddr: %s", errResolveTCPAddr)
	}

	tcpconn, errDialTCP := net.DialTCP("tcp", nil, tcpAddr)
	if errDialTCP != nil {
		log.Debugf("ProxyRequest errDialTCP: %s", errDialTCP)
	}

	// str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + head + conn_header + "\r\n\r\n"
	str_t := fmt.Sprintf("GET %s HTTP/1.1\r\nHost: %s\r\n%s\r\nAccept: */*\r\nProxy-Connection: Keep-Alive\r\n\r\n", url, channel_name[7:], head)
	log.Debugf("ProxyRequest str_t: %s", str_t)
	//向tcpconn中写入数据
	_, errWrite := tcpconn.Write([]byte(str_t))
	if errWrite != nil {
		log.Debugf("ProxyRequest errWrite: %s", errWrite)
	}
	buf := make([]byte, 8192)
	tcpconn.SetDeadline(time.Now().Add(time.Duration(10) * time.Second))

	head_end := false
	var (
		content                 string
		thehead                 string
		h_length                int
		http_status             int
		original_content_length int
		o_content_length        int
		total_length            int
		head_length             int
	)

	for {
		length, errRead := tcpconn.Read(buf)
		if errRead != nil {
			tcpconn.Close()
			log.Debugf("ProxyRequest errRead: %s", errRead)
			if read_flag {
				return content
			}
			break
		}

		if !head_end {
			recvStr := string(buf[:length])
			is_head_end := strings.Contains(recvStr, "\r\n\r\n")
			if !is_head_end {
				head_length += length
			} else {
				thehead, h_length, http_status, original_content_length = get_head_length(recvStr)
				o_content_length = original_content_length
				head_length += h_length
				head_end = true
				log.Debugf("ProxyRequest o_content_length: %d|| head_length: %d|| http_status: %d|| thehead: %s", o_content_length, head_length, http_status, thehead)
			}
		}

		if length > 0 {
			total_length += length
			if read_flag {
				content += string(buf[:length])
			}
			tcpconn.SetDeadline(time.Now().Add(time.Duration(10) * time.Second))
		}

		// the length received satisfied with Content_Length, close connection initiatively.
		if (o_content_length > 0) && ((total_length - head_length - 4) >= o_content_length) {
			tcpconn.Close()
			log.Debugf("ProxyRequest [the length of receiving satisfied with Content_Length, close connection initiatively.] total_length: %d|| head_length: %d|| o_content_length: %d", total_length, head_length, o_content_length)
			if read_flag {
				return content
			}
			break
		}
	}
	return "ProxyRequest FINISHED."
}

func ltsProxyRequest(ip_to string, head string, url_id string, url string, is_little_ts bool, countRainbowChan chan int, TsOutputChan chan Ts_struct) {
	channel_name := getChannelName(url)
	tcpAddr, errResolveTCPAddr := net.ResolveTCPAddr("tcp4", ip_to)
	if errResolveTCPAddr != nil {
		log.Debugf("ltsProxyRequest errResolveTCPAddr: %s", errResolveTCPAddr)
	}

	tcpconn, errDialTCP := net.DialTCP("tcp", nil, tcpAddr)
	if errDialTCP != nil {
		log.Debugf("ltsProxyRequest errDialTCP: %s", errDialTCP)
	}

	// str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + head + conn_header + "\r\n\r\n"
	str_t := fmt.Sprintf("GET %s&vkey=3039134e23ab17addeff8979e8bc5e43 HTTP/1.1\r\nHost: %s\r\n%s\r\nAccept: */*\r\nProxy-Connection: Keep-Alive\r\n\r\n", url, channel_name[7:], head)
	log.Debugf("ltsProxyRequest str_t: %s", str_t)
	//向tcpconn中写入数据
	_, errWrite := tcpconn.Write([]byte(str_t))
	if errWrite != nil {
		log.Debugf("ltsProxyRequest errWrite: %s", errWrite)
	}
	buf := make([]byte, 8192)
	tcpconn.SetDeadline(time.Now().Add(time.Duration(10) * time.Second))

	body_length := 0
	head_end := false
	var (
		thehead                 string
		h_length                int
		http_status             int
		original_content_length int
		o_content_length        int
		total_length            int
		head_length             int
	)

	for {
		length, errRead := tcpconn.Read(buf)
		if errRead != nil {
			tcpconn.Close()
			log.Debugf("ltsProxyRequest errRead: %s|| url: %s", errRead, url)
			if errRead == io.EOF {
				if is_little_ts {
					var ts_st Ts_struct
					ts_st.Ts_uid = url_id
					ts_st.Url = url
					ts_st.Http_status = http_status // HTTP status code from header
					ts_st.Body_length = body_length // Content-Length from header
					ts_st.Status = "Done..."
					TsOutputChan <- ts_st
					countRainbowChan <- 1
					<-TsConcurrent_ch
				}
			}
			break
		}

		if !head_end {
			recvStr := string(buf[:length])
			is_head_end := strings.Contains(recvStr, "\r\n\r\n")
			if !is_head_end {
				head_length += length
			} else {
				thehead, h_length, http_status, original_content_length = get_head_length(recvStr)
				o_content_length = original_content_length
				head_length += h_length
				head_end = true
				log.Debugf("ltsProxyRequest url: %s o_content_length: %d|| head_length: %d|| http_status: %d|| thehead: %s", url, o_content_length, head_length, http_status, thehead)
			}
		}

		if length > 0 {
			total_length += length
			tcpconn.SetDeadline(time.Now().Add(time.Duration(10) * time.Second))
		}

		// the length received satisfied with Content_Length, close connection initiatively.
		if (o_content_length > 0) && ((total_length - head_length - 4) >= o_content_length) {
			tcpconn.Close()
			log.Debugf("ltsProxyRequest url: %s|| [the length of receiving satisfied with Content_Length, close connection initiatively.] total_length: %d|| head_length: %d|| o_content_length: %d", url, total_length, head_length, o_content_length)
			body_length = total_length - head_length - 4
			log.Debugf("ltsProxyRequest body_length: %d", body_length)
			if is_little_ts {
				var ts_st Ts_struct
				ts_st.Ts_uid = url_id
				ts_st.Url = url
				ts_st.Http_status = http_status // HTTP status code from header
				ts_st.Body_length = body_length // Content-Length from header
				ts_st.Status = "Done.."
				countRainbowChan <- 1
				TsOutputChan <- ts_st
				<-TsConcurrent_ch
				log.Debugf("ltsProxyRequest url: %s|| ts_st: %+v", url, ts_st)
			}
			break
		}
	}
}

func GetProxy_http(ip_to string, head string, receiveBody_t ReceiveBody) {
	// channel_name := getChannelName(receiveBody_t.Url)
	tcpAddr, errResolveTCPAddr := net.ResolveTCPAddr("tcp4", ip_to)
	if errResolveTCPAddr != nil {
		log.Debugf("GetProxy_http errResolveTCPAddr: %s|| url_id: %s|| url: %s", errResolveTCPAddr, receiveBody_t.Url_id, receiveBody_t.Url)
	}

	tcpconn, errDialTCP := net.DialTCP("tcp", nil, tcpAddr)
	if errDialTCP != nil {
		log.Debugf("GetProxy_http errDialTCP: %s|| url_id: %s|| url: %s", errDialTCP, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", errDialTCP)
		PostRealBody(receiveBody_t)
		if receiveBody_t.Conn != 0 {
			check_conn_out(receiveBody_t)
		}
		<-maxConcurrent_ch
		return
	}
	host := GetHost(receiveBody_t.Url)
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
		PostRealBody(receiveBody_t)
		if receiveBody_t.Conn != 0 {
			check_conn_out(receiveBody_t)
		}
		<-maxConcurrent_ch
		return

	}
	buf := make([]byte, 8192)
	log.Debugf("GetProxy_http Read_time_out: %d", Read_time_out)
	tcpconn.SetDeadline(time.Now().Add(time.Duration(Read_time_out) * time.Second))

	head_end := false
	var (
		thehead                 string
		h_length                int
		http_status             int
		original_content_length int
		o_content_length        int
		total_length            int
		head_length             int
		body_first_start        int
	)
	st := float64(time.Now().UnixNano() / 1e6)
	md5Ctx := md5.New()
	body_first := false
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
			PostRealBody(receiveBody_t)
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
			recvStr := string(buf[:length])
			is_head_end := strings.Contains(recvStr, "\r\n\r\n")
			if !is_head_end {
				head_length += length
			} else {
				thehead, h_length, http_status, original_content_length = get_head_length(recvStr)
				o_content_length = original_content_length
				head_length += h_length
				head_end = true
				receiveBody_t.Http_status = http_status
				log.Debugf("GetProxy_http o_content_length: %d|| head_length: %d|| http_status: %d|| thehead: %s|| url_id: %s|| url: %s", o_content_length, head_length, http_status, thehead, receiveBody_t.Url_id, receiveBody_t.Url)
			}
		}

		// body begin.make the md5 checksum.
		if total_length >= head_length+4 {
			if !body_first {
				body_first = true
				body_first_start = head_length + 4
				log.Debugf("GetProxy_http body_first_start: %d|| url_id: %s|| url: %s", body_first_start, receiveBody_t.Url_id, receiveBody_t.Url)
				if body_first_start != 0 {
					md5Ctx.Write(buf[body_first_start:length])
				}
			} else {
				md5Ctx.Write(buf[:length])
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
			// output md5 checksum
			cipherStr := md5Ctx.Sum(nil)
			receiveBody_t.Check_result = hex.EncodeToString(cipherStr)
			if receiveBody_t.Conn != 0 {
				check_conn_out(receiveBody_t)
			}
			PostRealBody(receiveBody_t)
			<-maxConcurrent_ch
			break
		}
	}
}

// https
func GetProxy_https(ip_to string, head string, receiveBody_t ReceiveBody) {
	// channel_name := getChannelName(receiveBody_t.Url)
	host := GetHost(receiveBody_t.Url)
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
		PostRealBody(receiveBody_t)
		if receiveBody_t.Conn != 0 {
			check_conn_out(receiveBody_t)
		}
		<-maxConcurrent_ch
		return
	}

	//向tcpconn中写入数据
	_, err3 := tcpconn.Write([]byte(str_t))
	if err3 != nil {
		log.Debugf("GetProxy_https err3: %s|| url_id: %s|| url: %s", err3, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err3)
		PostRealBody(receiveBody_t)
		if receiveBody_t.Conn != 0 {
			check_conn_out(receiveBody_t)
		}
		<-maxConcurrent_ch
		return
	}
	buf := make([]byte, 8192)
	log.Debugf("GetProxy_https Read_time_out: %s|| url_id: %s|| url: %s", Read_time_out, receiveBody_t.Url_id, receiveBody_t.Url)
	tcpconn.SetDeadline(time.Now().Add(time.Duration(Read_time_out) * time.Second))

	head_end := false
	var (
		thehead                 string
		h_length                int
		http_status             int
		original_content_length int
		o_content_length        int
		total_length            int
		head_length             int
	)
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
			PostRealBody(receiveBody_t)
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
				thehead, h_length, http_status, original_content_length = get_head_length(recvStr)
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
			PostRealBody(receiveBody_t)
			if receiveBody_t.Conn != 0 {
				check_conn_out(receiveBody_t)
			}
			<-maxConcurrent_ch
			break
		}
	}
}

func Process_go(body ReceiveBody) {
	if body.Conn == 0 {
		maxConcurrent_ch <- 1
	}
	is_http_t := is_http(body.Url)
	log.Debugf("Process_go Url_id: %s|| url: %s|| Ip_to: %s|| Head_send: %s|| is_http_t: %t", body.Url_id, body.Url, Ip_to, Head_send, is_http_t)
	if is_http_t == true {
		GetProxy_http(Ip_to, Head_send, body)
	} else {
		GetProxy_https(Ip_to, Head_send, body)
	}
	log.Debugf("Process_go [finished] Url_id: %s|| status: %s|| Url: %s", body.Url_id, body.Status, body.Url)
}

func ProcessConnTask(body ReceiveBody) {
	ConnConcurrent_ch <- 1
	// init the conn struct
	conn := body.Conn
	channel_name := getChannelName(body.Url)
	connKey := channel_name + "__" + fmt.Sprintf("%d", conn)

	var conn_in_struct ConnInStruct
	conn_in_struct.Url_id = body.Url_id
	conn_in_struct.Url = body.Url
	conn_in_struct.Conn = body.Conn
	conn_in_struct.Status = "UNPROCESSED"

	var conn_struct ConnStruct
	conn_struct.TasksIn = append(conn_struct.TasksIn, conn_in_struct)
	conn_struct.InNum += 1

	_, is_existed := ConnTaskStruct.Tasks[connKey]
	if !is_existed {
		conn_struct.ConnChan = make(chan int, conn)
	}

	var conn_map = make(map[string]ConnStruct)
	conn_map[connKey] = conn_struct

	ConnTaskStruct.write(conn_map)

	ConnTaskStruct.Tasks[connKey].ConnChan <- 1

	log.Debugf("ProcessConnTask [Initialization done.] ConnTaskStruct: %+v|| Url_id: %+v|| Url: %+v", ConnTaskStruct, body.Url_id, body.Url)

	Process_go(body)
}

// this function is suitable for qq's m3u8 tasks
func get_m3u8_qq(url_id string, url string) M3U8_struct {
	var theM3VKEY = ".m3u8?vkey=3039134e23ab17addeff8979e8bc5e43"
	var channel_name = getChannelName(url)
	var fetch_num int
	var url_filtered []string
	// var rainbow_map = make(map[string]interface{})
	var m3u8_struct M3U8_struct
	var file_name string
	// log.Debugf("channel_name: %s|| url_id: %s", channel_name, url_id)
	url = strings.TrimSpace(url)
	url = strings.SplitN(url, "?", 2)[0]
	// ex1: http://ltslx.qq.com/w0025mpfimu.321004.1.ts
	// ex2: http://ltslx.qq.com/u0025s9uq4c.320001.ts
	if strings.HasSuffix(url, ".ts") {
		path_part := strings.SplitN(url, "/", 4)[3]
		dotArray := strings.Split(path_part, ".")
		if len(dotArray) == 4 {
			file_name = fmt.Sprintf("%s.%s.%s", dotArray[0], dotArray[1], dotArray[3])
		} else if len(dotArray) == 3 {
			file_name = fmt.Sprintf("%s.%s.%s", dotArray[0], dotArray[1], dotArray[2])
		}
		url_m3u8 := fmt.Sprintf("%s/%s%s", channel_name, file_name, theM3VKEY)
		log.Debugf("get_m3u8_qq url_id: %s|| url: %s|| path_part: %s|| file_name: %s|| url_m3u8: %s", url_id, url, path_part, file_name, url_m3u8)
		file_m3u8 := ProxyRequest(Ip_to, Head_send, url_id, url_m3u8, true)
		log.Debugf("get_m3u8_qq url_id: %s|| url: %s|| len(file_m3u8): %d", url_id, url, len(file_m3u8))
		for _, url_front_all := range strings.Split(file_m3u8, "\n") {
			if strings.Contains(url_front_all, path_part) {
				url_ts := url_front_all
				fetch_num += 1
				url_filtered = append(url_filtered, fmt.Sprintf("%s/%s", channel_name, url_ts))
			}
		}
		m3u8_struct.uid = url_id
		m3u8_struct.fetch_num = fetch_num
		m3u8_struct.url_filtered = url_filtered
	}
	return m3u8_struct
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

		// process all kinds of tasks.
		for i := 0; i < len(body); i++ {
			if !body[i].Switch_m3u8 {
				if body[i].Conn != 0 {
					go ProcessConnTask(body[i])
				} else {
					go Process_go(body[i])
				}
			} else {
				body[i].Http_status = -1
				go PostRealBody(body[i])
			}
		}

	}
}

func PostRealBody(data ReceiveBody) {
	var (
		request       *http.Request
		errNewRequest error
		response      *http.Response
		errDo         error
		r_status      int
	)

	for i := 0; i < 4; i++ {

		b, err := json.Marshal(data)
		if err != nil {
			log.Debugf("PostRealBody json err:%s", err)
		}
		body := bytes.NewBuffer([]byte(b))
		log.Debugf("PostRealBody body: %+v", body)
		Report_address := "http://" + data.Report_ip + ":" + data.Report_port + "/content/preload_report?id=" + data.Url_id

		//可以通过client中transport的Dial函数,在自定义Dial函数里面设置建立连接超时时长和发送接受数据超时
		client := &http.Client{
			Transport: &http.Transport{
				Dial: func(netw, addr string) (net.Conn, error) {
					conn, err := net.DialTimeout(netw, addr, time.Second*3) //设置建立连接超时
					if err != nil {
						return nil, err
					}
					conn.SetDeadline(time.Now().Add(time.Second * 5)) //设置发送接收数据超时
					return conn, nil
				},
				ResponseHeaderTimeout: time.Second * 2,
			},
		}
		if i > 0 {
			log.Debugf("PostRealBody body: %s|| url_id: %s|| url: %s", body, data.Url_id, data.Url)
		}
		request, errNewRequest = http.NewRequest("POST", Report_address, body) //提交请求;用指定的方法，网址，可选的主体返回一个新的*Request
		if errNewRequest != nil {
			log.Debugf("PostRealBody errNewRequest: %s|| url_id: %s|| url: %s", errNewRequest, data.Url_id, data.Url)
			time.Sleep(time.Second * time.Duration(RandInt(7)))
			continue
		}
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Host", "www.report.com")
		request.Header.Set("Accept", "*/*")
		request.Header.Set("User-Agent", "ChinaCache")
		request.Header.Set("X-CC-Preload-Report", "ChinaCache")
		request.Header.Set("Content-Length", fmt.Sprintf("%d", len(b)))
		request.Header.Set("Connection", "close")

		response, errDo = client.Do(request) //前面预处理一些参数，状态，Do执行发送；处理返回结果;Do:发送请求,
		if errDo != nil {
			log.Debugf("PostRealBody retried No.%d times, errDo: %s|| request: %+v, Report_address: %s|| url_id: %s|| url: %s", i, errDo, request, Report_address, data.Url_id, data.Url)
			time.Sleep(time.Second * time.Duration(RandInt(7)))
			continue
		}
		r_status = response.StatusCode //获取返回状态码，正常是200
		if r_status != 200 {
			log.Debugf("PostRealBody retried No.%d times, r_status: %d, url_id: %s|| url: %s", i, r_status, data.Url_id, data.Url)
			time.Sleep(time.Second * time.Duration(RandInt(7)))
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

	// Parse m3u8 and preload the little ts tasks.
	if data.Switch_m3u8 {
		var countRainbowChan = make(chan int, TsConcurrent_num)
		var TsOutputChan = make(chan Ts_struct, TsConcurrent_num)
		m3u8_struct := get_m3u8_qq(data.Url_id, data.Url)

		log.Debugf("[lts tasks processing.] m3u8_struct: %+v|| url_id: %+v", m3u8_struct, data.Url_id)

		for _, url_ts := range m3u8_struct.url_filtered {
			TsConcurrent_ch <- 1
			go ltsProxyRequest(Ip_to, Head_send, data.Url_id, url_ts, true, countRainbowChan, TsOutputChan)
		}
		log.Debugf("[lts tasks processing.] go ltsProxyRequest(Ip_to, Head_send, data.Url_id, url_ts, true, countRainbowChan, TsOutputChan)")

		donelts := 0
		for numlts := range countRainbowChan {
			func(num int) {
				donelts += num
			}(numlts)
			if donelts == m3u8_struct.fetch_num {
				close(countRainbowChan)
			}
		}

		log.Debugf("[lts tasks processing.] donelts: %d", donelts)

		var lts_info []Ts_struct
		for m3_st := range TsOutputChan {
			func(m3_st Ts_struct) {
				lts_info = append(lts_info, m3_st)
			}(m3_st)
			if len(lts_info) == donelts {
				close(TsOutputChan)
			}
		}

		log.Debugf("[lts tasks processing.] url_id: %s|| progress_bar[%d/%d]|| lts_info: %+v", data.Url_id, donelts, m3u8_struct.fetch_num, lts_info)
	}

}
