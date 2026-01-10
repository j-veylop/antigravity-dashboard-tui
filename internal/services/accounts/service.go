// Package accounts provides account management with file watching and persistence.
package accounts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/j-veylop/antigravity-dashboard-tui/internal/logger"
	"github.com/j-veylop/antigravity-dashboard-tui/internal/models"
)

// AccountsFile represents the JSON file structure for accounts storage.
type AccountsFile struct {
	Accounts      []models.Account `json:"accounts"`
	ActiveAccount string           `json:"activeAccount,omitempty"`
	Version       int              `json:"version,omitempty"`
}

// Event represents an account service event.
type Event struct {
	Type    EventType
	Error   error
	Account *models.Account
}

// EventType defines the type of account event.
type EventType int

const (
	EventAccountsLoaded EventType = iota
	EventAccountsChanged
	EventAccountAdded
	EventAccountUpdated
	EventAccountDeleted
	EventActiveAccountChanged
	EventError
)

// Service manages accounts with file watching and change notifications.
type Service struct {
	mu            sync.RWMutex
	accounts      []models.Account
	activeAccount string
	filePath      string
	watcher       *fsnotify.Watcher
	onChange      func()
	eventChan     chan Event
	stopChan      chan struct{}
	debounceTimer *time.Timer
}

// defaultAccountsPath returns the default accounts file path.
func defaultAccountsPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "opencode", "antigravity-accounts.json")
}

// New creates a new accounts service and starts file watching.
func New(filePath string) (*Service, error) {
	if filePath == "" {
		filePath = defaultAccountsPath()
	}

	s := &Service{
		accounts:  make([]models.Account, 0),
		filePath:  filePath,
		eventChan: make(chan Event, 100),
		stopChan:  make(chan struct{}),
	}

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	// Load accounts from file
	if err := s.loadAccounts(); err != nil {
		// If file doesn't exist, create empty accounts file
		if os.IsNotExist(err) {
			if err := s.saveAccounts(); err != nil {
				return nil, fmt.Errorf("failed to create accounts file: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load accounts: %w", err)
		}
	}

	// Start file watcher
	if err := s.startWatcher(); err != nil {
		return nil, fmt.Errorf("failed to start file watcher: %w", err)
	}

	s.sendEvent(Event{Type: EventAccountsLoaded})

	return s, nil
}

// Events returns the event channel for subscribing to account changes.
func (s *Service) Events() <-chan Event {
	return s.eventChan
}

// GetAccounts returns a copy of all accounts.
func (s *Service) GetAccounts() []models.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	accounts := make([]models.Account, len(s.accounts))
	for i, acc := range s.accounts {
		accounts[i] = acc

		if acc.RateLimitResetTimes != nil {
			accounts[i].RateLimitResetTimes = make(map[string]int64, len(acc.RateLimitResetTimes))
			for k, v := range acc.RateLimitResetTimes {
				accounts[i].RateLimitResetTimes[k] = v
			}
		}
	}
	return accounts
}

// GetActiveAccount returns the currently active account.
func (s *Service) GetActiveAccount() *models.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.accounts {
		if s.accounts[i].ID == s.activeAccount || s.accounts[i].Email == s.activeAccount {
			acc := s.accounts[i]
			return &acc
		}
	}

	// Return first account if no active account set
	if len(s.accounts) > 0 {
		acc := s.accounts[0]
		return &acc
	}

	return nil
}

// GetActiveAccountID returns the ID of the active account.
func (s *Service) GetActiveAccountID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.activeAccount
}

