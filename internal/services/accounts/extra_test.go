package accounts

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

func TestWatchFileChange(t *testing.T) {
	svc, accountsPath := newTestService(t)
	// svc.Close() handled by cleanup

	// Wait for initial load event
	<-svc.Events()

	// Write new content to file directly
	newContent := []byte(`{
		"accounts": [
			{"email": "watched@example.com", "id": "watched-1"}
		],
		"activeAccount": "watched-1",
		"version": 1
	}`)

	if err := os.WriteFile(accountsPath, newContent, 0600); err != nil {
		t.Fatalf("WriteFile() failed: %v", err)
	}

	// Wait for change event
	timeout := time.After(2 * time.Second)
	found := false
	for {
		select {
		case event := <-svc.Events():
			if event.Type == EventAccountsChanged {
				found = true
				goto Done
			}
		case <-timeout:
			t.Fatal("timeout waiting for EventAccountsChanged")
		}
	}
Done:
	if !found {
		t.Error("EventAccountsChanged not received")
	}

	// Verify accounts reloaded
	accounts := svc.GetAccounts()
	if len(accounts) != 1 {
		t.Errorf("expected 1 account after reload, got %d", len(accounts))
	}
	if accounts[0].Email != "watched@example.com" {
		t.Errorf("expected email watched@example.com, got %s", accounts[0].Email)
	}
}

func TestLoadJSDashboardFormat(t *testing.T) {
	tmpDir := t.TempDir()
	accountsPath := filepath.Join(tmpDir, "accounts_js.json")

	content := []byte(`{
		"accounts": [
			{
				"email": "js@example.com", 
				"projectId": "js-proj", 
				"rateLimitResetTimes": {"claude": 123}
			}
		],
		"activeIndex": 0,
		"version": 1
	}`)

	if err := os.WriteFile(accountsPath, content, 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	svc, err := New(accountsPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer svc.Close()

	accounts := svc.GetAccounts()
	if len(accounts) != 1 {
		t.Errorf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].Email != "js@example.com" {
		t.Errorf("expected email js@example.com, got %s", accounts[0].Email)
	}
	if accounts[0].ID != "js-proj" {
		t.Errorf("expected ID js-proj, got %s", accounts[0].ID)
	}
}

func TestLoadLegacyFormat(t *testing.T) {
	tmpDir := t.TempDir()
	accountsPath := filepath.Join(tmpDir, "accounts_legacy.json")

	content := []byte(`[
		{"email": "legacy@example.com", "projectId": "legacy-proj"}
	]`)

	if err := os.WriteFile(accountsPath, content, 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	svc, err := New(accountsPath)
	if err != nil {
		t.Fatalf("New() failed: %v", err)
	}
	defer svc.Close()

	accounts := svc.GetAccounts()
	if len(accounts) != 1 {
		t.Errorf("expected 1 account, got %d", len(accounts))
	}
	if accounts[0].Email != "legacy@example.com" {
		t.Errorf("expected email legacy@example.com, got %s", accounts[0].Email)
	}
}

func TestSendEvent_Full(t *testing.T) {
	svc, _ := newTestService(t)
	// svc.Close() handled by cleanup

	// Fill channel
	for i := 0; i < 110; i++ {
		svc.sendEvent(Event{Type: EventAccountsChanged})
	}

	if len(svc.Events()) != 100 {
		t.Errorf("expected 100 events, got %d", len(svc.Events()))
	}
}

func TestDefaultAccountsPath(t *testing.T) {
	path := defaultAccountsPath()
	if path == "" {
		t.Error("defaultAccountsPath returned empty string")
	}
}

func TestSaveAccounts_Public(t *testing.T) {
	svc, _ := newTestService(t)
	// svc.Close() handled by cleanup

	err := svc.saveAccounts() // Calls saveAccountsLocked
	if err != nil {
		t.Errorf("saveAccounts() failed: %v", err)
	}
}

func TestUpdateAccountEmail_Success(t *testing.T) {
	svc, _ := newTestService(t)
	// svc.Close() handled by cleanup

	svc.AddAccount(&models.Account{Email: "old@test.com", ID: "1"})

	err := svc.UpdateAccountEmail("old@test.com", "new@test.com")
	if err != nil {
		t.Errorf("UpdateAccountEmail failed: %v", err)
	}

	if svc.GetAccountByEmail("new@test.com") == nil {
		t.Error("new email not found")
	}
	if svc.GetAccountByEmail("old@test.com") != nil {
		t.Error("old email still found")
	}
}

func TestUpdateAccountEmail_NotFound(t *testing.T) {
	svc, _ := newTestService(t)
	// svc.Close() handled by cleanup

	err := svc.UpdateAccountEmail("missing@test.com", "new@test.com")
	if err == nil {
		t.Error("UpdateAccountEmail should fail for missing account")
	}
}

func TestHandleFileChange_Error(t *testing.T) {
	svc, accountsPath := newTestService(t)
	// svc.Close() handled by cleanup

	<-svc.Events()

	// Write invalid JSON
	if err := os.WriteFile(accountsPath, []byte("{invalid"), 0600); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	timeout := time.After(2 * time.Second)
	found := false
	for {
		select {
		case event := <-svc.Events():
			if event.Type == EventError {
				found = true
				goto Done
			}
		case <-timeout:
			t.Fatal("timeout waiting for EventError")
		}
	}
Done:
	if !found {
		t.Error("EventError not received")
	}
}

func TestParseAccounts_ActiveNotFound(t *testing.T) {
	svc, _ := newTestService(t)

	// Active account "missing" is not in list. Should default to first.
	data := []byte(`{
		"accounts": [
			{"email": "test@example.com", "id": "acc-1"}
		],
		"activeAccount": "missing"
	}`)

	accounts, active, err := svc.parseAccounts(data)
	if err != nil {
		t.Fatalf("parseAccounts failed: %v", err)
	}

	if len(accounts) != 1 {
		t.Errorf("got %d accounts", len(accounts))
	}
	// "acc-1" is ID of first account
	if active != "acc-1" {
		t.Errorf("expected active account 'acc-1', got '%s'", active)
	}
}

func TestParseAccounts_NoActiveSet(t *testing.T) {
	svc, _ := newTestService(t)

	// No activeAccount field. Should default to first.
	data := []byte(`{
		"accounts": [
			{"email": "test@example.com", "id": "acc-1"}
		]
	}`)

	_, active, err := svc.parseAccounts(data)
	if err != nil {
		t.Fatalf("parseAccounts failed: %v", err)
	}

	if active != "acc-1" {
		t.Errorf("expected active account 'acc-1', got '%s'", active)
	}
}
