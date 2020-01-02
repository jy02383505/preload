// version: 2.4
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
var maxConcurrent_ch = ut.MaxConcurrent_ch

// var Head_send = ut.Head
var Head_send = "User-Agent:Mozilla/5.0 (Windows NT 6.3; WOW64 PRELOAD)"
var Ip_to = ut.Ip_to

// var Report_address = ut.Report_address
var Read_time_out = ut.Read_time_out
var ssl_ip_to = ut.Ssl_ip_to
var ssl_ip_host = ut.Ssl_ip_host

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
	Header_list        []map[string]string `json:"header_list"`
}

// //RequestChannelBody 获取频道deny key时的参数
// type RequestChannelBody struct {
// 	Channels []string `json:"channels"`
// }

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

// judge url http or https
func is_http(str string) (is_http_t bool) {
	array_str := strings.SplitN(str, "/", 4)
	is_http_t = true
	if len(array_str) > 0 {
		// log.Debugf("is_http url: %s, array_str[0]: %s", str, array_str[0])
		if array_str[0] == "https:" {
			is_http_t = false
		}

	}
	return is_http_t
}

// func GetProxy(ip_to string, head string, receiveBody_t ReceiveBody){
//     //time1 := time.Now().Unix()
//    // fmt.Println("test2")
//     proxy := func(_ *http.Request) (*url.URL, error){
// 	//return url.Parse("http://58.215.108.210:80")
// 	//return url.Parse("http://58.215.108.19:80")
// 	return url.Parse(ip_to)
//         }
//     //transport := &http.Transport{Proxy: proxy}
//     //client := &http.Client{Transport: transport}
//     //transport := &http.Transport{Proxy: proxy, ResponseHeaderTimeout : 1 * time.Second,
//     //    ExpectContinueTimeout: 1 * time.Second, DisableKeepAlives: true}
//     // transport := &http.Transport{Proxy: proxy, DisableKeepAlives: false}
//         transport := &http.Transport{Proxy: proxy}
//     // client := &http.Client{Transport: transport, Timeout: 15 * time.Second}
//         client := &http.Client{Transport: transport}
//     // req, err := http.NewRequest("GET", receiveBody_t.Url, strings.NewReader(""))
//     req, err := http.NewRequest("GET", receiveBody_t.Url, nil)
//     head_array := strings.Split(head, ",")
//     //fmt.Println(head_array)
//     for i := 0; i < len(head_array); i++{
//         head_temp := strings.Split(head_array[i], ":")
//         if len(head_temp) != 2{
//             continue
//         }
//         req.Header.Set(head_temp[0], head_temp[1])
//         }
//     host, len_str := GetHost(receiveBody_t.Url)
//     if len_str >= 3 {
//         req.Header.Set("Host", host)
//         }else{
//             log.Debugf("host,%s" ,host)
//         }

//     //fmt.Println("head map",req.Header)
//     //req.Header.Set("User-Agent1", "sdfsdf")
//     //fmt.Println("test2", req.Header)

//      if err != nil {
//          //fmt.Println("111")
//      	log.Debugf("error 1")
//      	receiveBody_t.Status = "failed"
//      	go PostRealBody(receiveBody_t)
//          <- maxConcurrent_ch

//      }else{
//      resp, err := client.Do(req)
//       if err != nil {
//           //fmt.Println("get data error:", err)
//           //resp.Body.Close()
//       	log.Debugf("get data error:%s", err)
//       	log.Debugf("error 2")
//       	receiveBody_t.Status = "failed"
//       	go PostRealBody(receiveBody_t)

//           <- maxConcurrent_ch
//       }else{
//          if resp != nil {
//          	log.Debugf("header:%s, StatusCode:%s", resp.Header, resp.StatusCode)
//          }

//       //    scanner_n := bufio.NewScanner(body)
// 		    // all_line_count := 0
// 		    // for scanner_n.Scan(){
// 		    //      all_line_count++
// 		    // }

//          defer resp.Body.Close()
//          // io.Copy(ioutil.Discard,resp.Body)
//          body, err := ioutil.ReadAll(resp.Body)
//          if body != nil {
//          	// log.Debugf("body, %s", string(body))
//          }
//          if err != nil {}
//          //fmt.Println("zhixingdaoci")
//          log.Debugf("success")
//          receiveBody_t.Status = "success"
//          go PostRealBody(receiveBody_t)
//          <- maxConcurrent_ch
//            }
//     }

// }

func get_head_length(str string) (string, int, int) {
	req_arr := strings.SplitN(str, "\r\n\r\n", 2)
	thehead := req_arr[0]
	first_line := strings.SplitN(thehead, "\r\n", 2)[0]
	http_status, err := strconv.Atoi(strings.SplitN(first_line, " ", 3)[1])
	if err != nil {
		log.Debugf("get_head_length http_status convert to int error.http_status: %s", http_status)
	}
	head_length := len(thehead)
	return thehead, head_length, http_status
}

