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
	TCP = "tcp"
	UDP = "udp"
)

func client(proto, host string, port int, message string) error {
	conn, err := net.Dial(proto, fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	defer conn.Close()
	fmt.Fprint(conn, message)
	return nil
}

func server(proto, host string, port int) error {
	hostPort := fmt.Sprintf("%s:%d", host, port)
	if proto == TCP {
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
	if proto == UDP {
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
	return fmt.Errorf("unsupported protocol: %q", proto)
}

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

	if *serverFlag {
		if err := server(protocol, *hostFlag, *portFlag); err != nil {
			log.Fatalf("could not create server, err: %q", err)
		}
		return
	}

	sleepInterval := 1000000000 / *rateFlag
	sleepTime := time.Nanosecond * time.Duration(sleepInterval)
	for {
		time.Sleep(sleepTime)
		go func() {
			if err := client(protocol, *hostFlag, *portFlag, "msg"); err != nil {
				log.Printf("got error on connection attempt, err: %q", err)
			}
		}()
	}
}
