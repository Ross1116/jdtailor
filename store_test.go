package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewStoreCreatesLocalPaths(t *testing.T) {
	root := t.TempDir()

	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	for _, path := range []string{
		filepath.Join(root, "data", "app.db"),
		filepath.Join(root, "logs", "app.log"),
		filepath.Join(root, "generated"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("expected %s to exist: %v", path, err)
		}
	}
}

func TestMigrationsAreIdempotent(t *testing.T) {
	root := t.TempDir()

	store, err := NewStore(root)
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()
	if err := store.migrate(t.Context()); err != nil {
		t.Fatalf("second migrate() error = %v", err)
	}

	var count int
	if err := store.db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if count != 1 {
		t.Fatalf("migration version count = %d, want 1", count)
	}
}

func TestSettingsSaveLoad(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	initial, err := store.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() error = %v", err)
	}
	if initial.Provider != defaultProvider {
		t.Fatalf("Provider = %q, want %q", initial.Provider, defaultProvider)
	}
	if initial.APIKeyConfigured {
		t.Fatal("APIKeyConfigured = true, want false")
	}

	saved, err := store.SaveSettings(SaveSettingsInput{
		Provider: "openai",
		Model:    "gpt-test",
	})
	if err != nil {
		t.Fatalf("SaveSettings() error = %v", err)
	}
	if saved.Model != "gpt-test" {
		t.Fatalf("saved Model = %q, want gpt-test", saved.Model)
	}

	loaded, err := store.GetSettings()
	if err != nil {
		t.Fatalf("GetSettings() after save error = %v", err)
	}
	if loaded.Model != "gpt-test" {
		t.Fatalf("loaded Model = %q, want gpt-test", loaded.Model)
	}
}

func TestEventLogging(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore() error = %v", err)
	}
	defer store.Close()

	if err := store.LogEvent("warning", "test event"); err != nil {
		t.Fatalf("LogEvent() error = %v", err)
	}

	events, err := store.GetRecentEvents(5)
	if err != nil {
		t.Fatalf("GetRecentEvents() error = %v", err)
	}
	if len(events) == 0 {
		t.Fatal("GetRecentEvents() returned no events")
	}
	if events[0].Message != "test event" {
		t.Fatalf("latest event = %q, want test event", events[0].Message)
	}
}
