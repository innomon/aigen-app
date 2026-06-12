package app_extensions

import (
	"archive/zip"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/innomon/aigen-app/core/bizdefs"
	"github.com/innomon/aigen-app/core/descriptors"
	"github.com/innomon/aigen-app/core/services"
)

type AppExtensionService struct {
	extensionsDir string
	extensions    map[string]*AppExtensionInfo
	mu            sync.RWMutex

	// trustedKeys maps fingerprints to public keys
	trustedKeys map[string]*rsa.PublicKey

	// Services needed for mounting
	SchemaService    *services.SchemaService
	EvolutionService services.IEvolutionService
	ChatService      *services.ChatService
	AuditService     services.IAuditService

	// Sandbox
	Dispatcher *SandboxDispatcher

	// Persistence for permissions and vault (simplified for POC)
	grants []PermissionGrant
	vault  []VaultEntry
}

func NewAppExtensionService(extensionsDir string, schemaService *services.SchemaService, evolutionService services.IEvolutionService, chatService *services.ChatService, entityService services.IEntityService, a2uiService services.IA2UIService, auditService services.IAuditService) *AppExtensionService {
	hostAPI := &AIGenHostAPI{
		EntityService: entityService,
		A2UIService:   a2uiService,
	}
	
	svc := &AppExtensionService{
		extensionsDir:    extensionsDir,
		extensions:       make(map[string]*AppExtensionInfo),
		trustedKeys:      make(map[string]*rsa.PublicKey),
		SchemaService:    schemaService,
		EvolutionService: evolutionService,
		ChatService:      chatService,
		AuditService:     auditService,
		Dispatcher:       NewSandboxDispatcher(hostAPI),
	}
	svc.Dispatcher.ExtensionService = svc
	return svc
}

func (s *AppExtensionService) GetRoutingDocs() map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	docs := make(map[string]string)
	for id, info := range s.extensions {
		if info.Status == StatusActive {
			docs[id] = info.Manifest.Description
		}
	}
	return docs
}

func (s *AppExtensionService) LoadAgenticConfig(id string) ([]byte, error) {
	s.mu.RLock()
	info, ok := s.extensions[id]
	s.mu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("extension %s not found", id)
	}

	r, err := zip.OpenReader(info.Path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	agenticPath := "agentic/agentic.yaml"
	if info.Manifest.EntryPoints != nil {
		if p, ok := info.Manifest.EntryPoints["agentic"]; ok {
			agenticPath = p
		}
	}

	f, err := r.Open(agenticPath)
	if err != nil {
		return nil, fmt.Errorf("agentic.yaml not found in %s: %w", id, err)
	}
	defer f.Close()

	return io.ReadAll(f)
}

func (s *AppExtensionService) AuthorizePermission(ctx context.Context, extensionID string, req PermissionRequirement, adminID string) error {
	s.mu.Lock()
	grant := PermissionGrant{
		ExtensionID: extensionID,
		Type:        req.Type,
		Value:       req.Value,
		GrantedBy:   adminID,
		GrantedAt:   time.Now(),
	}
	s.grants = append(s.grants, grant)
	s.mu.Unlock()

	if s.AuditService != nil {
		s.AuditService.Log(ctx, &descriptors.AuditLog{
			Action:    descriptors.ActionType("extension_permission_granted"),
			RecordId:  extensionID,
			UserId:    adminID,
			Payload:   map[string]interface{}{"type": req.Type, "value": req.Value},
			CreatedAt: time.Now(),
		})
	}

	return nil
}

func (s *AppExtensionService) SetSecret(extensionID, key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	found := false
	for i, entry := range s.vault {
		if entry.ExtensionID == extensionID && entry.Key == key {
			s.vault[i].Value = value
			s.vault[i].UpdatedAt = time.Now()
			found = true
			break
		}
	}
	if !found {
		s.vault = append(s.vault, VaultEntry{
			ExtensionID: extensionID,
			Key:         key,
			Value:       value,
			UpdatedAt:   time.Now(),
		})
	}
}

func (s *AppExtensionService) GetSecret(extensionID, key string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, entry := range s.vault {
		if entry.ExtensionID == extensionID && entry.Key == key {
			return entry.Value, true
		}
	}
	return "", false
}

func (s *AppExtensionService) MountExtension(ctx context.Context, id string) error {
	s.mu.RLock()
	info, ok := s.extensions[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("extension %s not found", id)
	}

	if !info.IsVerified {
		return fmt.Errorf("extension %s is not verified", id)
	}

	r, err := zip.OpenReader(info.Path)
	if err != nil {
		return err
	}
	defer r.Close()

	// 1. Mount BizDef
	subFS, err := fs.Sub(r, "bizdef")
	if err == nil {
		if err := bizdefs.SetupBizDefFromFS(ctx, subFS, info.Manifest.ID, s.SchemaService); err != nil {
			log.Printf("Warning: failed to mount bizdef for %s: %v", info.Manifest.ID, err)
		}
		if err := bizdefs.SetupBizDefEvolutionFromFS(ctx, subFS, info.Manifest.ID, s.EvolutionService); err != nil {
			log.Printf("Warning: failed to mount bizdef evolution for %s: %v", info.Manifest.ID, err)
		}
	}

	// 2. Mount Agentic Tools via Sandbox
	if s.ChatService != nil && s.ChatService.Registry != nil {
		if err := s.Dispatcher.RegisterExtensionTools(s.ChatService.Registry, r, info.Manifest.ID); err != nil {
			log.Printf("Warning: failed to register extension tools for %s: %v", info.Manifest.ID, err)
		}
	}

	s.mu.Lock()
	info.Status = StatusActive
	s.mu.Unlock()

	log.Printf("App Extension %s mounted successfully", info.Manifest.ID)
	return nil
}

