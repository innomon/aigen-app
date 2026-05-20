package plugins

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"path/filepath"

	"github.com/innomon/agentic/pkg/registry"
)

// SandboxDispatcher handles the execution of scripts from plugins.
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
	// This is where we'd parse agentic.yaml and for each tool with a sandbox:// handler,
	// we register a tool handler in the registry that calls d.Execute.
	
	// Mock implementation for the POC:
	// Let's assume we find a tool named "calculate_risk" in the plugin.
	
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

	// Prepare Sandbox Environment
	env := d.prepareEnv(ctx, pluginID)

	switch ext {
	case ".js":
		return d.executeJS(ctx, string(data), args, env)
	case ".lua":
		return d.executeLua(ctx, string(data), args, env)
	case ".star":
		return d.executeStarlark(ctx, string(data), args, env)
	case ".wasm":
		return d.executeWASM(ctx, data, args, env)
	default:
		return nil, fmt.Errorf("unsupported script extension: %s", ext)
	}
}

type SandboxEnv struct {
	GetSecret func(key string) (string, bool)
	Fetch     func(url string, options map[string]any) (any, error)
}

func (d *SandboxDispatcher) prepareEnv(ctx context.Context, pluginID string) *SandboxEnv {
	return &SandboxEnv{
		GetSecret: func(key string) (string, bool) {
			if d.PluginService == nil {
				return "", false
			}
			// Only allow if listed in manifest
			info, ok := d.PluginService.Get(pluginID)
			if !ok {
				return "", false
			}
			allowed := false
			for _, v := range info.Manifest.EnvVars {
				if v == key {
					allowed = true
					break
				}
			}
			if !allowed {
				log.Printf("Security Alert: Plugin %s attempted to access unlisted env var %s", pluginID, key)
				return "", false
			}
			return d.PluginService.GetSecret(pluginID, key)
		},
		Fetch: func(url string, options map[string]any) (any, error) {
			if d.PluginService == nil {
				return nil, fmt.Errorf("plugin service not available")
			}
			// Check permissions
			allowed := false
			d.PluginService.mu.RLock()
			for _, g := range d.PluginService.grants {
				if g.PluginID == pluginID && g.Type == "http" {
					// Simplified glob match
					if strings.Contains(url, strings.ReplaceAll(g.Value, "*", "")) {
						allowed = true
						break
					}
				}
			}
			d.PluginService.mu.RUnlock()

			if !allowed {
				return nil, fmt.Errorf("unauthorized HTTP access to %s. Ask admin for 'http' permission for %s", url, url)
			}

			log.Printf("Plugin %s performing authorized HTTP fetch to %s", pluginID, url)
			// Implementation would use http.Client here
			return map[string]any{"status": 200, "body": "Mock response"}, nil
		},
	}
}

func (d *SandboxDispatcher) executeJS(ctx context.Context, script string, args map[string]any, env *SandboxEnv) (any, error) {
	// TODO: Integrate with goja/otto and inject env
	log.Printf("Executing JS script with SandboxEnv (Mock)...")
	return map[string]any{"status": "success", "result": 42}, nil
}

func (d *SandboxDispatcher) executeLua(ctx context.Context, script string, args map[string]any, env *SandboxEnv) (any, error) {
	log.Printf("Executing Lua script (Sandbox Mock)...")
	return nil, fmt.Errorf("lua sandbox not yet implemented")
}

func (d *SandboxDispatcher) executeStarlark(ctx context.Context, script string, args map[string]any, env *SandboxEnv) (any, error) {
	log.Printf("Executing Starlark script (Sandbox Mock)...")
	return nil, fmt.Errorf("starlark sandbox not yet implemented")
}

func (d *SandboxDispatcher) executeWASM(ctx context.Context, binary []byte, args map[string]any, env *SandboxEnv) (any, error) {
	log.Printf("Executing WASM script (Sandbox Mock)...")
	return nil, fmt.Errorf("wasm sandbox not yet implemented")
}
