package main  
  
import (  
    "net"  
    "fmt"  
    "os"  
    "time"  
    "strconv"
)  
  
func main() {  
  
    pTCPAddr, error := net.ResolveTCPAddr("tcp4", ":9999")  
    if error != nil {  
        fmt.Fprintf(os.Stdout, "Error: %s", error.Error())  
        return  
    }  
    pTCPListener, error := net.ListenTCP("tcp4", pTCPAddr)  
    if error != nil {  
        fmt.Fprintf(os.Stdout, "Error: %s", error.Error())  
        return  
    }  
    defer pTCPListener.Close()  
  
    for {  
        pTCPConn, error := pTCPListener.AcceptTCP()  
        if error != nil {  
            fmt.Fprintf(os.Stdout, "Error: %s", error.Error())  
            continue  
        }  
        go connHandler(pTCPConn)  
    }  
}  
  
func connHandler(conn *net.TCPConn) {  
    defer conn.Close()  
    now := time.Now()  
    for i := 0; i < 4; i++{
        time.Sleep(5 * 1e9)
        conn.Write([]byte(now.String() + "sdfsdfasdfkjlsfdjlsjfdlsdjflskjdflrouqwipoeriuwoeirlskdjf,xzmcnxznjasodfosaiufdwoeriuasdf\n")) 
        conn.Write([]byte(strconv.Itoa(i)))
        
    }
    for i := 0; i < 4; i++{
        time.Sleep(10 * 1e9)
        conn.Write([]byte(now.String() + "sdfsdfasdfkjlsfdjlsjfdlsdjflskjdflrouqwipoeriuwoeirlskdjf,xzmcnxznjasodfosaiufdwoeriuasdf\n")) 
        conn.Write([]byte(strconv.Itoa(i)))
        
    }

     
}  
