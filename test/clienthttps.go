package main
import(
    "fmt"
    "crypto/tls"
    "time"
)

func main(){
	conf := &tls.Config{
		InsecureSkipVerify: true,
	}
	conn, err := tls.Dial("tcp", "127.0.0.1:443", conf)
	if err != nil {
		fmt.Printf("err1:%s", err)
		return
	}
	defer conn.Close()
	//https://te-cdn.sao.so/top/custom-template/48/saveData.json
	url := "https://127.0.0.1/top/base-template/36/saveData.json"
	host := "te-cdn.sao.so"
	str_t := "GET " + url + " HTTP/1.1\r\n" + "Host:" + host + "\r\n\r\n"
	_, err1 := conn.Write([]byte(str_t))
	if err1 != nil {
		fmt.Printf("err, %s", err1)
		return
	}
	buf := make([]byte, 8192)
	conn.SetDeadline(time.Now().Add(time.Duration(30) * time.Second))
	for{
		length, err := conn.Read(buf)
		if err != nil {
			fmt.Printf("close:%s", err)
			break
		}
		if length > 0{
			fmt.Printf("data length:%s, data:%s", length, buf[:length])
			conn.SetDeadline(time.Now().Add(time.Duration(30) * time.Second))
			// time.Sleep(2 * 1e9)
		}
	}
}
