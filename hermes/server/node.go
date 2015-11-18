package server

import (
	"bufio"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

type Node struct {
	id  string
	hub *msghub
}

func NewNode() *Node {
	return &Node{
		id:  randStr(4),
		hub: newMsgHub(),
	}
}

func (n *Node) FollowAddr(addr string) error {
	c, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("cannot connect: %s", err)
	}
	//defer c.Close()dd TODO
	return n.Follow(c)
}

func (n *Node) Follow(rw io.ReadWriter) error {
	rw.Write([]byte("replica\n"))
	go n.replicate(rw)
	return nil
}

func (n *Node) replicate(rw io.ReadWriter) error {

	go func() {
		read := json.NewDecoder(rw).Decode
		for {
			var m Message
			if err := read(&m); err != nil {
				if err == io.EOF {
					return
				}
				log.Printf("cannot decode message: %s", err)
				return
			}
			println("received replica message", m.MessageID)
			n.hub.Publish(&m)
		}
	}()

	repc := make(chan *Message, 8)
	n.hub.Subscribe(repc)
	defer n.hub.Unsubscribe(repc)

	write := json.NewEncoder(rw).Encode
	for m := range repc {
		if err := write(m); err != nil {
			log.Printf("cannot write to replica: %s", err)
			return err
		}
	}
	return nil
}

func (n *Node) Serve(ln net.Listener) error {
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go n.handleClient(c)
	}
}

func (n *Node) handleClient(c net.Conn) {
	defer c.Close()
	rd := bufio.NewReader(c)
	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			return
		}
		switch line := strings.TrimSpace(line); line {
		case "replica":
			log.Print("replica client connected")
			n.replicate(c)
		case "v1":
			log.Print("v1 client connected")
			n.handleClientV1(rd, c)
		default:
			fmt.Fprintln(c, "unknown client type")
			return
		}
	}
}

func (n *Node) handleClientV1(rd *bufio.Reader, w io.Writer) {
	msgc := make(chan *Message, 4)
	n.hub.Subscribe(msgc)
	defer n.hub.Unsubscribe(msgc)

	go func() {
		for m := range msgc {
			println("writing to client", m.MessageID)
			if _, err := fmt.Fprintln(w, m.Content); err != nil {
				log.Printf("cannot write to client %v: %s", msgc, err)
				return
			}
		}
	}()

	id := struct {
		cnt    int
		prefix string
	}{
		cnt:    0,
		prefix: randStr(8),
	}

	for {
		line, err := rd.ReadString('\n')
		if err != nil {
			log.Printf("cannot read from client %v: %s", msgc, err)
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		id.cnt++
		n.hub.Publish(&Message{
			MessageID: fmt.Sprintf("%s:%d", id.prefix, id.cnt),
			Created:   time.Now(),
			Content:   line,
		})
	}
}

func randStr(size int) string {
	b := make([]byte, size)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base32.StdEncoding.EncodeToString(b)[:size]
}
