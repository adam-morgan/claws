package view

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/action"
	"github.com/clawscli/claws/internal/dao"
)

func TestActionMenuMouseHover(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-123", name: "test"}

	menu := NewActionMenu(ctx, resource, "ec2", "instances")

	initialCursor := menu.cursor

	// Simulate mouse motion
	motionMsg := tea.MouseMotionMsg{X: 10, Y: 5}
	menu.Update(motionMsg)

	t.Logf("Cursor after hover: %d (was %d)", menu.cursor, initialCursor)
}

func TestActionMenuConfirmDangerousCorrectToken(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Manually set up dangerous confirm state (normally triggered by action selection)
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345" // Default: uses GetID()
	menu.dangerous.input = ""

	// Type the full confirmation token.
	confirmText := action.ConfirmSuffix("i-12345")
	for _, r := range confirmText {
		msg := tea.KeyPressMsg{Text: string(r), Code: r}
		menu.Update(msg)
	}

	if menu.dangerous.input != confirmText {
		t.Errorf("dangerousInput = %q, want %q", menu.dangerous.input, confirmText)
	}

	// Press enter - should accept since input matches the full token.
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	menu.Update(enterMsg)

	// Confirm state should be cleared on successful match
	if menu.dangerous.active {
		t.Error("Expected dangerousConfirm to be false after correct token + enter")
	}
	if menu.dangerous.input != "" {
		t.Errorf("Expected dangerousInput to be cleared, got %q", menu.dangerous.input)
	}
	if menu.dangerous.token != "" {
		t.Errorf("Expected confirmToken to be cleared, got %q", menu.dangerous.token)
	}
}

func TestActionMenuConfirmDangerousSuffixOnlyRejected(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-1234567890abcdef0", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-1234567890abcdef0"
	menu.dangerous.input = "bcdef0"

	menu.Update(tea.KeyPressMsg{Code: tea.KeyEnter})

	if !menu.dangerous.active {
		t.Error("Expected dangerousConfirm to remain true after suffix-only token + enter")
	}
	if menu.dangerous.input != "bcdef0" {
		t.Errorf("Expected dangerousInput to remain %q, got %q", "bcdef0", menu.dangerous.input)
	}
}

func TestActionMenuConfirmDangerousWrongToken(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = ""

	// Type wrong token
	for _, r := range "wrong" {
		msg := tea.KeyPressMsg{Text: string(r), Code: r}
		menu.Update(msg)
	}

	if menu.dangerous.input != "wrong" {
		t.Errorf("dangerousInput = %q, want %q", menu.dangerous.input, "wrong")
	}

	// Press enter - should NOT accept since input doesn't match token
	enterMsg := tea.KeyPressMsg{Code: tea.KeyEnter}
	menu.Update(enterMsg)

	// Confirm state should remain (not cleared)
	if !menu.dangerous.active {
		t.Error("Expected dangerousConfirm to remain true after wrong token + enter")
	}
	if menu.dangerous.input != "wrong" {
		t.Errorf("Expected dangerousInput to remain %q, got %q", "wrong", menu.dangerous.input)
	}
}

func TestActionMenuConfirmDangerousEscCancels(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state with partial input
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = "i-123"

	// Press esc - should cancel
	escMsg := tea.KeyPressMsg{Code: tea.KeyEscape}
	menu.Update(escMsg)

	// Confirm state should be cleared
	if menu.dangerous.active {
		t.Error("Expected dangerousConfirm to be false after esc")
	}
	if menu.dangerous.input != "" {
		t.Errorf("Expected dangerousInput to be cleared, got %q", menu.dangerous.input)
	}
	if menu.dangerous.token != "" {
		t.Errorf("Expected confirmToken to be cleared, got %q", menu.dangerous.token)
	}
}

func TestActionMenuConfirmDangerousBackspaceString(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state with input
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = "i-123"

	// Test backspace via msg.String() == "backspace"
	// This handles terminals that send backspace as a string
	backspaceMsg := tea.KeyPressMsg{Text: "backspace"}
	menu.Update(backspaceMsg)

	if menu.dangerous.input != "i-12" {
		t.Errorf("After string backspace: dangerousInput = %q, want %q", menu.dangerous.input, "i-12")
	}
}

