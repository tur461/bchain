package network

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

type Peer struct {
	Run string
}

const (
	writeWait = 10 * time.Second

	pongWait = 60 * time.Second

	PingPeriod = (pongWait * 9) / 10

	maxMessageSize = 512

	minPortRange = 8000

	maxPortRange = 8010
)

type Client struct {
	hub *Hub

	conn net.Conn

	send chan []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

type IPeer interface {
	setup(string)
}

func (p Peer) Setup(listen, serve string) {
	var ch chan int
	go setupWsServer(listen, serve)
	// if a == "server" {
	// 	go setupServer()
	// } else {
	// 	go setupClient()
	// }
	<-ch
	ch <- 1
}

func setupClient() {
	conn, err := net.Dial("tcp", "localhost:8080")
	if err != nil {
		fmt.Println(err)
		return
	}
	_, err = conn.Write([]byte("Hello World"))
	if err != nil {
		fmt.Println(err)
		return
	}

	conn.Close()

}

func setupWsServer(listenPort string, acceptPort string) {
	l, _ := strconv.Atoi(listenPort)
	a, _ := strconv.Atoi(acceptPort)
	if l > maxPortRange ||
		l < minPortRange ||
		a > maxPortRange ||
		a < minPortRange {
		log.Fatal(errors.New("port range excedded"))
	}
	// fmt.Println("listening at", listenPort)

	go wsEndpoint(listenPort, acceptPort)
	// http.HandleFunc("/ws", wsEndpoint)
	// log.Fatal(listenWs(listenPort))
}

func acceptWs(port string) {
	// http.(":"+port, nil)
}

func listenWs(port string) error {
	fmt.Println("Server running on port: " + port + " ..")
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		fmt.Println(err)
		return err
	}
	hub := newHub()
	go hub.run()
	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
		// go handleConnection(conn)
		go client.readPump()
	}
}

// func wsEndpoint(w http.ResponseWriter, r *http.Request) {
func wsEndpoint(listen, p string) {
	var client *Client
	// upgrader.CheckOrigin = func(r *http.Request) bool { return true }

	// ws, err := upgrader.Upgrade(w, r, nil)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// log.Println("Connection established")
	hub := newHub()
	go hub.run()
	port, _ := strconv.Atoi(p)
	l, _ := strconv.Atoi(listen)
	counter := 0
	for {
		counter++
		if port == l {
			continue
		}
		if counter == 10 {
			break
		}
		sPort := strconv.Itoa(port)
		fmt.Println(sPort)
		conn, err := net.Dial("tcp", "localhost:"+sPort)
		if err != nil {
			port++
			if port > maxPortRange {
				port = minPortRange
			}
			continue
		}
		fmt.Println("got the connection :D" + sPort)
		client = &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
		go client.writePump()
		// go client.readPump()
	}
	go listenWs(listen)
	//
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	// c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// c.conn.SetPongHandler(func(string) error {
	// 	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	// 	return nil
	// })
	if err := func() error {
		err := c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return err
	}(); err != nil {
		fmt.Println(err)
		return
	}
	result := make([]byte, maxMessageSize)
	for {
		_, err := c.conn.Read(result)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Println("error %v", err)
			}
			c.conn.Close()
			fmt.Println("connected peer lost")
			return
		}

		result = bytes.TrimSpace(bytes.Replace(result, []byte{'\n'}, []byte{' '}, -1))
		c.hub.broadcast <- result
		fmt.Println(string(result))
	}
}

func (c *Client) writePump() {
	stdReader := bufio.NewReader(os.Stdin)
	// fmt.Println("got the connection :D")
	ticker := time.NewTicker(PingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		fmt.Print("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading from stdin")
			panic(err)
		}
		c.send <- []byte(sendData)
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.Write([]byte("not able to sent message"))
				return
			}
			fmt.Println("sending " + string(message) + " ..")
			c.conn.Write(message)

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if _, err := c.conn.Write([]byte("ping controll message")); err != nil {
				return
			}
		default:
			defer func() error {
				return c.conn.Close()
			}()
		}
	}
}

func setupServer() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, maxMessageSize)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Recieved %s", buf)

}