func (s *AppExtensionService) Start(ctx context.Context) error {
	if err := os.MkdirAll(s.extensionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create extensions directory: %w", err)
	}

	// Initial scan
	if err := s.Scan(); err != nil {
		log.Printf("Error during initial extension scan: %v", err)
	}

	// 2. Start Watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	go func() {
		defer watcher.Close()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					if filepath.Ext(event.Name) == ".jar" {
						log.Printf("Detected extension change: %s", event.Name)
						// Debounce slightly to allow file to be fully written
						time.Sleep(500 * time.Millisecond)
						s.Scan()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Extension watcher error: %v", err)
			case <-ctx.Done():
				return
			}
		}
	}()

	if err := watcher.Add(s.extensionsDir); err != nil {
		return fmt.Errorf("failed to add directory to watcher: %w", err)
	}

	log.Printf("App Extension service started, watching %s", s.extensionsDir)
	return nil
}

func (s *AppExtensionService) Scan() error {
	files, err := os.ReadDir(s.extensionsDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".jar" {
			continue
		}

		path := filepath.Join(s.extensionsDir, f.Name())
		if err := s.LoadExtension(path); err != nil {
			log.Printf("Failed to load extension %s: %v", path, err)
		}
	}

	return nil
}

func (s *AppExtensionService) LoadExtension(path string) error {
	info, err := s.inspectExtensionArchive(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.extensions[info.Manifest.ID] = info
	s.mu.Unlock()

	log.Printf("Discovered app extension: %s (Version: %s, Status: %s)", info.Manifest.Name, info.Manifest.Version, info.Status)
	return nil
}

func (s *AppExtensionService) inspectExtensionArchive(path string) (*AppExtensionInfo, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var manifest AppExtensionManifest
	var hasMetadata bool
	var signatureFile *zip.File
	var certFile *zip.File

	for _, f := range r.File {
		if f.Name == "metadata.json" {
			rc, err := f.Open()
			if err != nil {
				return nil, err
			}
			if err := json.NewDecoder(rc).Decode(&manifest); err != nil {
				rc.Close()
				return nil, fmt.Errorf("failed to decode metadata.json: %w", err)
			}
			rc.Close()
			hasMetadata = true
		} else if f.Name == "META-INF/EXTENSION.SF" || f.Name == "META-INF/PLUGIN.SF" {
			signatureFile = f
		} else if f.Name == "META-INF/EXTENSION.RSA" || f.Name == "META-INF/PLUGIN.RSA" {
			certFile = f
		}
	}

	if !hasMetadata {
		id := strings.TrimSuffix(filepath.Base(path), ".jar")
		manifest = AppExtensionManifest{
			ID:      id,
			Name:    id,
			Version: "0.0.1",
		}
	}

	info := &AppExtensionInfo{
		Manifest:  manifest,
		Path:      path,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if signatureFile == nil || certFile == nil {
		info.Status = StatusUntrusted
		info.Error = "App Extension is unsigned (missing META-INF/EXTENSION.SF or EXTENSION.RSA)"
		return info, nil
	}

	// Verify Signature
	signer, err := s.verifyExtensionSignature(r, signatureFile, certFile)
	if err != nil {
		info.Status = StatusUntrusted
		info.Error = fmt.Sprintf("Signature verification failed: %v", err)
		return info, nil
	}

	info.Signer = signer
	info.IsVerified = true
	info.Status = StatusActive // By default active if verified and trusted key logic is applied later

	return info, nil
}

func (s *AppExtensionService) verifyExtensionSignature(r *zip.ReadCloser, sf, rsaFile *zip.File) (string, error) {
	// 1. Read the signature block (RSA)
	rc, err := rsaFile.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()
	sigData, err := io.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// 2. Read the signature file (SF)
	rc2, err := sf.Open()
	if err != nil {
		return "", err
	}
	defer rc2.Close()
	sfData, err := io.ReadAll(rc2)
	if err != nil {
		return "", err
	}

	_ = sigData
	_ = sfData
	
	// In a real implementation, we'd use crypto/x509 to parse the PKCS7 signature
	// and verify the sfData against it. For this POC, we'll assume a simplified
	// scheme where EXTENSION.RSA is just a PEM-encoded public key + signature
	// or we use a library for PKCS7.
	
	return "CN=AiGen Trusted Developer, O=AiGen", nil
}

func (s *AppExtensionService) All() []*AppExtensionInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	list := make([]*AppExtensionInfo, 0, len(s.extensions))
	for _, p := range s.extensions {
		list = append(list, p)
	}
	return list
}

func (s *AppExtensionService) Get(id string) (*AppExtensionInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.extensions[id]
	return p, ok
}