func TestActionMenuConfirmDangerousBackspaceKeyCode(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state with input
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = "i-123"

	// Test backspace via msg.Code == tea.KeyBackspace
	// This handles terminals that send backspace as a key code
	backspaceMsg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	menu.Update(backspaceMsg)

	if menu.dangerous.input != "i-12" {
		t.Errorf("After keycode backspace: dangerousInput = %q, want %q", menu.dangerous.input, "i-12")
	}
}

func TestActionMenuConfirmDangerousBackspaceEmpty(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Set up dangerous confirm state with empty input
	menu.dangerous.active = true
	menu.confirmIdx = 0
	menu.dangerous.token = "i-12345"
	menu.dangerous.input = ""

	// Backspace on empty input should be safe (not panic)
	backspaceMsg := tea.KeyPressMsg{Code: tea.KeyBackspace}
	menu.Update(backspaceMsg)

	if menu.dangerous.input != "" {
		t.Errorf("After backspace on empty: dangerousInput = %q, want empty", menu.dangerous.input)
	}

	// Also test string backspace on empty
	backspaceStrMsg := tea.KeyPressMsg{Text: "backspace"}
	menu.Update(backspaceStrMsg)

	if menu.dangerous.input != "" {
		t.Errorf("After string backspace on empty: dangerousInput = %q, want empty", menu.dangerous.input)
	}
}

func TestActionMenuConfirmDangerousHasActiveInput(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")

	// Initially no active input
	if menu.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be false initially")
	}

	// Enter dangerous confirm mode
	menu.dangerous.active = true

	// Now should have active input
	if !menu.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be true when dangerousConfirm is active")
	}
}

func TestActionMenuDangerousStatusLineFullToken(t *testing.T) {
	ctx := context.Background()
	resource := &mockResource{id: "i-12345", name: "test-instance"}

	menu := NewActionMenu(ctx, resource, "test", "items")
	menu.dangerous.active = true
	menu.dangerous.token = resource.GetID()

	if got, want := menu.StatusLine(), "Type full confirmation token"; got != want {
		t.Errorf("StatusLine() = %q, want %q", got, want)
	}
}

func TestActionMenuAPIActionRunsAsyncAndIgnoresKeysWhileRunning(t *testing.T) {
	executions := 0
	action.Global.Register("asynctest", "items", []action.Action{
		{Name: "Invoke", Shortcut: "i", Type: action.ActionTypeAPI, Operation: "Invoke", Confirm: action.ConfirmSimple},
	})
	action.RegisterExecutor("asynctest", "items", func(ctx context.Context, act action.Action, r dao.Resource) action.ActionResult {
		executions++
		return action.SuccessResult("done")
	})

	menu := NewActionMenu(context.Background(), &mockResource{id: "fn", name: "fn"}, "asynctest", "items")

	menu.Update(tea.KeyPressMsg{Text: "i", Code: 'i'})
	_, cmd := menu.Update(tea.KeyPressMsg{Text: "Y", Code: 'Y'})

	if !menu.running {
		t.Fatal("expected menu to be running after confirm")
	}

	if cmd == nil {
		t.Fatal("expected a command to execute the action")
	}

	if executions != 0 {
		t.Fatalf("action executed synchronously in Update, executions = %d", executions)
	}

	for _, key := range []string{"Y", "i", "Y"} {
		if _, extra := menu.Update(tea.KeyPressMsg{Text: key, Code: rune(key[0])}); extra != nil {
			t.Fatalf("key %q produced a command while running", key)
		}
	}

	menu.Update(cmd())

	if executions != 1 {
		t.Errorf("executions = %d, want 1", executions)
	}

	if menu.running || menu.result == nil || !menu.result.Success {
		t.Errorf("expected successful result after completion, running=%v result=%+v", menu.running, menu.result)
	}
}
