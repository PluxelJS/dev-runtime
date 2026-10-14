package app

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestBrowserUpgradePreservesWorkspace(t *testing.T) {
	for _, entry := range []string{"init", "open"} {
		t.Run(entry, func(t *testing.T) {
			a, _ := testApp(t)
			ctx := context.Background()
			pw, err := a.Service(ctx, "playwright")
			if err != nil {
				t.Fatal(err)
			}
			pw.Image = "example.test/custom:1"
			pw.Enabled = true
			if err = a.Save(ctx, pw); err != nil {
				t.Fatal(err)
			}
			before, _ := a.Service(ctx, "playwright")
			secret, _ := a.Store.Secret(ctx, "postgres-admin")
			path := a.Store.File("cliproxy/settings/config.yaml")
			if err = os.WriteFile(path, []byte("custom-config"), 0600); err != nil {
				t.Fatal(err)
			}
			if err = a.Store.Delete(ctx, "service", "lightpanda"); err != nil {
				t.Fatal(err)
			}
			for i := 0; i < 2; i++ {
				if entry == "init" {
					err = Init(ctx, a.Store.Dir, "auto", "")
				} else {
					other, e := Open(ctx, a.Store.Dir)
					err = e
					if other != nil {
						other.Close()
					}
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			after, _ := a.Service(ctx, "playwright")
			if !reflect.DeepEqual(before, after) {
				t.Fatal("custom browser overwritten")
			}
			lp, err := a.Service(ctx, "lightpanda")
			if err != nil || lp.Enabled || lp.Revision != 1 {
				t.Fatalf("new preset: %+v, %v", lp, err)
			}
			data, _ := os.ReadFile(path)
			afterSecret, _ := a.Store.Secret(ctx, "postgres-admin")
			if string(data) != "custom-config" || secret != afterSecret {
				t.Fatal("existing files/secrets changed")
			}
		})
	}
}

func TestBrowserPresetsAndDiagnosis(t *testing.T) {
	a, f := testApp(t)
	ctx := context.Background()
	for _, id := range []string{"playwright", "lightpanda"} {
		t.Run(id, func(t *testing.T) {
			s, err := a.Service(ctx, id)
			if err != nil {
				t.Fatal(err)
			}
			if err = Validate(s); err != nil {
				t.Fatal(err)
			}
			if s.Enabled || s.Ports[0].Host != "127.0.0.1" {
				t.Fatal("browser must be opt-in and loopback only")
			}
			if _, err = a.Lifecycle(ctx, "start", id, s.Revision); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(f.creates[len(f.creates)-1].Service, s) {
				t.Fatal("preset not passed to engine")
			}
			path := "/"
			if id == "lightpanda" {
				path = "/json/version"
			}
			status := http.StatusOK
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != path {
					t.Errorf("path = %s, want %s", r.URL.Path, path)
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			s.Ports[0].Published = uint16(server.Listener.Addr().(*net.TCPAddr).Port)
			if err = a.Save(ctx, s); err != nil {
				t.Fatal(err)
			}
			d, err := a.Diagnose(ctx, id)
			if err != nil || !d.OK {
				t.Fatalf("healthy: %+v %v", d, err)
			}
			status = http.StatusServiceUnavailable
			d, err = a.Diagnose(ctx, id)
			if err == nil || d.OK {
				t.Fatalf("unhealthy: %+v %v", d, err)
			}
		})
	}
}

func TestOpenUpgradedWorkspaceDoesNotWaitForOperationLock(t *testing.T) {
	a, _ := testApp(t)
	ctx := context.Background()
	unlock, err := a.Store.Lock(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	other, err := Open(ctx, a.Store.Dir)
	if err != nil {
		t.Fatal(err)
	}
	other.Close()
}
