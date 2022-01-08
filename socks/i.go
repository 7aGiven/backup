package main

import "fmt"
import "net"
import "time"
import "context"
import "crypto/tls"
import "io"
import "strconv"
import "bytes"

type cs struct {
	server *net.TCPConn
	client net.Conn
}

var ch = make(chan net.Conn)

func main() {
	cert, _ := tls.LoadX509KeyPair("/etc/letsencrypt/live/www.giiiiiv.buzz/fullchain.pem", "/etc/letsencrypt/live/www.giiiiiv.buzz/privkey.pem")
	config := &tls.Config{Certificates: []tls.Certificate{cert}}
	listen, err := (&net.ListenConfig{KeepAlive: 3 * time.Minute}).Listen(context.Background(), "tcp4", ":443")
	if err != nil {
		fmt.Println(err)
		return
	}
	listen = tls.NewListener(listen, config)
	for i := byte(0); i <= byte(200); i++ {
		go proxy(i)
	}
	var connection net.Conn
	for {
		connection, err = listen.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		ch <- connection
	}
}
func proxy(index byte) {
	fmt.Printf("启动第%v个goroutine\n", index)
	buf := make([]byte, 1024*1024)
	chcs := make(chan bool)
	var conn net.Conn
	var domain string
	var port string
	var n int
	var length byte
	var addr *net.TCPAddr
	var server *net.TCPConn
	var err error
	go func() {
		for {
			<-chcs
			server.ReadFrom(conn)
		}
	}()
	for {
		conn = <-ch
		func() {
			defer conn.Close()
			n, _ = conn.Read(buf)
			if n == 3 && buf[0] == 5 && buf[1] == 1 && buf[2] == 0 {
				conn.Write([]byte{5, 0})
				n, _ = conn.Read(buf)
				if n >= 7 && buf[0] == 5 && buf[2] == 0 {
					switch buf[1] {
					case byte(1):
						switch buf[3] {
						case byte(3):
							length = buf[4]
							port = strconv.Itoa(256*int(buf[5+length]) + int(buf[6+length]))
							domain = string(buf[5 : 5+length])
						case byte(1):
							port = strconv.Itoa(256*int(buf[8]) + int(buf[9]))
							domain = net.IPv4(buf[4], buf[5], buf[6], buf[7]).String()
						default:
							return
						}
						addr, err = net.ResolveTCPAddr("tcp4", domain+":"+port)
						if err != nil {
							return
						}
						fmt.Println("TCP", addr, domain+":"+port)
						server, err = net.DialTCP("tcp4", nil, addr)
						if err != nil {
							return
						}
						defer server.Close()
						conn.Write([]byte{5, 0, 0, 1, 0, 0, 0, 0, 0, 0})
						chcs <- true
						io.CopyBuffer(conn, server, buf)
					case byte(3):
						return
					}
				}
			} else {
				addr, _ = net.ResolveTCPAddr("tcp4", "127.0.0.1:80")
				server, err = net.DialTCP("tcp4", nil, addr)
				if err != nil {
					return
				}
				defer server.Close()
				server.Write(buf[:n])
				chcs <- true
				io.Copy(conn, server)
			}
		}()
	}
}
