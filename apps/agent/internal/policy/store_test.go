package policy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPutPersistsAndReloads(t *testing.T) {
	dir := t.TempDir()
	store, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.Put(Document{
		AutoCapture:     false,
		ArchiveAll:      false,
		CaptureDelayMs:  800,
		DomainAllowlist: []string{"Example.COM.", " example.com "},
		DomainDenylist:  []string{"ads.example.com"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Revision != 2 || got.AutoCapture || got.ArchiveAll || got.CaptureDelayMs != 800 {
		t.Fatalf("put=%+v", got)
	}
	if len(got.DomainAllowlist) != 1 || got.DomainAllowlist[0] != "example.com" {
		t.Fatalf("allowlist=%v", got.DomainAllowlist)
	}

	reloaded, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	again := reloaded.Get()
	if again.Revision != 2 || again.AutoCapture || again.DomainAllowlist[0] != "example.com" {
		t.Fatalf("reloaded=%+v", again)
	}
	if _, err := os.Stat(filepath.Join(dir, fileName)); err != nil {
		t.Fatal(err)
	}
}

func TestInvalidDelayFallsBack(t *testing.T) {
	store := Memory()
	got, err := store.Put(Document{CaptureDelayMs: 10, ArchiveAll: true, AutoCapture: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.CaptureDelayMs != defaultDelayMs {
		t.Fatalf("delay=%d", got.CaptureDelayMs)
	}
}