func GetProxy_http(ip_to string, head string, receiveBody_t ReceiveBody) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", ip_to)
	if err != nil {
		log.Debugf("GetProxy_http err: %s|| url_id: %s|| url: %s", err, receiveBody_t.Url_id, receiveBody_t.Url)
	}

	tcpconn, err2 := net.DialTCP("tcp", nil, tcpAddr)
	if err2 != nil {
		log.Debugf("GetProxy_http err2: %s|| url_id: %s|| url: %s", err2, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err2)
		go PostRealBody(receiveBody_t)
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
	if receiveBody_t.Is_compressed {
		str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + "Accept-Encoding:gzip\r\n" + head + "\r\n\r\n"
	} else {
		str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + head + "\r\n\r\n"
	}

	log.Debugf("GetProxy_http str_t: %s|| url_id: %s|| url: %s", str_t, receiveBody_t.Url_id, receiveBody_t.Url)
	//向tcpconn中写入数据
	_, err3 := tcpconn.Write([]byte(str_t))
	if err3 != nil {
		log.Debugf("GetProxy_http err3: %s|| url_id: %s|| url: %s", err3, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err3)
		go PostRealBody(receiveBody_t)
		<-maxConcurrent_ch
		return

	}
	buf := make([]byte, 8192)
	log.Debugf("GetProxy_http Read_time_out: %d", Read_time_out)
	tcpconn.SetDeadline(time.Now().Add(time.Duration(Read_time_out) * time.Second))

	total_length := 0
	head_end := false
	head_length := 0
	st := float64(time.Now().UnixNano() / 1e6)
	for {
		length, err := tcpconn.Read(buf)
		if err != nil {
			tcpconn.Close()
			log.Debugf("GetProxy_http err: %s|| url_id: %s|| url: %s", err, receiveBody_t.Url_id, receiveBody_t.Url)
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
			if err == io.EOF {
				receiveBody_t.Status = "Done"
			} else {
				receiveBody_t.Status = fmt.Sprintf("%s", err)
			}
			go PostRealBody(receiveBody_t)
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
				thehead, h_length, http_status := get_head_length(recvStr)
				head_length += h_length
				head_end = true
				receiveBody_t.Http_status = http_status
				log.Debugf("GetProxy_http head_length: %d, http_status: %d, thehead: %s, url_id: %s, url: %s", head_length, http_status, thehead, receiveBody_t.Url_id, receiveBody_t.Url)
			}
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
	host := GetHost_url(receiveBody_t.Url)
	var headers_str string
	for _, d := range receiveBody_t.Header_list {
		for k, v := range d {
			headers_str += k + ":" + v + "\r\n"
		}
	}
	log.Debugf("GetProxy_https headers_str: %s|| url_id: %s|| url: %s", headers_str, receiveBody_t.Url_id, receiveBody_t.Url)

	var str_t string
	if receiveBody_t.Is_compressed {
		str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + "Accept-Encoding:gzip\r\n" + head + "\r\n\r\n"
	} else {
		str_t = "GET " + receiveBody_t.Url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n" + headers_str + head + "\r\n\r\n"
	}
	log.Debugf("GetProxy_https str_t: %s, url_id: %s, url: %s", str_t, receiveBody_t.Url_id, receiveBody_t.Url)
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	tcpconn, err2 := tls.Dial("tcp", ssl_ip_to, conf)
	if err2 != nil {
		log.Debugf("GetProxy_https err2: %s, url_id: %s, url: %s", err2, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err2)
		go PostRealBody(receiveBody_t)
		<-maxConcurrent_ch
		return
	}

	//     log.Debugf("str_t:%s", str_t)
	//向tcpconn中写入数据
	_, err3 := tcpconn.Write([]byte(str_t))
	if err3 != nil {
		log.Debugf("GetProxy_https err3: %s, url_id: %s, url: %s", err3, receiveBody_t.Url_id, receiveBody_t.Url)
		receiveBody_t.Status = fmt.Sprintf("%s", err3)
		go PostRealBody(receiveBody_t)
		<-maxConcurrent_ch
		return
	}
	buf := make([]byte, 8192)
	log.Debugf("GetProxy_https Read_time_out: %s, url_id: %s, url: %s", Read_time_out, receiveBody_t.Url_id, receiveBody_t.Url)
	tcpconn.SetDeadline(time.Now().Add(time.Duration(Read_time_out) * time.Second))

	total_length := 0
	head_end := false
	head_length := 0
	st := float64(time.Now().UnixNano() / 1e6)
	for {
		length, err := tcpconn.Read(buf)
		if err != nil {
			tcpconn.Close()
			log.Debugf("GetProxy_https err: %s, url_id: %s, url: %s", err, receiveBody_t.Url_id, receiveBody_t.Url)
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
			log.Debugf("GetProxy_https total_length: %s, delta_t: %s, url_id: %s, url: %s", total_length, delta_t, receiveBody_t.Url_id, receiveBody_t.Url)
			if err == io.EOF {
				receiveBody_t.Status = "Done"
			} else {
				receiveBody_t.Status = fmt.Sprintf("%s", err)
			}
			go PostRealBody(receiveBody_t)
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
				thehead, h_length, http_status := get_head_length(recvStr)
				receiveBody_t.Http_status = http_status
				head_length += h_length
				head_end = true
				log.Debugf("GetProxy_https head_length: %s, http_status: %s, thehead: %s, url_id: %s, url: %s", head_length, http_status, thehead, receiveBody_t.Url_id, receiveBody_t.Url)
			}
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
	log.Debugf("Process_go url: %s, Ip_to: %s, Head_send: %s, url_id: %s, is_http_t: %t", body.Url, Ip_to, Head_send, body.Url_id, is_http_t)
	if is_http_t == true {
		GetProxy_http(Ip_to, Head_send, body)
	} else {
		GetProxy_https(Ip_to, Head_send, body)
	}

	// time.Sleep(3 * 1e9)
	log.Debugf("Process_go finished: %s, url_id: %s, status: %s", body.Url, body.Url_id, body.Status)
	// if ip == nil {
	// 	log.Errorf("%s, IP Invalid!", body.IP)
	// } else {
	// 	ProcessKeyData(body)
	// }
	// <- maxConcurrent_ch
}

/*TaskRequestPost 处理实时接收汇报的请求，并异步判断请求
post: [{}]
return:{"msg":"ok"}
*/
func TaskRequestPost(writer http.ResponseWriter, r *http.Request) {
	var body []ReceiveBody
	var err error
	result, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()
	log.Debugf("TaskRequestPost request_body: %s", result)

	err = json.Unmarshal(result, &body)

	// err = json.Unmarshal([]byte(result), &body)

	// buf := new(bytes.Buffer)
	// // log.Debug(request.Body)
	// buf.ReadFrom(request.Body)
	// err = json.Unmarshal(buf.Bytes(), &body)
	//
	if err != nil {
		log.Error("TaskRequestPost json.Unmarshal[error]", err)
	}
	log.Debugf("TaskRequestPost RemoteAddr: %s, URL.Path: %s, body: %+v", r.RemoteAddr, r.URL.Path, body)
	time.Sleep(1)
	for i := 0; i < len(body); i++ {
		go Process_go(body[i])
	}

	writer.WriteHeader(200)
	var param = make(map[string]interface{})
	param["msg"] = "ok"
	msg, _ := json.Marshal(param)
	_, err = writer.Write([]byte(msg))
	if err != nil {
		http.Error(writer, "Interal ERROR: ", 500)
	}
}

func PostRealBody(data ReceiveBody) {
	b, err := json.Marshal(data)
	if err != nil {
		log.Debugf("PostRealBody json err:%s", err)
	}
	body := bytes.NewBuffer([]byte(b))
	log.Debugf("PostRealBody body: %+v", body)
	Report_address := "http://" + data.Report_ip + ":" + data.Report_port + "/content/preload_report"
	//可以通过client中transport的Dail函数,在自定义Dail函数里面设置建立连接超时时长和发送接受数据超时
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
	for i := 0; i < 4; i++ {
		request, errNewRequest := http.NewRequest("POST", Report_address, body) //提交请求;用指定的方法，网址，可选的主体放回一个新的*Request
		if errNewRequest != nil {
			log.Debugf("PostRealBody retried No.%d times, errNewRequest: %s, url_id: %s, url: %s", i, errNewRequest, data.Url_id, data.Url)
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
			log.Debugf("PostRealBody retried No.%d times, errDo: %s, request: %+v, Report_address: %s, url_id: %s, url: %s", i, errDo, request, Report_address, data.Url_id, data.Url)
			time.Sleep(1e9)
			continue
		}
		status := response.StatusCode //获取返回状态码，正常是200
		if status != 200 {
			log.Debugf("PostRealBody retried No.%d times, status: %d, url_id: %s, url: %s", i, status, data.Url_id, data.Url)
			time.Sleep(1e9)
			continue
		}
		defer response.Body.Close()
		r_body, errReadAll := ioutil.ReadAll(response.Body)
		if errReadAll != nil {
			log.Debugf("PostRealBody retried No.%d times, errReadAll: %s, url_id: %s, url: %s", i, errReadAll, data.Url_id, data.Url)
			return
		}
		log.Debugf("PostRealBody retried No.%d times, status: %d, r_body: %s, url_id: %s, url: %s", i, status, r_body, data.Url_id, data.Url)
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
	// 		log.Debugf("PostRealBody[Post] retried No.%d times, error: %s, url_id: %s, url: %s", i, err, data.Url_id, data.Url)
	// 		time.Sleep(1e9)
	// 		continue
	// 	}
	// 	r_body, err := ioutil.ReadAll(resp.Body)
	// 	if err != nil {
	// 		// handle error
	// 		log.Debugf("PostRealBody[ReadAll] retried No.%d times, error: %s, url_id: %s, url: %s", i, err, data.Url_id, data.Url)
	// 	}
	// 	log.Debugf("PostRealBody resp retried No.%d times, r_body: %s, url_id: %s, url: %s", i, string(r_body), data.Url_id, data.Url)
	// 	defer resp.Body.Close()
	// 	break
	// }

}