// SetActiveAccount sets the active account by ID or email.
func (s *Service) SetActiveAccount(idOrEmail string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Verify account exists
	found := false
	for _, acc := range s.accounts {
		if acc.ID == idOrEmail || acc.Email == idOrEmail {
			found = true
			s.activeAccount = acc.ID
			if s.activeAccount == "" {
				s.activeAccount = acc.Email
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("account not found: %s", idOrEmail)
	}

	if err := s.saveAccountsLocked(); err != nil {
		return fmt.Errorf("failed to save accounts: %w", err)
	}

	s.sendEvent(Event{Type: EventActiveAccountChanged})
	return nil
}

// AddAccount adds a new account.
func (s *Service) AddAccount(account models.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check for duplicate
	for _, acc := range s.accounts {
		if acc.Email == account.Email {
			return fmt.Errorf("account with email %s already exists", account.Email)
		}
	}

	// Set defaults
	if account.ID == "" {
		account.ID = fmt.Sprintf("acc_%d", time.Now().UnixNano())
	}
	if account.AddedAt.IsZero() {
		account.AddedAt = time.Now()
	}

	s.accounts = append(s.accounts, account)

	// Set as active if first account
	if len(s.accounts) == 1 {
		s.activeAccount = account.ID
	}

	if err := s.saveAccountsLocked(); err != nil {
		// Rollback
		s.accounts = s.accounts[:len(s.accounts)-1]
		return fmt.Errorf("failed to save accounts: %w", err)
	}

	s.sendEvent(Event{Type: EventAccountAdded, Account: &account})
	return nil
}

// UpdateAccount updates an existing account.
func (s *Service) UpdateAccount(account models.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	found := false
	for i, acc := range s.accounts {
		if acc.ID == account.ID || acc.Email == account.Email {
			// Preserve ID if updating by email
			if account.ID == "" {
				account.ID = acc.ID
			}
			// Preserve AddedAt
			if account.AddedAt.IsZero() {
				account.AddedAt = acc.AddedAt
			}
			s.accounts[i] = account
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("account not found: %s", account.Email)
	}

	if err := s.saveAccountsLocked(); err != nil {
		return fmt.Errorf("failed to save accounts: %w", err)
	}

	s.sendEvent(Event{Type: EventAccountUpdated, Account: &account})
	return nil
}

// DeleteAccount removes an account by ID or email.
func (s *Service) DeleteAccount(idOrEmail string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := -1
	var deleted models.Account
	for i, acc := range s.accounts {
		if acc.ID == idOrEmail || acc.Email == idOrEmail {
			idx = i
			deleted = acc
			break
		}
	}

	if idx == -1 {
		return fmt.Errorf("account not found: %s", idOrEmail)
	}

	s.accounts = append(s.accounts[:idx], s.accounts[idx+1:]...)

	// Update active account if deleted
	if s.activeAccount == deleted.ID || s.activeAccount == deleted.Email {
		if len(s.accounts) > 0 {
			s.activeAccount = s.accounts[0].ID
			if s.activeAccount == "" {
				s.activeAccount = s.accounts[0].Email
			}
		} else {
			s.activeAccount = ""
		}
	}

	if err := s.saveAccountsLocked(); err != nil {
		return fmt.Errorf("failed to save accounts: %w", err)
	}

	s.sendEvent(Event{Type: EventAccountDeleted, Account: &deleted})
	return nil
}

// GetAccountByEmail returns an account by email address.
func (s *Service) GetAccountByEmail(email string) *models.Account {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for i := range s.accounts {
		if s.accounts[i].Email == email {
			acc := s.accounts[i]
			return &acc
		}
	}
	return nil
}

// Count returns the number of accounts.
func (s *Service) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.accounts)
}

// UpdateAccountEmail updates the email address of an account (implements quota.AccountProvider).
func (s *Service) UpdateAccountEmail(oldEmail, newEmail string) error {
	acc := s.GetAccountByEmail(oldEmail)
	if acc == nil {
		return fmt.Errorf("account not found: %s", oldEmail)
	}

	newAcc := *acc
	newAcc.Email = newEmail

	return s.UpdateAccount(newAcc)
}

// parseAccounts parses account data handling multiple formats.
func (s *Service) parseAccounts(data []byte) ([]models.Account, string, error) {
	// 1. Try format from JS dashboard (via models package)
	var rawFile struct {
		Version     int                     `json:"version"`
		Accounts    []models.RawAccountData `json:"accounts"`
		ActiveIndex int                     `json:"activeIndex"`
	}

	if err := json.Unmarshal(data, &rawFile); err == nil {
		accounts := make([]models.Account, len(rawFile.Accounts))
		for i, raw := range rawFile.Accounts {
			modelAcc := raw.ToAccount()

			id := modelAcc.ProjectID
			email := modelAcc.Email
			if email == "" {
				email = modelAcc.ProjectID
			}

			accounts[i] = modelAcc
			accounts[i].ID = id
			accounts[i].IsActive = true
		}

		var activeAccount string
		if rawFile.ActiveIndex >= 0 && rawFile.ActiveIndex < len(accounts) {
			activeAccount = accounts[rawFile.ActiveIndex].ID
		} else if len(accounts) > 0 {
			activeAccount = accounts[0].ID
		}

		return accounts, activeAccount, nil
	}

	// 2. Try standard AccountsFile format
	var accountsFile AccountsFile
	if err := json.Unmarshal(data, &accountsFile); err == nil {
		activeAccount := accountsFile.ActiveAccount

		if activeAccount != "" {
			found := false
			for _, acc := range accountsFile.Accounts {
				if acc.ID == activeAccount || acc.Email == activeAccount {
					found = true
					break
				}
			}
			if !found && len(accountsFile.Accounts) > 0 {
				activeAccount = accountsFile.Accounts[0].ID
				if activeAccount == "" {
					activeAccount = accountsFile.Accounts[0].Email
				}
			}
		} else if len(accountsFile.Accounts) > 0 {
			activeAccount = accountsFile.Accounts[0].ID
			if activeAccount == "" {
				activeAccount = accountsFile.Accounts[0].Email
			}
		}
		return accountsFile.Accounts, activeAccount, nil
	}

	// 3. Try legacy array format
	var accounts []models.Account
	if err := json.Unmarshal(data, &accounts); err == nil {
		var activeAccount string
		if len(accounts) > 0 {
			activeAccount = accounts[0].Email
		}

		for i := range accounts {
			if accounts[i].RateLimitResetTimes == nil {
				accounts[i].RateLimitResetTimes = make(map[string]int64)
			}
		}
		return accounts, activeAccount, nil
	}

	return nil, "", fmt.Errorf("failed to parse accounts file: invalid format")
}

// loadAccounts loads accounts from the JSON file.
func (s *Service) loadAccounts() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	accounts, activeAccount, err := s.parseAccounts(data)
	if err != nil {
		return err
	}

	s.accounts = accounts
	s.activeAccount = activeAccount
	return nil
}

