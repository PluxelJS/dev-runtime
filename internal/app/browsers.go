package app

import (
	"context"
	"errors"

	"github.com/PluxelJS/Proxy-LLM-API/internal/state"
)

const playwrightVersion = "1.64.0"

// Browser services are opt-in and expose only loopback endpoints.
func browserDefaults() []state.Service {
	base := func(id, image string, port uint16, args []string) state.Service {
		return state.Service{ID: id, Revision: 1, Image: image, Restart: "unless-stopped",
			Args: args, Env: map[string]string{}, Mounts: []state.Mount{},
			Ports: []state.Port{{Host: "127.0.0.1", Published: port, Target: port}}}
	}
	pw := base("playwright", "mcr.microsoft.com/playwright:v"+playwrightVersion+"-noble", 3000,
		[]string{"npx", "--yes", "playwright@" + playwrightVersion, "run-server", "--host", "0.0.0.0", "--port", "3000"})
	pw.User = "pwuser"
	pw.Env["PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD"] = "1"
	lp := base("lightpanda", "docker.io/lightpanda/browser:1.0.0", 9222,
		[]string{"/bin/lightpanda", "serve", "--host", "0.0.0.0", "--port", "9222"})
	lp.Env["LIGHTPANDA_DISABLE_TELEMETRY"] = "true"
	return []state.Service{pw, lp}
}

// Called with the workspace lock held. Never regenerate secrets/files or
// overwrite user configuration while adding presets to an existing workspace.
func (a *App) ensureBrowserDefaults(ctx context.Context) error {
	for _, s := range browserDefaults() {
		_, err := a.Service(ctx, s.ID)
		if errors.Is(err, state.ErrNotFound) {
			if err = a.Store.Put(ctx, "service", s.ID, s); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

// Existing, complete workspaces retain lock-free reads during long operations.
func (a *App) upgradeBrowserDefaults(ctx context.Context) error {
	for _, s := range browserDefaults() {
		_, err := a.Service(ctx, s.ID)
		if errors.Is(err, state.ErrNotFound) {
			unlock, err := a.Store.Lock(ctx)
			if err != nil {
				return err
			}
			defer unlock()
			return a.ensureBrowserDefaults(ctx)
		}
		if err != nil {
			return err
		}
	}
	return nil
}
