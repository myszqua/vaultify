package vaultify

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchManagerEvents(t *testing.T) {
	p := NewDefaultProvider(map[string]any{"key": "value"})
	loader := New(p)

	err := loader.Load(context.Background())
	require.NoError(t, err)

	wm := NewWatchManager(loader, newNopLogger())

	eventReceived := make(chan WatchEvent, 1)

	wm.OnChange(func(event WatchEvent) {
		eventReceived <- event
	})

	err = wm.Start(context.Background())
	require.NoError(t, err)

	err = wm.Stop()
	assert.NoError(t, err)
}

func TestWatchManagerEventsChannel(t *testing.T) {
	p := NewDefaultProvider(map[string]any{})
	loader := New(p)
	wm := NewWatchManager(loader, newNopLogger())

	events := wm.Events()
	assert.NotNil(t, events)
}

func TestListenForReload(_ *testing.T) {
	p := NewDefaultProvider(map[string]any{"key": "value"})
	loader := New(p)

	ctx, cancel := context.WithCancel(context.Background())

	_ = loader.Load(ctx)

	ListenForReload(ctx, loader)

	time.Sleep(10 * time.Millisecond)
	cancel()
}
