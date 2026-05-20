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
		return d.Execute(ctx, fsys, "scripts/calculate_risk.js", args)
	})

	return nil
}

func (d *SandboxDispatcher) Execute(ctx context.Context, fsys fs.FS, scriptPath string, args map[string]any) (any, error) {
	ext := filepath.Ext(scriptPath)
	data, err := fs.ReadFile(fsys, scriptPath)
	if err != nil {
		return nil, err
	}

	switch ext {
	case ".js":
		return d.executeJS(ctx, string(data), args)
	case ".lua":
		return d.executeLua(ctx, string(data), args)
	case ".star":
		return d.executeStarlark(ctx, string(data), args)
	case ".wasm":
		return d.executeWASM(ctx, data, args)
	default:
		return nil, fmt.Errorf("unsupported script extension: %s", ext)
	}
}

func (d *SandboxDispatcher) executeJS(ctx context.Context, script string, args map[string]any) (any, error) {
	// TODO: Integrate with github.com/robertkrimen/otto or goja
	log.Printf("Executing JS script (Sandbox Mock)...")
	return map[string]any{"status": "success", "result": 42}, nil
}

func (d *SandboxDispatcher) executeLua(ctx context.Context, script string, args map[string]any) (any, error) {
	log.Printf("Executing Lua script (Sandbox Mock)...")
	return nil, fmt.Errorf("lua sandbox not yet implemented")
}

func (d *SandboxDispatcher) executeStarlark(ctx context.Context, script string, args map[string]any) (any, error) {
	log.Printf("Executing Starlark script (Sandbox Mock)...")
	return nil, fmt.Errorf("starlark sandbox not yet implemented")
}

func (d *SandboxDispatcher) executeWASM(ctx context.Context, binary []byte, args map[string]any) (any, error) {
	log.Printf("Executing WASM script (Sandbox Mock)...")
	return nil, fmt.Errorf("wasm sandbox not yet implemented")
}
