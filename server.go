package mist

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"strings"
)

// start
func (m *Mist) start() {
	m.log.Info("[MIST] Starting server...\n")

	//
	go func() {

		//
		l, err := net.Listen("tcp", ":"+m.port)
		if err != nil {
			m.log.Error("%+v\n", err)
		}

		defer l.Close()

		m.log.Info("[MIST] Listening on port %+v\n", m.port)

		// Listen for an incoming connection.
		for {
			conn, err := l.Accept()
			if err != nil {
				m.log.Error("%+v\n", err)
			}

			// Handle connections in a new goroutine.
			go m.handleConnection(conn)
		}
	}()
}

// handleConnection
func (m *Mist) handleConnection(conn net.Conn) {

	m.log.Debug("[MIST] New connection detected: %+v\n", conn)

	var cmd string
	var tags string

	//
	r := bufio.NewReader(conn)

	//
	sub := Subscription{
		Sub: make(chan Message),
	}

	//
	done := make(chan bool)

	// create our 'publish handler'
	go func() {
		for {
			select {

			//
			case msg := <-sub.Sub:

				b, err := json.Marshal(msg)
				if err != nil {
					m.log.Error("[MIST] Failed to marshal message: %v\n", err)
				}

				//
				bsize := make([]byte, 4)
    		binary.LittleEndian.PutUint32(bsize, uint32(len(b)))

				if _, err := conn.Write(append(bsize, b...)); err != nil {
					break
				}

			//
			case <-done:
				break

			}
		}
	}()

	//
	for {
		l, err := r.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				conn.Close()
				m.Unsubscribe(sub)
				done <- true
				// close(sub.Sub)
				break
			} else {
				m.log.Error("[MIST] Error reading stream: %+v\n", err.Error())
			}
		}

		split := strings.Split(strings.TrimSpace(l), " ")
		cmd = split[0]

		if len(split) > 1 {
			tags = split[1]
		}

		// create a subscription for each tag
		sub.Tags = strings.Split(tags, ",")

		//
		switch cmd {
		case "subscribe":
			m.Subscribe(sub)
		case "unsubscribe":
			m.Unsubscribe(sub)
		case "subscriptions":
			m.List()
		default:
			m.log.Error("[MIST] Unknown command: %+v\n", cmd)
		}
	}

	return
}