// saveAccounts saves accounts to the JSON file (public version).
func (s *Service) saveAccounts() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveAccountsLocked()
}

// saveAccountsLocked saves accounts to the JSON file (must hold lock).
func (s *Service) saveAccountsLocked() error {
	accountsFile := AccountsFile{
		Accounts:      s.accounts,
		ActiveAccount: s.activeAccount,
		Version:       1,
	}

	data, err := json.MarshalIndent(accountsFile, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal accounts: %w", err)
	}

	// Write to temp file first, then rename
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmpFile, s.filePath); err != nil {
		if removeErr := os.Remove(tmpFile); removeErr != nil {
			logger.Error("failed to remove temp file", "error", removeErr)
		}
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// startWatcher starts the file system watcher.
func (s *Service) startWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	s.watcher = watcher

	// Watch the directory (to catch file creation/deletion)
	dir := filepath.Dir(s.filePath)
	if err := watcher.Add(dir); err != nil {
		if closeErr := watcher.Close(); closeErr != nil {
			logger.Error("failed to close watcher", "error", closeErr)
		}
		return err
	}

	go s.watchLoop()
	return nil
}

// watchLoop handles file system events with debouncing.
func (s *Service) watchLoop() {
	const debounceInterval = 100 * time.Millisecond

	for {
		select {
		case event, ok := <-s.watcher.Events:
			if !ok {
				return
			}

			// Only care about our accounts file
			if filepath.Base(event.Name) != filepath.Base(s.filePath) {
				continue
			}

			// Handle write/create events
			if event.Op&(fsnotify.Write|fsnotify.Create) != 0 {
				// Debounce rapid changes
				if s.debounceTimer != nil {
					s.debounceTimer.Stop()
				}
				s.debounceTimer = time.AfterFunc(debounceInterval, func() {
					s.handleFileChange()
				})
			}

		case err, ok := <-s.watcher.Errors:
			if !ok {
				return
			}
			s.sendEvent(Event{Type: EventError, Error: err})

		case <-s.stopChan:
			return
		}
	}
}

// handleFileChange reloads accounts from file after external change.
func (s *Service) handleFileChange() {
	s.mu.Lock()
	oldAccounts := make([]models.Account, len(s.accounts))
	copy(oldAccounts, s.accounts)
	s.mu.Unlock()

	if err := s.loadAccountsWithLock(); err != nil {
		s.sendEvent(Event{Type: EventError, Error: err})
		return
	}

	s.sendEvent(Event{Type: EventAccountsChanged})

	s.mu.RLock()
	onChange := s.onChange
	s.mu.RUnlock()

	if onChange != nil {
		onChange()
	}
}

// loadAccountsWithLock loads accounts while holding the lock.
func (s *Service) loadAccountsWithLock() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	accounts, activeAccount, err := s.parseAccounts(data)
	if err != nil {
		return err
	}

	s.accounts = accounts
	s.activeAccount = activeAccount
	return nil
}

// sendEvent sends an event to the event channel non-blocking.
func (s *Service) sendEvent(event Event) {
	select {
	case s.eventChan <- event:
	default:
		// Channel full, drop oldest event
		select {
		case <-s.eventChan:
		default:
		}
		select {
		case s.eventChan <- event:
		default:
		}
	}
}

// Close stops the file watcher and cleans up resources.
func (s *Service) Close() error {
	close(s.stopChan)

	if s.debounceTimer != nil {
		s.debounceTimer.Stop()
	}

	if s.watcher != nil {
		return s.watcher.Close()
	}
	return nil
}
