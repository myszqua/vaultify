package vaultify

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

// WatchManager coordinates provider watches, reloads the loader on changes,
// and dispatches events to registered callbacks.
type WatchManager struct {
	loader    *ConfigLoader
	logger    Logger
	events    chan WatchEvent
	callbacks []func(WatchEvent)
	mu        sync.Mutex
	cancel    context.CancelFunc
	stopOnce  sync.Once
}

// NewWatchManager creates a WatchManager for the given loader.
func NewWatchManager(loader *ConfigLoader, logger Logger) *WatchManager {
	return &WatchManager{
		loader: loader,
		logger: logger,
		events: make(chan WatchEvent, 100), //nolint:mnd
	}
}

// Start begins watching all loader providers and processing events.
func (wm *WatchManager) Start(ctx context.Context) error {
	watchCtx, cancel := context.WithCancel(ctx)
	wm.cancel = cancel

	wm.loader.mu.Lock()
	wm.loader.watchCtx = watchCtx
	wm.loader.watchStop = cancel
	wm.loader.mu.Unlock()

	for _, p := range wm.loader.providers {
		if err := p.Watch(watchCtx, wm.events); err != nil {
			wm.logger.Warn("failed to start watch for provider ", p.Name(), ": ", err)

			continue
		}

		wm.logger.Debug("watch started for provider: ", p.Name())
	}

	go wm.processEvents(watchCtx)

	return nil
}

func (wm *WatchManager) processEvents(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-wm.events:
			wm.handleEvent(event)
		}
	}
}

func (wm *WatchManager) handleEvent(event WatchEvent) {
	wm.logger.Debug("watch event from ", event.Source, ": ", event.Type)

	if event.Type == EventError {
		wm.logger.Warn("watch error from ", event.Source, ": ", event.Error)

		return
	}

	if event.Type == EventChanged {
		wm.logger.Debug("reloading config due to change in ", event.Source)

		if err := wm.loader.Reload(wm.loader.watchCtx); err != nil {
			wm.logger.Warn("reload failed: ", err)

			return
		}
	}

	wm.mu.Lock()
	defer wm.mu.Unlock()

	for _, cb := range wm.callbacks {
		cb(event)
	}
}

// OnChange registers a callback invoked for every processed watch event.
func (wm *WatchManager) OnChange(callback func(WatchEvent)) {
	wm.mu.Lock()
	defer wm.mu.Unlock()

	wm.callbacks = append(wm.callbacks, callback)
}

// Events returns the internal watch event channel.
func (wm *WatchManager) Events() <-chan WatchEvent {
	return wm.events
}

// Stop cancels watching and stops all providers.
func (wm *WatchManager) Stop() error {
	wm.stopOnce.Do(func() {
		if wm.cancel != nil {
			wm.cancel()
		}

		for _, p := range wm.loader.providers {
			if err := p.Stop(); err != nil {
				wm.logger.Warn("failed to stop provider ", p.Name(), ": ", err)
			}
		}

		close(wm.events)
	})

	return nil
}

// ListenForReload reloads the loader when a SIGHUP signal is received.
func ListenForReload(ctx context.Context, loader *ConfigLoader) {
	sigs := make(chan os.Signal, 1) //nolint:mnd
	signal.Notify(sigs, syscall.SIGHUP)

	go func() {
		for {
			select {
			case <-ctx.Done():
				signal.Stop(sigs)

				return
			case <-sigs:
				if err := loader.Reload(ctx); err != nil {
					fmt.Fprintf(os.Stderr, "SIGHUP reload failed: %v\n", err) //nolint:errcheck
				}
			}
		}
	}()
}
