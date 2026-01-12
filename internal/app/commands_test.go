package app

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestCommands_Tick(t *testing.T) {
	cmds := NewCommands(nil)
	cmd := cmds.Tick(time.Millisecond)
	if cmd == nil {
		t.Error("Tick returned nil")
	}
}

func TestCommands_DefaultTick(t *testing.T) {
	cmds := NewCommands(nil)
	cmd := cmds.DefaultTick()
	if cmd == nil {
		t.Error("DefaultTick returned nil")
	}
}

func TestCommands_Notifications(t *testing.T) {
	cmds := NewCommands(nil)

	tests := []struct {
		name string
		fn   func(string) tea.Cmd
		want NotificationType
	}{
		{"Success", cmds.NotifySuccess, NotificationSuccess},
		{"Error", cmds.NotifyError, NotificationError},
		{"Warning", cmds.NotifyWarning, NotificationWarning},
		{"Info", cmds.NotifyInfo, NotificationInfo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := tt.fn("msg")
			msg := cmd()

			addMsg, ok := msg.(AddNotificationMsg)
			if !ok {
				t.Fatalf("Expected AddNotificationMsg, got %T", msg)
			}
			if addMsg.Type != tt.want {
				t.Errorf("Type = %v, want %v", addMsg.Type, tt.want)
			}
			if addMsg.Message != "msg" {
				t.Errorf("Message = %q, want msg", addMsg.Message)
			}
		})
	}
}

func TestCommands_ClearNotification(t *testing.T) {
	cmds := NewCommands(nil)
	// Mock time.Tick or just check it returns a command
	cmd := cmds.ClearNotification("id", time.Millisecond)
	if cmd == nil {
		t.Error("ClearNotification returned nil")
	}
}

func TestCommands_Quit(t *testing.T) {
	cmds := NewCommands(nil)
	cmd := cmds.Quit()
	msg := cmd()
	if _, ok := msg.(tea.QuitMsg); !ok {
		t.Errorf("Expected QuitMsg, got %T", msg)
	}
}

func TestCommands_Batch(t *testing.T) {
	cmds := NewCommands(nil)
	cmd := cmds.Batch(cmds.Quit(), cmds.NotifyInfo("test"))
	if cmd == nil {
		t.Error("Batch returned nil")
	}
	// Executing batch command returns BatchMsg usually, but here likely nil or specific tea implementation
	// We just check it's not nil
}

func TestCommands_Delayed(t *testing.T) {
	cmds := NewCommands(nil)
	cmd := cmds.Delayed(time.Millisecond, QuitMsg{})
	if cmd == nil {
		t.Error("Delayed returned nil")
	}
}
