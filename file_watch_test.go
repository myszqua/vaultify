package vaultify

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFileProviderWatch(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(path, []byte("key: value\n"), 0o600)
	require.NoError(t, err)

	provider := NewFileProvider([]string{path})

	events := make(chan WatchEvent, 1)
	err = provider.Watch(context.Background(), events)
	require.NoError(t, err)

	defer func() {
		_ = provider.Stop()
	}()

	time.Sleep(50 * time.Millisecond)

	err = os.WriteFile(path, []byte("key: newvalue\n"), 0o600)
	require.NoError(t, err)

	select {
	case event := <-events:
		assert.Equal(t, "file", event.Source)
		assert.Equal(t, EventChanged, event.Type)
	case <-time.After(2 * time.Second):
		t.Fatal("expected file change event")
	}
}

func TestWatchManagerReloadOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	err := os.WriteFile(path, []byte("key: initial\n"), 0o600)
	require.NoError(t, err)

	fileProvider := NewFileProvider([]string{path}, WithFilePriority(20))
	loader := New(fileProvider)

	err = loader.Load(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "initial", loader.GetString("key"))

	wm := NewWatchManager(loader, newNopLogger())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	err = wm.Start(ctx)
	require.NoError(t, err)

	time.Sleep(50 * time.Millisecond)

	err = os.WriteFile(path, []byte("key: updated\n"), 0o600)
	require.NoError(t, err)

	time.Sleep(500 * time.Millisecond)

	assert.Equal(t, "updated", loader.GetString("key"))
}
