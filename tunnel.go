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
	AutoStart  bool   `json:"autoStart"`
	// Jump host / bastion (all optional)
	BastionHost     string `json:"bastionHost,omitempty"`
	BastionPort     int    `json:"bastionPort,omitempty"`
	BastionUser     string `json:"bastionUser,omitempty"`
	BastionAuthType string `json:"bastionAuthType,omitempty"`
	BastionPassword string `json:"bastionPassword,omitempty"`
	BastionKeyPath  string `json:"bastionKeyPath,omitempty"`
}

type TunnelStatus struct {
	ID           string `json:"id"`
	Active       bool   `json:"active"`
	Error        string `json:"error"`
	Uptime       string `json:"uptime"`
	Reconnecting bool   `json:"reconnecting"`
}

type activeTunnel struct {
	config       TunnelConfig
	clientMu     sync.RWMutex
	client       *ssh.Client
	bastionClient *ssh.Client
	listener     net.Listener
	startTime    time.Time
	done         chan struct{}
	reconnecting bool
	lastError    string
}

type TunnelManager struct {
	mu         sync.RWMutex
	active     map[string]*activeTunnel
	settingsMu sync.RWMutex
	settings   Settings
}

func NewTunnelManager(settings Settings) *TunnelManager {
	return &TunnelManager{
		active:   make(map[string]*activeTunnel),
		settings: settings,
	}
}

func (tm *TunnelManager) UpdateSettings(s Settings) {
	tm.settingsMu.Lock()
	tm.settings = s
	tm.settingsMu.Unlock()
}

func (tm *TunnelManager) getSettings() Settings {
	tm.settingsMu.RLock()
	defer tm.settingsMu.RUnlock()
	return tm.settings
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

// dialSSH establishes an SSH connection (optionally via bastion) and returns
// the target client and an optional bastion client.
func dialSSH(cfg TunnelConfig) (*ssh.Client, *ssh.Client, error) {
	authMethods, err := buildAuthMethods(cfg.AuthType, cfg.Password, cfg.KeyPath)
	if err != nil {
		return nil, nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", cfg.SSHHost, cfg.SSHPort)

	if cfg.BastionHost != "" {
		bastionPort := cfg.BastionPort
		if bastionPort == 0 {
			bastionPort = 22
		}
		bastionAuth, berr := buildAuthMethods(cfg.BastionAuthType, cfg.BastionPassword, cfg.BastionKeyPath)
		if berr != nil {
			return nil, nil, berr
		}
		bastionCfg := &ssh.ClientConfig{
			User:            cfg.BastionUser,
			Auth:            bastionAuth,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         10 * time.Second,
		}
		bastionAddr := fmt.Sprintf("%s:%d", cfg.BastionHost, bastionPort)
		bastionClient, berr := ssh.Dial("tcp", bastionAddr, bastionCfg)
		if berr != nil {
			return nil, nil, fmt.Errorf("bastion dial failed: %w", berr)
		}
		conn, berr := bastionClient.Dial("tcp", addr)
		if berr != nil {
			bastionClient.Close()
			return nil, nil, fmt.Errorf("bastion to SSH host failed: %w", berr)
		}
		ncc, chans, reqs, berr := ssh.NewClientConn(conn, addr, sshConfig)
		if berr != nil {
			conn.Close()
			bastionClient.Close()
			return nil, nil, fmt.Errorf("SSH handshake via bastion failed: %w", berr)
		}
		return ssh.NewClient(ncc, chans, reqs), bastionClient, nil
	}

	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, nil, fmt.Errorf("SSH dial failed: %w", err)
	}
	return client, nil, nil
}

func (tm *TunnelManager) Start(cfg TunnelConfig) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if _, exists := tm.active[cfg.ID]; exists {
		return fmt.Errorf("tunnel %s is already running", cfg.ID)
	}

	client, bastionClient, err := dialSSH(cfg)
	if err != nil {
		return err
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
	go at.monitor(tm, tm.getSettings().KeepaliveSeconds)
	return nil
}

func (at *activeTunnel) getClient() *ssh.Client {
	at.clientMu.RLock()
	defer at.clientMu.RUnlock()
	return at.client
}

// monitor sends SSH keepalives every 15 seconds and reconnects on failure.
// It is the sole owner of the SSH client lifecycle after Start() returns.
func (at *activeTunnel) monitor(tm *TunnelManager, keepaliveSeconds int) {
	defer func() {
		at.clientMu.Lock()
		if at.client != nil {
			at.client.Close()
		}
		if at.bastionClient != nil {
			at.bastionClient.Close()
		}
		at.clientMu.Unlock()
	}()

	ticker := time.NewTicker(time.Duration(keepaliveSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-at.done:
			return
		case <-ticker.C:
			at.clientMu.RLock()
			client := at.client
			at.clientMu.RUnlock()

			_, _, err := client.SendRequest("keepalive@openssh.com", true, nil)
			if err != nil && tm.getSettings().AutoReconnect {
				at.reconnect()
			}
		}
	}
}

// reconnect closes the dead SSH connection and redials with exponential backoff.
func (at *activeTunnel) reconnect() {
	at.clientMu.Lock()
	at.reconnecting = true
	if at.client != nil {
		at.client.Close()
		at.client = nil
	}
	if at.bastionClient != nil {
		at.bastionClient.Close()
		at.bastionClient = nil
	}
	at.clientMu.Unlock()

	backoff := 2 * time.Second
	for {
		select {
		case <-at.done:
			return
		case <-time.After(backoff):
		}

		client, bastionClient, err := dialSSH(at.config)
		if err != nil {
			at.clientMu.Lock()
			at.lastError = err.Error()
			at.clientMu.Unlock()
			if backoff < 30*time.Second {
				backoff *= 2
			}
			continue
		}

		at.clientMu.Lock()
		at.client = client
		at.bastionClient = bastionClient
		at.reconnecting = false
		at.lastError = ""
		at.clientMu.Unlock()
		return
	}
}

func (at *activeTunnel) accept() {
	defer at.listener.Close()
	for {
		conn, err := at.listener.Accept()
		if err != nil {
			return
		}
		go at.forward(conn)
	}
}

func (at *activeTunnel) forward(local net.Conn) {
	defer local.Close()
	client := at.getClient()
	if client == nil {
		return
	}
	remoteAddr := fmt.Sprintf("%s:%d", at.config.RemoteHost, at.config.RemotePort)
	remote, err := client.Dial("tcp", remoteAddr)
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
	at.clientMu.RLock()
	reconnecting := at.reconnecting
	lastError := at.lastError
	at.clientMu.RUnlock()
	return TunnelStatus{
		ID:           id,
		Active:       true,
		Uptime:       time.Since(at.startTime).Round(time.Second).String(),
		Reconnecting: reconnecting,
		Error:        lastError,
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
		delete(tm.active, id)
	}
}
