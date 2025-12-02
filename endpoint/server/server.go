package server

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// 嵌入静态文件
//
//go:embed static/*
var staticFiles embed.FS

// APIResponse 定义统一的API响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// ServiceManager manages agents and proxies
type ServiceManager struct {
	mu      sync.Mutex
	agents  map[string]*ServiceInstance
	proxies map[string]*ServiceInstance
}

// ServiceInstance represents a running agent or proxy
type ServiceInstance struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"` // "agent" or "proxy"
	Args    []string           `json:"args"`
	Ctx     context.Context    `json:"-"` // Ignore in JSON
	Cancel  context.CancelFunc `json:"-"` // Ignore in JSON
	Status  string             `json:"status"`
	ErrChan chan error         `json:"-"` // Ignore in JSON
}

// NewServiceManager creates a new service manager
func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		agents:  make(map[string]*ServiceInstance),
		proxies: make(map[string]*ServiceInstance),
	}
}

var serviceManager = NewServiceManager()

// StartServer starts the tnet management server
func StartServer(listenAddress string) error {
	// 设置路由
	http.HandleFunc("/", serveStaticFile)
	http.HandleFunc("/api/agents", handleAgents)
	http.HandleFunc("/api/proxies", handleProxies)
	http.HandleFunc("/api/agents/start", handleStartAgent)
	http.HandleFunc("/api/agents/stop", handleStopAgent)
	http.HandleFunc("/api/agents/restart", handleRestartAgent)
	http.HandleFunc("/api/proxies/start", handleStartProxy)
	http.HandleFunc("/api/proxies/stop", handleStopProxy)
	http.HandleFunc("/api/proxies/restart", handleRestartProxy)
	http.HandleFunc("/api/agents/delete", handleDeleteAgent)
	http.HandleFunc("/api/proxies/delete", handleDeleteProxy)

	// 启动服务器
	log.Printf("Starting tnet management server on %s", listenAddress)
	return http.ListenAndServe(listenAddress, nil)
}

// serveStaticFile 提供静态文件服务
func serveStaticFile(w http.ResponseWriter, r *http.Request) {
	// 如果路径是根路径，重定向到index.html
	path := r.URL.Path
	if path == "/" {
		path = "/static/index.html"
	} else if !strings.HasPrefix(path, "/static/") {
		// 对于非/static路径，尝试在static目录下查找
		path = "/static" + path
	}

	// 从嵌入的文件系统中读取文件
	content, err := staticFiles.ReadFile(strings.TrimPrefix(path, "/"))
	if err != nil {
		// 文件不存在，返回404
		http.NotFound(w, r)
		return
	}

	// 设置适当的Content-Type
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	}

	// 写入响应
	w.Write(content)
}

// HTTP handlers
func handleAgents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get all agents
	agents := serviceManager.GetAgents()

	// Return response, ensure we return an array
	json.NewEncoder(w).Encode(agents)
}

func handleProxies(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Get all proxies
	proxies := serviceManager.GetProxies()

	// Return response, ensure we return an array
	json.NewEncoder(w).Encode(proxies)
}

func handleStartAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var req struct {
		Args []string `json:"args"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Start agent with auto-generated ID
	instance, err := serviceManager.StartAgent(req.Args)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to start agent: %v", err),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    instance,
	})
}

func handleStartProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var req struct {
		Args []string `json:"args"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Invalid request: %v", err),
		})
		return
	}

	// Start proxy with auto-generated ID
	instance, err := serviceManager.StartProxy(req.Args)
	if err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to start proxy: %v", err),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    instance,
	})
}

func handleStopAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get agent ID from query parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   "Missing agent ID",
		})
		return
	}

	// Stop agent
	if err := serviceManager.StopAgent(id); err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to stop agent: %v", err),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func handleStopProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get proxy ID from query parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   "Missing proxy ID",
		})
		return
	}

	// Stop proxy
	if err := serviceManager.StopProxy(id); err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to stop proxy: %v", err),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func handleRestartAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get agent ID from query parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   "Missing agent ID",
		})
		return
	}

	// Restart agent
	if err := serviceManager.RestartAgent(id); err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to restart agent: %v", err),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func handleRestartProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get proxy ID from query parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   "Missing proxy ID",
		})
		return
	}

	// Restart proxy
	if err := serviceManager.RestartProxy(id); err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to restart proxy: %v", err),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func handleDeleteAgent(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get agent ID from query parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   "Missing agent ID",
		})
		return
	}

	// Delete agent
	if err := serviceManager.DeleteAgent(id); err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to delete agent: %v", err),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

func handleDeleteProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	// Get proxy ID from query parameter
	id := r.URL.Query().Get("id")
	if id == "" {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   "Missing proxy ID",
		})
		return
	}

	// Delete proxy
	if err := serviceManager.DeleteProxy(id); err != nil {
		json.NewEncoder(w).Encode(APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Failed to delete proxy: %v", err),
		})
		return
	}

	// Return response
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
	})
}

