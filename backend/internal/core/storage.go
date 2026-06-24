package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"software-web-manager/backend/internal/storage"
)

// EnsureStorage lazily initializes the storage driver on the handler, falling
// back to local storage when the configured driver is unavailable.
func (h *Handler) EnsureStorage() error {
	if h.Storage != nil {
		return nil
	}
	store, err := storage.New(context.Background(), h.Cfg)
	if err != nil && h.Cfg.StorageDriver != "local" {
		fallbackCfg := h.Cfg
		fallbackCfg.StorageDriver = "local"
		store, err = storage.New(context.Background(), fallbackCfg)
	}
	if err != nil {
		return err
	}
	h.Storage = store
	return nil
}

// DeleteStoragePaths removes the given object-storage paths, best-effort.
func (h *Handler) DeleteStoragePaths(ctx context.Context, paths []string) {
	if len(paths) == 0 {
		return
	}
	if err := h.EnsureStorage(); err != nil {
		return
	}
	if h.Storage == nil {
		return
	}
	for _, path := range paths {
		path = strings.TrimSpace(path)
		if path == "" {
			continue
		}
		_ = h.Storage.Delete(ctx, path)
	}
}

// DeleteLocalTicketDir removes a ticket's local-storage directory when using the
// local storage driver, best-effort.
func (h *Handler) DeleteLocalTicketDir(ticketID string) {
	if !strings.EqualFold(h.Cfg.StorageDriver, "local") {
		return
	}
	root := strings.TrimSpace(h.Cfg.LocalStoragePath)
	if root == "" || ticketID == "" {
		return
	}
	dir := filepath.Join(root, "tickets", ticketID)
	_ = os.RemoveAll(dir)
}
