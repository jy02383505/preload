package main
import(
      "fmt"
      // "bytes"
      "net"
      "time"
)


func main(){


	GetProxy("127.0.0.1:9999", "/")
}



func GetProxy(ip_to string, url_t string){


	  tcpAddr, err := net.ResolveTCPAddr("tcp4", ip_to)
	  if err != nil {
	  	fmt.Printf("get Proxy err:%s", err.Error())
	  }

      tcpconn, err2 := net.DialTCP("tcp", nil, tcpAddr);
      if err2 != nil{
      	fmt.Printf("err2:%s", err2.Error())
      	// receiveBody_t.Status = "failed"
        return
      }
        // host, _ := GetHost(receiveBody_t.Url)
	    // if len_str >= 3 {
	    //     req.Header.Set("Host", host)
	    //     }else{
	    //         log.Debugf("host,%s" ,host)
	    //     }
         // host, _ := ""
         str_t := "GET " + url_t + " HTTP/1.1" +  "\r\n\r\n"
         fmt.Printf("str_t:%s\n", str_t)
      	 //向tcpconn中写入数据
        _, err3 := tcpconn.Write([]byte(str_t));
        if err3 != nil{
        	fmt.Printf("err3:%s\n", err3)
	      	// receiveBody_t.Status = "failed"
	       //  go PostRealBody(receiveBody_t)
	       //  <- maxConcurrent_ch
	        return
        }
        buf := make([]byte, 64)
        tcpconn.SetDeadline(time.Now().Add(time.Duration(6) * time.Second))
        for{

        	length, err := tcpconn.Read(buf)
        	if err != nil{
        		tcpconn.Close()
        		fmt.Printf("tcp conn err:%s\n", err)
		        break
        	}
        	Data := buf[:length]
        	// messnager := make(chan byte)
        	if length > 0{  
            // buf[lenght]=0 
               fmt.Printf("length:%s\n", length) 
               fmt.Printf("content:%s\n", Data)
               tcpconn.SetDeadline(time.Now().Add(time.Duration(6) * time.Second))
            }
        //心跳计时  
        // go HeartBeating(tcpconn,messnager, 6)  
        //检测每次Client是否有数据传来  
        // go GravelChannel(Data,messnager)  
        //fmt.Println("Rec[",conn.RemoteAddr().String(),"] Say :" ,string(buf[0:lenght]))  
        // reciveStr :=string(buf[0:lenght]) 
        // if reciveStr != nil {}
        }
        fmt.Printf("post end, url_t:%s\n", url_t)
}


//心跳计时，根据GravelChannel判断Client是否在设定时间内发来信息  
// func HeartBeating(conn net.Conn, readerChannel chan byte,timeout int) {  
//         select {  
//         case fk := <-readerChannel:  
//             fmt.Printf(conn.RemoteAddr().String(), "receive data string:", string(fk))  
//             conn.SetDeadline(time.Now().Add(time.Duration(timeout) * time.Second))  
//             //conn.SetReadDeadline(time.Now().Add(time.Duration(5) * time.Second))  
//             break  
//         case <-time.After(time.Second*2):  
//             fmt.Printf("It's really weird to get Nothing!!!\n")  
//             conn.Close()  
//         }  
  
// } 

// func GravelChannel(n []byte,mess chan byte){  
//     for i , v := range n{ 
//         fmt.Printf("%s byte  v:%s\n", i, v) 
//         mess <- v  
//     }  
//     close(mess)  
// }  