// StartAgent starts a new agent instance
func (sm *ServiceManager) StartAgent(args []string) (*ServiceInstance, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate unique ID
	id := uuid.New().String()[:8] // Use first 8 chars of UUID for brevity

	// Create context for the agent
	ctx, cancel := context.WithCancel(context.Background())

	// Create service instance
	instance := &ServiceInstance{
		ID:      id,
		Type:    "agent",
		Args:    args,
		Ctx:     ctx,
		Cancel:  cancel,
		Status:  "starting",
		ErrChan: make(chan error, 1),
	}

	// Add to manager
	sm.agents[id] = instance

	// Start agent in a goroutine
	go sm.runInstance(instance)

	// Update status to running
	instance.Status = "running"
	return instance, nil
}

// RestartAgent restarts an existing agent instance
func (sm *ServiceManager) RestartAgent(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if agent exists
	instance, exists := sm.agents[id]
	if !exists {
		return fmt.Errorf("agent with ID %s not found", id)
	}

	// Stop the existing instance
	instance.Cancel()

	// Create new context for the agent
	ctx, cancel := context.WithCancel(context.Background())

	// Update instance with new context and status
	instance.Ctx = ctx
	instance.Cancel = cancel
	instance.Status = "restarting"

	// Start agent in a new goroutine
	go sm.runInstance(instance)

	// Update status to running
	instance.Status = "running"
	return nil
}

// runInstance runs an instance in a goroutine
func (sm *ServiceManager) runInstance(instance *ServiceInstance) {
	// Determine the command to run
	cmdName := "agent"
	if instance.Type == "proxy" {
		cmdName = "proxy"
	}

	// Execute command
	cmd := exec.CommandContext(instance.Ctx, os.Args[0], append([]string{cmdName}, instance.Args...)...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	instance.ErrChan <- err

	// Update status
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if err != nil {
		instance.Status = fmt.Sprintf("error: %v", err)
	} else {
		instance.Status = "stopped"
	}
}

// StopAgent stops a running agent instance
func (sm *ServiceManager) StopAgent(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if agent exists
	instance, exists := sm.agents[id]
	if !exists {
		return fmt.Errorf("agent with ID %s not found", id)
	}

	// Cancel the context to stop the agent
	instance.Cancel()
	instance.Status = "stopping"

	return nil
}

// DeleteAgent deletes an agent instance
func (sm *ServiceManager) DeleteAgent(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if agent exists
	instance, exists := sm.agents[id]
	if !exists {
		return fmt.Errorf("agent with ID %s not found", id)
	}

	// Stop the agent if it's running
	if instance.Status == "running" || instance.Status == "starting" {
		instance.Cancel()
	}

	// Remove from manager
	delete(sm.agents, id)

	return nil
}

// GetAgents returns all agent instances
func (sm *ServiceManager) GetAgents() []*ServiceInstance {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Return a slice to avoid race conditions
	agents := make([]*ServiceInstance, 0, len(sm.agents))
	for _, instance := range sm.agents {
		agents = append(agents, instance)
	}

	return agents
}

// StartProxy starts a new proxy instance
func (sm *ServiceManager) StartProxy(args []string) (*ServiceInstance, error) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Generate unique ID
	id := uuid.New().String()[:8] // Use first 8 chars of UUID for brevity

	// Create context for the proxy
	ctx, cancel := context.WithCancel(context.Background())

	// Create service instance
	instance := &ServiceInstance{
		ID:      id,
		Type:    "proxy",
		Args:    args,
		Ctx:     ctx,
		Cancel:  cancel,
		Status:  "starting",
		ErrChan: make(chan error, 1),
	}

	// Add to manager
	sm.proxies[id] = instance

	// Start proxy in a goroutine
	go sm.runInstance(instance)

	// Update status to running
	instance.Status = "running"
	return instance, nil
}

// RestartProxy restarts an existing proxy instance
func (sm *ServiceManager) RestartProxy(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if proxy exists
	instance, exists := sm.proxies[id]
	if !exists {
		return fmt.Errorf("proxy with ID %s not found", id)
	}

	// Stop the existing instance
	instance.Cancel()

	// Create new context for the proxy
	ctx, cancel := context.WithCancel(context.Background())

	// Update instance with new context and status
	instance.Ctx = ctx
	instance.Cancel = cancel
	instance.Status = "restarting"

	// Start proxy in a new goroutine
	go sm.runInstance(instance)

	// Update status to running
	instance.Status = "running"
	return nil
}

// StopProxy stops a running proxy instance
func (sm *ServiceManager) StopProxy(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if proxy exists
	instance, exists := sm.proxies[id]
	if !exists {
		return fmt.Errorf("proxy with ID %s not found", id)
	}

	// Cancel the context to stop the proxy
	instance.Cancel()
	instance.Status = "stopping"

	return nil
}

// DeleteProxy deletes a proxy instance
func (sm *ServiceManager) DeleteProxy(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if proxy exists
	instance, exists := sm.proxies[id]
	if !exists {
		return fmt.Errorf("proxy with ID %s not found", id)
	}

	// Stop the proxy if it's running
	if instance.Status == "running" || instance.Status == "starting" {
		instance.Cancel()
	}

	// Remove from manager
	delete(sm.proxies, id)

	return nil
}

// GetProxies returns all proxy instances
func (sm *ServiceManager) GetProxies() []*ServiceInstance {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Return a slice to avoid race conditions
	proxies := make([]*ServiceInstance, 0, len(sm.proxies))
	for _, instance := range sm.proxies {
		proxies = append(proxies, instance)
	}

	return proxies
}
