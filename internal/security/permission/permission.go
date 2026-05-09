package permission

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"ferryman-agent/internal/pubsub"
	"github.com/google/uuid"
)

var ErrorPermissionDenied = errors.New("permission denied")

type CreatePermissionRequest struct {
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type Service interface {
	pubsub.Subscriber[PermissionRequest]
	GrantPersistant(permission PermissionRequest)
	Grant(permission PermissionRequest)
	Deny(permission PermissionRequest)
	Request(opts CreatePermissionRequest) bool
	AutoApproveSession(sessionID string)
}

type permissionService struct {
	*pubsub.Broker[PermissionRequest]
	sessionPermissions  []PermissionRequest
	pendingRequests     sync.Map
	autoApproveSessions []string
	workingDir          string
}

func NewService(workingDir ...string) Service {
	root := ""
	if len(workingDir) > 0 {
		root = workingDir[0]
	}
	return &permissionService{
		Broker:             pubsub.NewBroker[PermissionRequest](),
		sessionPermissions: make([]PermissionRequest, 0),
		workingDir:         root,
	}
}

func NewServiceWithWorkingDir(workingDir string) Service {
	return NewService(workingDir)
}

func (s *permissionService) GrantPersistant(permission PermissionRequest) {
	if respCh, ok := s.pendingRequests.Load(permission.ID); ok {
		respCh.(chan bool) <- true
	}
	s.sessionPermissions = append(s.sessionPermissions, permission)
}

func (s *permissionService) Grant(permission PermissionRequest) {
	if respCh, ok := s.pendingRequests.Load(permission.ID); ok {
		respCh.(chan bool) <- true
	}
}

func (s *permissionService) Deny(permission PermissionRequest) {
	if respCh, ok := s.pendingRequests.Load(permission.ID); ok {
		respCh.(chan bool) <- false
	}
}

func (s *permissionService) Request(opts CreatePermissionRequest) bool {
	if slices.Contains(s.autoApproveSessions, opts.SessionID) {
		return true
	}
	dir := filepath.Dir(opts.Path)
	if dir == "." {
		dir = s.root()
	}
	permission := PermissionRequest{
		ID:          uuid.New().String(),
		Path:        dir,
		SessionID:   opts.SessionID,
		ToolName:    opts.ToolName,
		Description: opts.Description,
		Action:      opts.Action,
		Params:      opts.Params,
	}
	for _, p := range s.sessionPermissions {
		if p.ToolName == permission.ToolName && p.Action == permission.Action && p.SessionID == permission.SessionID && p.Path == permission.Path {
			return true
		}
	}
	respCh := make(chan bool, 1)
	s.pendingRequests.Store(permission.ID, respCh)
	defer s.pendingRequests.Delete(permission.ID)
	s.Publish(pubsub.CreatedEvent, permission)
	return <-respCh
}

func (s *permissionService) AutoApproveSession(sessionID string) {
	s.autoApproveSessions = append(s.autoApproveSessions, sessionID)
}

func (s *permissionService) root() string {
	if s.workingDir != "" {
		return s.workingDir
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}
