package plugins

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"path/filepath"

	_ "github.com/innomon/agentic/pkg/sandbox/engines/quickjs"
	"github.com/innomon/agentic/pkg/registry"
	"github.com/innomon/agentic/pkg/sandbox"
)

// SandboxDispatcher handles the execution of scripts from plugins using the agentic sandbox manager.
type SandboxDispatcher struct {
	// HostAPI provides the CMS services to the sandbox
	HostAPI interface{}
	// PluginService used for permission and vault checks
	PluginService *PluginService
}

func NewSandboxDispatcher(hostAPI interface{}) *SandboxDispatcher {
	return &SandboxDispatcher{HostAPI: hostAPI}
}

// RegisterPluginTools reads the agentic config and registers tools that use sandboxes.
func (d *SandboxDispatcher) RegisterPluginTools(reg *registry.Registry, fsys fs.FS, pluginID string) error {
	// In a real scenario, we'd parse agentic.yaml to find tools with type: sandbox
	// Mocking for POC:
	registry.RegisterToolHandler(fmt.Sprintf("%s_calculate_risk", pluginID), func(ctx context.Context, args map[string]any) (any, error) {
		return d.Execute(ctx, pluginID, fsys, "scripts/calculate_risk.js", args)
	})

	return nil
}

func (d *SandboxDispatcher) Execute(ctx context.Context, pluginID string, fsys fs.FS, scriptPath string, args map[string]any) (any, error) {
	ext := filepath.Ext(scriptPath)
	data, err := fs.ReadFile(fsys, scriptPath)
	if err != nil {
		return nil, err
	}

	// 1. Resolve VM Type
	vmType := ""
	switch ext {
	case ".js":
		vmType = "quickjs"
	case ".lua":
		vmType = "lua"
	case ".star":
		vmType = "starlark"
	case ".wasm":
		vmType = "wasm"
	default:
		return nil, fmt.Errorf("unsupported script extension: %s", ext)
	}

	// 2. Prepare VM Config from Manifest and Grants
	cfg, err := d.prepareVMConfig(pluginID, vmType)
	if err != nil {
		return nil, err
	}

	// 3. Initialize Agentic Sandbox Manager
	var logBuf bytes.Buffer
	host := &sandbox.HostContext{
		// Tools: d.getPluginTools(pluginID), // Could bridge local tools here
		Logger: &logBuf,
	}
	manager := sandbox.NewManager(host)

	// 4. Run Script
	vm, err := manager.GetOrCreateVM(pluginID, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create sandbox VM: %w", err)
	}

	val, err := vm.Run(ctx, string(data))
	if err != nil {
		return nil, fmt.Errorf("sandbox execution failed: %w (Logs: %s)", err, logBuf.String())
	}

	return val, nil
}

func (d *SandboxDispatcher) prepareVMConfig(pluginID string, vmType string) (sandbox.VMConfig, error) {
	cfg := sandbox.VMConfig{
		Type: vmType,
	}

	if d.PluginService == nil {
		return cfg, nil
	}

	info, ok := d.PluginService.Get(pluginID)
	if !ok {
		return cfg, fmt.Errorf("plugin %s not found", pluginID)
	}

	// Map EnvVars (Secrets) to VM Env
	cfg.Env = make(map[string]string)
	for _, key := range info.Manifest.EnvVars {
		if val, ok := d.PluginService.GetSecret(pluginID, key); ok {
			cfg.Env[key] = val
		}
	}

	// Map Permissions (HTTP) to AllowNet
	d.PluginService.mu.RLock()
	for _, grant := range d.PluginService.grants {
		if grant.PluginID == pluginID && grant.Type == "http" {
			cfg.AllowNet = append(cfg.AllowNet, grant.Value)
		}
	}
	d.PluginService.mu.RUnlock()

	return cfg, nil
}
