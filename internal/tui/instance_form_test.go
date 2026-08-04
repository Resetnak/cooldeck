package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/resetnak/cooldeck/internal/app/demo"
	"github.com/resetnak/cooldeck/internal/config"
	"github.com/resetnak/cooldeck/internal/tui/components"
)

func TestAddInstanceFormValidatesAndSaves(t *testing.T) {
	cfg := config.Default()
	cfg.Instances = map[string]config.Instance{}
	var saved config.Config
	var storedKey, storedTok string

	model := New(Options{
		Config:  cfg,
		Service: demo.New(demo.Options{}),
		ASCII:   true,
		SaveConfig: func(c config.Config) error {
			saved = c
			return nil
		},
		StoreCredentials: func(inst config.Instance, token string) error {
			storedKey = inst.ID
			storedTok = token
			return nil
		},
	})

	model.openAddInstanceForm()
	if model.instanceForm == nil {
		t.Fatal("form not opened")
	}

	// Invalid submit.
	model.instanceForm.id = "Bad ID"
	model.instanceForm.name = "X"
	model.instanceForm.url = "not-a-url"
	model.instanceForm.token = "tok"
	model.submitInstanceForm()
	if model.instanceForm.err == "" {
		t.Fatal("expected validation error")
	}

	// Fill valid fields via key handler path.
	model.instanceForm = newAddInstanceForm()
	typeIn := func(s string) {
		for _, r := range s {
			model.handleInstanceFormKey(tea.KeyPressMsg{Code: r, Text: string(r)})
		}
	}
	typeIn("staging")
	model.handleInstanceFormKey(tea.KeyPressMsg{Code: tea.KeyTab})
	typeIn("Staging")
	model.handleInstanceFormKey(tea.KeyPressMsg{Code: tea.KeyTab})
	typeIn("https://coolify.example.com")
	model.handleInstanceFormKey(tea.KeyPressMsg{Code: tea.KeyTab})
	typeIn("secret-token")

	cmd := model.submitInstanceForm()
	if model.instanceForm != nil {
		t.Fatalf("form still open: %s", model.instanceForm.err)
	}
	if storedKey != "staging" || storedTok != "secret-token" {
		t.Fatalf("credentials = %q %q", storedKey, storedTok)
	}
	if saved.Instances["staging"].URL != "https://coolify.example.com" {
		t.Fatalf("saved url = %#v", saved.Instances["staging"])
	}
	if cmd == nil {
		t.Fatal("expected success toast")
	}
	// Toast cmd.
	if msg := cmd(); msg != nil {
		if toast, ok := msg.(toastMsg); ok && toast.Kind != int(components.ToastSuccess) {
			t.Fatalf("toast = %+v", toast)
		}
	}
}

func TestEditInstanceFormUpdatesNameURL(t *testing.T) {
	cfg := config.Default()
	cfg.Instances = map[string]config.Instance{
		"prod": {ID: "prod", Name: "Old", URL: "https://old.example", TokenSource: config.TokenSourceKeyring},
	}
	var saved config.Config
	model := New(Options{
		Config:  cfg,
		Service: labeledService{Service: demo.New(demo.Options{}), id: "prod", name: "Old"},
		ASCII:   true,
		SaveConfig: func(c config.Config) error {
			saved = c
			return nil
		},
	})
	model.refreshInstances()
	model.section = SectionInstances
	model.openEditInstanceForm()
	if model.instanceForm == nil || model.instanceForm.mode != instanceFormEdit {
		t.Fatal("edit form not open")
	}
	model.instanceForm.name = "Production"
	model.instanceForm.url = "https://new.example"
	model.submitInstanceForm()
	if saved.Instances["prod"].Name != "Production" || saved.Instances["prod"].URL != "https://new.example" {
		t.Fatalf("saved = %#v", saved.Instances["prod"])
	}
	if saved.Instances["prod"].TokenSource != config.TokenSourceKeyring {
		t.Fatal("token source should be unchanged")
	}
}

func TestDeploymentsActiveFilter(t *testing.T) {
	// Covered primarily in views; ensure model toggle path works.
	model := New(Options{Config: config.Default(), Service: demo.New(demo.Options{}), Demo: true, ASCII: true})
	model.section = SectionDeployments
	if model.deployments.ActiveOnly() {
		t.Fatal("default should show history")
	}
	model.handleDeploymentsKey(tea.KeyPressMsg{Code: 'a', Text: "a"})
	if !model.deployments.ActiveOnly() {
		t.Fatal("toggle did not enable active-only")
	}
}

func TestInstanceFormEscCloses(t *testing.T) {
	model := New(Options{
		Config: config.Default(), Service: demo.New(demo.Options{}), ASCII: true,
		SaveConfig: func(config.Config) error { return nil },
	})
	model.openAddInstanceForm()
	model.handleKey(tea.KeyPressMsg{Code: tea.KeyEscape})
	if model.instanceForm != nil {
		t.Fatal("esc did not close form")
	}
	_ = strings.Contains
}
