package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

type TunnelConfig struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	SSHHost    string `json:"sshHost"`
	SSHPort    int    `json:"sshPort"`
	User       string `json:"user"`
	AuthType   string `json:"authType"`
	Password   string `json:"password,omitempty"`
	KeyPath    string `json:"keyPath,omitempty"`
	LocalPort  int    `json:"localPort"`
	RemoteHost string `json:"remoteHost"`
	RemotePort int    `json:"remotePort"`
}

type TunnelStatus struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
	Error  string `json:"error"`
	Uptime string `json:"uptime"`
}

type activeTunnel struct {
	config    TunnelConfig
	client    *ssh.Client
	listener  net.Listener
	startTime time.Time
	done      chan struct{}
}

type TunnelManager struct {
	mu     sync.RWMutex
	active map[string]*activeTunnel
}

func NewTunnelManager() *TunnelManager {
	return &TunnelManager{active: make(map[string]*activeTunnel)}
}

func (tm *TunnelManager) Start(cfg TunnelConfig) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.active[cfg.ID]; exists {
		return fmt.Errorf("tunnel %s is already running", cfg.ID)
	}

	var authMethods []ssh.AuthMethod
	if cfg.AuthType == "key" {
		keyBytes, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return fmt.Errorf("failed to read key file: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", cfg.SSHHost, cfg.SSHPort)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH dial failed: %w", err)
	}

	localAddr := fmt.Sprintf("127.0.0.1:%d", cfg.LocalPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		client.Close()
		return fmt.Errorf("local listen failed on port %d: %w", cfg.LocalPort, err)
	}

	at := &activeTunnel{
		config:    cfg,
		client:    client,
		listener:  listener,
		startTime: time.Now(),
		done:      make(chan struct{}),
	}
	tm.active[cfg.ID] = at
	go at.accept()
	return nil
}

func (at *activeTunnel) accept() {
	defer func() {
		at.listener.Close()
		at.client.Close()
	}()
	for {
		conn, err := at.listener.Accept()
		if err != nil {
			select {
			case <-at.done:
			default:
			}
			return
		}
		go at.forward(conn)
	}
}

func (at *activeTunnel) forward(local net.Conn) {
	defer local.Close()
	remoteAddr := fmt.Sprintf("%s:%d", at.config.RemoteHost, at.config.RemotePort)
	remote, err := at.client.Dial("tcp", remoteAddr)
	if err != nil {
		return
	}
	defer remote.Close()
	done := make(chan struct{}, 2)
	go func() { io.Copy(local, remote); done <- struct{}{} }()
	go func() { io.Copy(remote, local); done <- struct{}{} }()
	<-done
}

func (tm *TunnelManager) Stop(id string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	at, exists := tm.active[id]
	if !exists {
		return nil
	}
	close(at.done)
	at.listener.Close()
	at.client.Close()
	delete(tm.active, id)
	return nil
}

func (tm *TunnelManager) IsActive(id string) bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	_, exists := tm.active[id]
	return exists
}

func (tm *TunnelManager) GetStatus(id string) TunnelStatus {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	at, exists := tm.active[id]
	if !exists {
		return TunnelStatus{ID: id, Active: false}
	}
	return TunnelStatus{
		ID:     id,
		Active: true,
		Uptime: time.Since(at.startTime).Round(time.Second).String(),
	}
}

func (tm *TunnelManager) GetAllStatuses(ids []string) []TunnelStatus {
	statuses := make([]TunnelStatus, len(ids))
	for i, id := range ids {
		statuses[i] = tm.GetStatus(id)
	}
	return statuses
}

func (tm *TunnelManager) StopAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	for id, at := range tm.active {
		close(at.done)
		at.listener.Close()
		at.client.Close()
		delete(tm.active, id)
	}
}
