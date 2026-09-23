package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyAndReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "payload.txt")
	if err := os.WriteFile(file, []byte("bot-ctl"), 0o644); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("bot-ctl"))
	checksum := file + ".sha256"
	if err := os.WriteFile(checksum, []byte(hex.EncodeToString(sum[:])+"  payload.txt\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256(file, checksum); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(dir, "bot-ctl")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	updated := filepath.Join(dir, "bot-ctl.new")
	if err := os.WriteFile(updated, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := replaceBinary(target, updated); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "new" {
		t.Fatalf("replacement = %q", data)
	}
	backup, err := os.ReadFile(target + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "old" {
		t.Fatalf("backup = %q", backup)
	}
}
