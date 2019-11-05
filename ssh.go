package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

var msg reply

func listenSSH() {
	for {
		if err := openTunnel(); err != nil {
			sshRemotePort = 0
			sshConnectTime = time.Time{}
			log.Printf("ssh connection failed: %v, restarting...", err)
			time.Sleep(time.Duration(config.SSHRetryGapSec) * time.Second)
		}
	}
}

func openTunnel() error {
	if len(msg.SSHServerHost) == 0 {
		return errors.New("ssh information is empty")
	}
	sshConfig := &ssh.ClientConfig{
		User:            msg.SSHServerUser,
		Auth:            make([]ssh.AuthMethod, 0),
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	if len(msg.SSHKey) > 0 {
		key, err := ssh.ParsePrivateKey([]byte(msg.SSHKey))
		if err != nil {
			return fmt.Errorf("failed to parase key: %w", err)
		}
		sshConfig.Auth = append(sshConfig.Auth, ssh.PublicKeys(key))
	}
	if len(msg.SSHPassword) > 0 {
		sshConfig.Auth = append(sshConfig.Auth, ssh.Password(msg.SSHPassword))
	}

	// Connect to the server
	serverConn, err := ssh.Dial("tcp", msg.SSHServer(), sshConfig)
	if err != nil {
		return fmt.Errorf("failed to connect remote ssh server %s: %w", msg.SSHServer(), err)
	}

	// Open a remote socket
	listener, err := serverConn.Listen("tcp", fmt.Sprintf("%s:%d", "localhost", 0))
	if err != nil {
		return fmt.Errorf("failed to open remote socket: %w", err)
	}
	defer safeClose(listener, "remote socket listener")
	sshRemotePort = port(listener.Addr())
	sshConnectTime = time.Now().UTC()
	log.Printf("ssh listener open: %s", listener.Addr().String())
	go doReport(-1)

	// Open a local socket
	for {
		local, err := net.Dial("tcp", config.SSHLocal())
		if err != nil {
			return fmt.Errorf("failed to connect local socket: %s", err)
		}
		client, err := listener.Accept()
		if err != nil {
			return fmt.Errorf("failed to listen local socket: %w", err)
		}
		handleClient(client, local)
	}
}

// handleClient handles local socket from the tunnel.
func handleClient(client net.Conn, remote net.Conn) {
	defer safeClose(client, "client")
	chDone := make(chan bool)

	// Start remote -> local data transfer
	go func() {
		_, err := io.Copy(client, remote)
		if err != nil {
			log.Printf("error while copy remote->local: %s", err)
		}
		chDone <- true
	}()

	// Start local -> remote data transfer
	go func() {
		_, err := io.Copy(remote, client)
		if err != nil {
			log.Printf("error while copy local->remote: %s", err)
		}
		chDone <- true
	}()
	<-chDone
}

func port(addr net.Addr) int {
	i := strings.LastIndex(addr.String(), ":")
	if i < 0 {
		return 0
	}
	s := addr.String()[i+1:]
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return n
}
