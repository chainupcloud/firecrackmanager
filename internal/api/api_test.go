package api

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureSSHRootPasswordLoginCreatesEarlyDropIn(t *testing.T) {
	mountPoint := t.TempDir()
	sshDir := filepath.Join(mountPoint, "etc", "ssh")
	dropInDir := filepath.Join(sshDir, "sshd_config.d")
	if err := os.MkdirAll(dropInDir, 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	mainConfig := "# comment\n\n" + sshdDropInInclude + "\nPasswordAuthentication no\n"
	if err := os.WriteFile(filepath.Join(sshDir, "sshd_config"), []byte(mainConfig), 0644); err != nil {
		t.Fatalf("WriteFile(sshd_config) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dropInDir, "60-cloudimg-settings.conf"), []byte("PasswordAuthentication no\n"), 0644); err != nil {
		t.Fatalf("WriteFile(60-cloudimg-settings.conf) error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dropInDir, "99-chainup-dev-key.conf"), []byte("PermitRootLogin prohibit-password\nPasswordAuthentication no\n"), 0644); err != nil {
		t.Fatalf("WriteFile(99-chainup-dev-key.conf) error = %v", err)
	}

	if err := ensureSSHRootPasswordLogin(mountPoint); err != nil {
		t.Fatalf("ensureSSHRootPasswordLogin() error = %v", err)
	}

	dropInPath := filepath.Join(dropInDir, sshdRootPasswordDropIn)
	data, err := os.ReadFile(dropInPath)
	if err != nil {
		t.Fatalf("ReadFile(%s) error = %v", sshdRootPasswordDropIn, err)
	}
	content := string(data)
	for _, want := range []string{
		"PermitRootLogin yes",
		"PasswordAuthentication yes",
		"PubkeyAuthentication yes",
		"AuthorizedKeysFile .ssh/authorized_keys",
		"KbdInteractiveAuthentication yes",
		"UsePAM yes",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("drop-in missing %q: %s", want, content)
		}
	}

	entries, err := os.ReadDir(dropInDir)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}
	if len(entries) < 3 || entries[0].Name() != sshdRootPasswordDropIn {
		t.Fatalf("managed drop-in must sort before image drop-ins, entries=%v", entryNames(entries))
	}

	updatedMain, err := os.ReadFile(filepath.Join(sshDir, "sshd_config"))
	if err != nil {
		t.Fatalf("ReadFile(sshd_config) error = %v", err)
	}
	if strings.Count(string(updatedMain), sshdDropInInclude) != 1 {
		t.Fatalf("sshd_config include should remain idempotent: %s", string(updatedMain))
	}
}

func TestEnsureSSHDConfigIncludesDropInsFirst(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "sshd_config")
	original := "PasswordAuthentication no\n" + sshdDropInInclude + "\n"
	if err := os.WriteFile(configPath, []byte(original), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := ensureSSHDConfigIncludesDropInsFirst(configPath); err != nil {
		t.Fatalf("ensureSSHDConfigIncludesDropInsFirst() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content := string(data)
	if !strings.HasPrefix(content, sshdDropInInclude+"\n") {
		t.Fatalf("sshd_config should include drop-ins before explicit directives: %s", content)
	}
}

func TestEnsureSSHRootPasswordLoginNoSSHInstalled(t *testing.T) {
	if err := ensureSSHRootPasswordLogin(t.TempDir()); err != nil {
		t.Fatalf("ensureSSHRootPasswordLogin() error = %v", err)
	}
}

func TestEnsureRootAuthorizedKeyCreatesIdempotentAuthorizedKeys(t *testing.T) {
	mountPoint := t.TempDir()

	if err := ensureRootAuthorizedKey(mountPoint, firecrackmanagerRootKey); err != nil {
		t.Fatalf("ensureRootAuthorizedKey() error = %v", err)
	}
	if err := ensureRootAuthorizedKey(mountPoint, firecrackmanagerRootKey); err != nil {
		t.Fatalf("ensureRootAuthorizedKey() second call error = %v", err)
	}

	authorizedKeysPath := filepath.Join(mountPoint, "root", ".ssh", "authorized_keys")
	data, err := os.ReadFile(authorizedKeysPath)
	if err != nil {
		t.Fatalf("ReadFile(authorized_keys) error = %v", err)
	}
	content := string(data)
	if strings.Count(content, firecrackmanagerRootKey) != 1 {
		t.Fatalf("authorized_keys should contain key exactly once: %q", content)
	}

	info, err := os.Stat(authorizedKeysPath)
	if err != nil {
		t.Fatalf("Stat(authorized_keys) error = %v", err)
	}
	if got := info.Mode().Perm(); got != 0600 {
		t.Fatalf("authorized_keys permissions = %v, want 0600", got)
	}

	sshDirInfo, err := os.Stat(filepath.Join(mountPoint, "root", ".ssh"))
	if err != nil {
		t.Fatalf("Stat(.ssh) error = %v", err)
	}
	if got := sshDirInfo.Mode().Perm(); got != 0700 {
		t.Fatalf(".ssh permissions = %v, want 0700", got)
	}
}

func TestNormalizeVMIPAddressStripsCIDR(t *testing.T) {
	tests := map[string]string{
		"192.168.100.2/24":    "192.168.100.2",
		" 192.168.100.3/24 ":  "192.168.100.3",
		"192.168.100.4":       "192.168.100.4",
		"":                    "",
	}
	for input, want := range tests {
		if got := normalizeVMIPAddress(input); got != want {
			t.Fatalf("normalizeVMIPAddress(%q) = %q, want %q", input, got, want)
		}
	}
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}
