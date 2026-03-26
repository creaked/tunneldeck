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
	// Jump host / bastion (all optional)
	BastionHost     string `json:"bastionHost,omitempty"`
	BastionPort     int    `json:"bastionPort,omitempty"`
	BastionUser     string `json:"bastionUser,omitempty"`
	BastionAuthType string `json:"bastionAuthType,omitempty"`
	BastionPassword string `json:"bastionPassword,omitempty"`
	BastionKeyPath  string `json:"bastionKeyPath,omitempty"`
}

type TunnelStatus struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
	Error  string `json:"error"`
	Uptime string `json:"uptime"`
}

type activeTunnel struct {
	config        TunnelConfig
	client        *ssh.Client
	bastionClient *ssh.Client
	listener      net.Listener
	startTime     time.Time
	done          chan struct{}
}

type TunnelManager struct {
	mu     sync.RWMutex
	active map[string]*activeTunnel
}

func NewTunnelManager() *TunnelManager {
	return &TunnelManager{active: make(map[string]*activeTunnel)}
}

func buildAuthMethods(authType, password, keyPath string) ([]ssh.AuthMethod, error) {
	if authType == "key" {
		keyBytes, err := os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read key file: %w", err)
		}
		signer, err := ssh.ParsePrivateKey(keyBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
	}
	return []ssh.AuthMethod{ssh.Password(password)}, nil
}

func (tm *TunnelManager) Start(cfg TunnelConfig) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.active[cfg.ID]; exists {
		return fmt.Errorf("tunnel %s is already running", cfg.ID)
	}

	authMethods, err := buildAuthMethods(cfg.AuthType, cfg.Password, cfg.KeyPath)
	if err != nil {
		return err
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", cfg.SSHHost, cfg.SSHPort)

	var client *ssh.Client
	var bastionClient *ssh.Client

	if cfg.BastionHost != "" {
		bastionPort := cfg.BastionPort
		if bastionPort == 0 {
			bastionPort = 22
		}
		bastionAuth, berr := buildAuthMethods(cfg.BastionAuthType, cfg.BastionPassword, cfg.BastionKeyPath)
		if berr != nil {
			return berr
		}
		bastionCfg := &ssh.ClientConfig{
			User:            cfg.BastionUser,
			Auth:            bastionAuth,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		}
		bastionAddr := fmt.Sprintf("%s:%d", cfg.BastionHost, bastionPort)
		bastionClient, berr = ssh.Dial("tcp", bastionAddr, bastionCfg)
		if berr != nil {
			return fmt.Errorf("bastion dial failed: %w", berr)
		}
		conn, berr := bastionClient.Dial("tcp", addr)
		if berr != nil {
			bastionClient.Close()
			return fmt.Errorf("bastion to SSH host failed: %w", berr)
		}
		ncc, chans, reqs, berr := ssh.NewClientConn(conn, addr, sshConfig)
		if berr != nil {
			conn.Close()
			bastionClient.Close()
			return fmt.Errorf("SSH handshake via bastion failed: %w", berr)
		}
		client = ssh.NewClient(ncc, chans, reqs)
	} else {
		client, err = ssh.Dial("tcp", addr, sshConfig)
		if err != nil {
			return fmt.Errorf("SSH dial failed: %w", err)
		}
	}

	localAddr := fmt.Sprintf("127.0.0.1:%d", cfg.LocalPort)
	listener, err := net.Listen("tcp", localAddr)
	if err != nil {
		client.Close()
		if bastionClient != nil {
			bastionClient.Close()
		}
		return fmt.Errorf("local listen failed on port %d: %w", cfg.LocalPort, err)
	}

	at := &activeTunnel{
		config:        cfg,
		client:        client,
		bastionClient: bastionClient,
		listener:      listener,
		startTime:     time.Now(),
		done:          make(chan struct{}),
	}
	tm.active[cfg.ID] = at
	go at.accept()
	return nil
}

func (at *activeTunnel) accept() {
	defer func() {
		at.listener.Close()
		at.client.Close()
		if at.bastionClient != nil {
			at.bastionClient.Close()
		}
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
	if at.bastionClient != nil {
		at.bastionClient.Close()
	}
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
		if at.bastionClient != nil {
			at.bastionClient.Close()
		}
		delete(tm.active, id)
	}
}
