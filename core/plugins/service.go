package plugins

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

	"github.com/innomon/aigen-app/core/bizdefs"
	"github.com/innomon/aigen-app/core/services"
)

type PluginService struct {
	pluginsDir string
	plugins    map[string]*PluginInfo
	mu         sync.RWMutex

	// trustedKeys maps fingerprints to public keys
	trustedKeys map[string]*rsa.PublicKey

	// Services needed for mounting
	SchemaService    *services.SchemaService
	EvolutionService services.IEvolutionService
	ChatService      *services.ChatService
	
	// Sandbox
	Dispatcher *SandboxDispatcher
}

func NewPluginService(pluginsDir string, schemaService *services.SchemaService, evolutionService services.IEvolutionService, chatService *services.ChatService, entityService services.IEntityService, a2uiService services.IA2UIService) *PluginService {
	hostAPI := &AIGenHostAPI{
		EntityService: entityService,
		A2UIService:   a2uiService,
	}
	
	return &PluginService{
		pluginsDir:       pluginsDir,
		plugins:          make(map[string]*PluginInfo),
		trustedKeys:      make(map[string]*rsa.PublicKey),
		SchemaService:    schemaService,
		EvolutionService: evolutionService,
		ChatService:      chatService,
		Dispatcher:       NewSandboxDispatcher(hostAPI),
	}
}

func (s *PluginService) MountPlugin(ctx context.Context, id string) error {
	s.mu.RLock()
	info, ok := s.plugins[id]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("plugin %s not found", id)
	}

	if !info.IsVerified {
		return fmt.Errorf("plugin %s is not verified", id)
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
		if err := s.Dispatcher.RegisterPluginTools(s.ChatService.Registry, r, info.Manifest.ID); err != nil {
			log.Printf("Warning: failed to register plugin tools for %s: %v", info.Manifest.ID, err)
		}
	}

	s.mu.Lock()
	info.Status = StatusActive
	s.mu.Unlock()

	log.Printf("Plugin %s mounted successfully", info.Manifest.ID)
	return nil
}

func (s *PluginService) Start(ctx context.Context) error {
	if err := os.MkdirAll(s.pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	// Initial scan
	if err := s.Scan(); err != nil {
		log.Printf("Error during initial plugin scan: %v", err)
	}

	// TODO: Implement watcher (fsnotify)
	
	return nil
}

func (s *PluginService) Scan() error {
	files, err := os.ReadDir(s.pluginsDir)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() || filepath.Ext(f.Name()) != ".jar" {
			continue
		}

		path := filepath.Join(s.pluginsDir, f.Name())
		if err := s.LoadPlugin(path); err != nil {
			log.Printf("Failed to load plugin %s: %v", path, err)
		}
	}

	return nil
}

func (s *PluginService) LoadPlugin(path string) error {
	info, err := s.inspectJar(path)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.plugins[info.Manifest.ID] = info
	s.mu.Unlock()

	log.Printf("Discovered plugin: %s (Version: %s, Status: %s)", info.Manifest.Name, info.Manifest.Version, info.Status)
	return nil
}

func (s *PluginService) inspectJar(path string) (*PluginInfo, error) {
	r, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var manifest PluginManifest
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
		} else if f.Name == "META-INF/PLUGIN.SF" {
			signatureFile = f
		} else if f.Name == "META-INF/PLUGIN.RSA" {
			certFile = f
		}
	}

	if !hasMetadata {
		id := strings.TrimSuffix(filepath.Base(path), ".jar")
		manifest = PluginManifest{
			ID:      id,
			Name:    id,
			Version: "0.0.1",
		}
	}

	info := &PluginInfo{
		Manifest:  manifest,
		Path:      path,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if signatureFile == nil || certFile == nil {
		info.Status = StatusUntrusted
		info.Error = "Plugin is unsigned (missing META-INF/PLUGIN.SF or PLUGIN.RSA)"
		return info, nil
	}

	// Verify Signature
	signer, err := s.verifyJarSignature(r, signatureFile, certFile)
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

func (s *PluginService) verifyJarSignature(r *zip.ReadCloser, sf, rsaFile *zip.File) (string, error) {
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
	// scheme where PLUGIN.RSA is just a PEM-encoded public key + signature
	// or we use a library for PKCS7.
	
	// Simplified logic for brainstorming/POC:
	// We check if the sfData matches the manifest digests of other files.
	
	// TODO: Full PKCS7 verification.
	// For now, let's extract the "signer" name from a hypothetical PEM if it exists,
	// or just return a mock fingerprint.
	
	return "CN=AiGen Trusted Developer, O=AiGen", nil
}

func (s *PluginService) All() []*PluginInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	list := make([]*PluginInfo, 0, len(s.plugins))
	for _, p := range s.plugins {
		list = append(list, p)
	}
	return list
}

func (s *PluginService) Get(id string) (*PluginInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.plugins[id]
	return p, ok
}
