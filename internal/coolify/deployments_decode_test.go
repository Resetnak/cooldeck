package coolify

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Coolify wraps per-application deployment history in {count, deployments}
// (DeployController::get_application_deployments), and the running-queue
// endpoint serialises a key-preserving Laravel collection, which json_encode
// turns into a keyed object whenever sortBy reorders the rows. Both shapes
// must decode; a bare array must keep working for older instances.
func TestDeploymentsDecodeRealCoolifyShapes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/deployments/applications/app-1":
			_, _ = fmt.Fprint(w, `{"count":2,"deployments":[
				{"deployment_uuid":"dep-2","application_id":"app-1","status":"finished","created_at":"2026-08-05T10:00:00Z"},
				{"deployment_uuid":"dep-1","application_id":"app-1","status":"failed","created_at":"2026-08-04T10:00:00Z"}
			]}`)
		case "/api/v1/deployments":
			_, _ = fmt.Fprint(w, `{
				"1":{"deployment_uuid":"dep-3","application_id":"app-1","status":"in_progress","created_at":"2026-08-06T10:00:00Z"},
				"0":{"deployment_uuid":"dep-4","application_id":"app-2","status":"queued","created_at":"2026-08-06T10:01:00Z"}
			}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := newTestService(t, server, time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC))

	history, err := service.Deployments(context.Background(), "app-1", 10)
	if err != nil {
		t.Fatalf("Deployments() error = %v", err)
	}
	if len(history) != 2 || history[0].UUID != "dep-2" {
		t.Fatalf("history = %#v", history)
	}

	running, err := service.listDeployments(context.Background())
	if err != nil {
		t.Fatalf("listDeployments() error = %v", err)
	}
	if len(running) != 2 || running[0].UUID != "dep-4" {
		t.Fatalf("running = %#v", running)
	}
}

// The fleet Deployments screen is fed from Dashboard, and the running-queue
// endpoint never returns finished work - history must be merged in from the
// per-application endpoints, deduplicated against the live queue.
func TestDashboardMergesPerApplicationHistory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/applications":
			_, _ = fmt.Fprint(w, `[
				{"uuid":"app-1","name":"api","status":"running:healthy"},
				{"uuid":"app-2","name":"web","status":"running:healthy"}
			]`)
		case "/api/v1/deployments":
			_, _ = fmt.Fprint(w, `[{"deployment_uuid":"dep-live","application_id":"app-1","status":"in_progress","created_at":"2026-08-06T12:00:00Z"}]`)
		case "/api/v1/deployments/applications/app-1":
			_, _ = fmt.Fprint(w, `{"count":2,"deployments":[
				{"deployment_uuid":"dep-live","application_id":"app-1","status":"in_progress","created_at":"2026-08-06T12:00:00Z"},
				{"deployment_uuid":"dep-old","application_id":"app-1","status":"finished","created_at":"2026-08-05T12:00:00Z"}
			]}`)
		case "/api/v1/deployments/applications/app-2":
			_, _ = fmt.Fprint(w, `{"count":1,"deployments":[
				{"deployment_uuid":"dep-web","application_id":"app-2","status":"failed","created_at":"2026-08-06T09:00:00Z"}
			]}`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	service := newTestService(t, server, time.Date(2026, 8, 6, 13, 0, 0, 0, time.UTC))
	dashboard, err := service.Dashboard(context.Background())
	if err != nil {
		t.Fatalf("Dashboard() error = %v", err)
	}
	if len(dashboard.ActiveDeployments) != 1 || dashboard.ActiveDeployments[0].UUID != "dep-live" {
		t.Fatalf("active = %#v", dashboard.ActiveDeployments)
	}
	got := make([]string, 0, len(dashboard.RecentDeployments))
	for _, d := range dashboard.RecentDeployments {
		got = append(got, d.UUID)
	}
	want := []string{"dep-live", "dep-web", "dep-old"}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("recent = %v, want %v", got, want)
	}
}
