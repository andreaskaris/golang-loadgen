package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

const (
	NANOSECONDS_PER_SECOND = 1000000000
	TCP                    = "tcp"
	UDP                    = "udp"
)

// client implements the client logic for a single connection. It opens a connection of type proto (TCP or UDP) to
// <host>:<port> and it writes <message> to the connection before closing it.
// Note: The terminology may be a bit confusing as the client pushes data to the server, not the other way around.
func client(proto, host string, port int, message string) error {
	conn, err := net.Dial(proto, fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Fprint(conn, message)
	return nil
}

// server implements the logic for the server. It uses helper functions tcpServer and udpServer to implement servers
// for the respective protocols.
func server(proto, host string, port int) error {
	hostPort := fmt.Sprintf("%s:%d", host, port)
	if proto == UDP {
		return udpServer(proto, hostPort)
	}
	if proto == TCP {
		return tcpServer(proto, hostPort)
	}
	return fmt.Errorf("unsupported protocol: %q", proto)
}

// udpServer implements the logic for a UDP server. It listens on a given UDP socket. It reads from the socket and
// prints the message of the client if flag -debug was provided.
func udpServer(proto, hostPort string) error {
	addr, err := net.ResolveUDPAddr(proto, hostPort)
	if err != nil {
		return err
	}
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return err
	}
	defer conn.Close()
	buffer := make([]byte, 1024)
	for {
		n, remote, err := conn.ReadFromUDP(buffer)
		if err != nil {
			log.Printf("could not read from UDP buffer, err: %q", err)
			continue
		}
		if *debugFlag {
			log.Printf("read from remote %s: %s", remote, string(buffer[:n]))
		}
	}
}

// tcpServer implements the logic for a TCP server. It listens on a given TCP socket. It accepts connections and then
// handles them in another go routine, handleConnection(conn).
func tcpServer(proto, hostPort string) error {
	ln, err := net.Listen(proto, hostPort)
	if err != nil {
		return err
	}
	for {
		conn, err := ln.Accept()
		if err != nil {
			return err
		}
		go handleConnection(conn)
	}
}

// handleConnection handles a single connection for the TCP server. The server reads the client's message and prints it
// if the -debug flag was provided. Otherwise it waits for the connection to be closed, to reach EOF or '\n' before closing
// the connection.
func handleConnection(conn net.Conn) {
	defer conn.Close()

	msg, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil && err != io.EOF {
		log.Printf("error reading from connection, err: %q", err)
	}
	if *debugFlag {
		log.Printf("read from remote %s: %s", conn.RemoteAddr().String(), msg)
	}
}

var (
	serverFlag   = flag.Bool("server", false, "server")
	protocolFlag = flag.String("protocol", "tcp", "protocol")
	hostFlag     = flag.String("host", "127.0.0.1", "host")
	portFlag     = flag.Int("port", 8080, "port")
	rateFlag     = flag.Int("rate-per-second", 1000, "rate of connections per second")
	debugFlag    = flag.Bool("debug", false, "debug")
)

func main() {
	// Parse command line flags.
	flag.Parse()

	var protocol string
	switch *protocolFlag {
	case TCP:
		protocol = TCP
	case UDP:
		protocol = UDP
	default:
		log.Fatal("Invalid protocol")
	}

	// Code for the server. See server() for more details.
	if *serverFlag {
		if err := server(protocol, *hostFlag, *portFlag); err != nil {
			log.Fatalf("could not create server, err: %q", err)
		}
		return
	}

	// Code for the client. The client calculates the sleep time between subsequent attempts based on the rate.
	// For example, if the rate is 1000, sleep for 1,000,000,000 / 1,000 = 1,000,000 nanoseconds between messages
	// -> send 100 messages per second.
	sleepInterval := NANOSECONDS_PER_SECOND / *rateFlag
	sleepTime := time.Nanosecond * time.Duration(sleepInterval)
	for {
		time.Sleep(sleepTime)
		// Run each client in its own go routine. See client() for the rest of the client logic.
		go func() {
			if err := client(protocol, *hostFlag, *portFlag, "msg"); err != nil {
				log.Printf("got error on connection attempt, err: %q", err)
			}
		}()
	}
}
