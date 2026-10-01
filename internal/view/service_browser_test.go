package view

import (
	"context"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/clawscli/claws/internal/registry"
)

func TestServiceBrowserNavigation(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register some test services
	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})
	reg.RegisterCustom("lambda", "functions", registry.Entry{})
	reg.RegisterCustom("iam", "roles", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)

	// Initialize to load services
	browser.Update(browser.Init()())

	// Check initial state
	if browser.cursor != 0 {
		t.Errorf("Initial cursor = %d, want 0", browser.cursor)
	}

	// Test navigation with 'l' (right)
	browser.Update(tea.KeyPressMsg{Code: 'l'})
	if browser.cursor != 1 {
		t.Errorf("After 'l', cursor = %d, want 1", browser.cursor)
	}

	// Test navigation with 'h' (left)
	browser.Update(tea.KeyPressMsg{Code: 'h'})
	if browser.cursor != 0 {
		t.Errorf("After 'h', cursor = %d, want 0", browser.cursor)
	}
}

func TestServiceBrowserFilter(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register test services
	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})
	reg.RegisterCustom("lambda", "functions", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())

	initialCount := len(browser.flatItems)
	if initialCount == 0 {
		t.Fatal("No services loaded")
	}

	// Activate filter mode
	browser.Update(tea.KeyPressMsg{Text: "/", Code: '/'})
	if !browser.filterActive {
		t.Error("Expected filter to be active after '/'")
	}

	// Type 'ec2' in filter
	for _, r := range "ec2" {
		browser.Update(tea.KeyPressMsg{Text: string(r), Code: r})
	}

	// Should have fewer items after filtering
	if len(browser.flatItems) >= initialCount {
		t.Errorf("Expected fewer items after filter, got %d (was %d)", len(browser.flatItems), initialCount)
	}

	// Press Esc to exit filter mode
	browser.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if browser.filterActive {
		t.Error("Expected filter to be inactive after Esc")
	}

	// Press 'c' to clear filter
	browser.Update(tea.KeyPressMsg{Code: 'c'})
	if len(browser.flatItems) != initialCount {
		t.Errorf("After clear, items = %d, want %d", len(browser.flatItems), initialCount)
	}
}

func TestServiceBrowserHasActiveInput(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	browser := NewServiceBrowser(ctx, reg)

	// Check ServiceBrowser implements InputCapture
	var _ InputCapture = browser

	// Initially no active input
	if browser.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be false initially")
	}

	// Activate filter
	browser.filterActive = true
	if !browser.HasActiveInput() {
		t.Error("Expected HasActiveInput() to be true when filter is active")
	}
}

func TestServiceBrowserCategoryNavigation(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	// Register services in different categories
	reg.RegisterCustom("ec2", "instances", registry.Entry{})    // Compute
	reg.RegisterCustom("lambda", "functions", registry.Entry{}) // Compute
	reg.RegisterCustom("s3", "buckets", registry.Entry{})       // Storage
	reg.RegisterCustom("iam", "roles", registry.Entry{})        // Security

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())

	initialCursor := browser.cursor
	initialCat := -1
	if len(browser.flatItems) > 0 {
		initialCat = browser.flatItems[browser.cursor].categoryIdx
	}

	// Test 'j' moves to next category
	browser.Update(tea.KeyPressMsg{Code: 'j'})

	if len(browser.flatItems) > 1 && browser.cursor > 0 {
		newCat := browser.flatItems[browser.cursor].categoryIdx
		if newCat == initialCat && browser.cursor != initialCursor {
			// If still in same category, cursor should have moved
			t.Log("Moved within category or wrapped")
		}
	}
}

func TestServiceBrowserMouseHover(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())
	browser.SetSize(100, 50)

	initialCursor := browser.cursor

	// Simulate mouse motion - exact position depends on layout
	// Just verify it doesn't crash and cursor can change
	motionMsg := tea.MouseMotionMsg{X: 30, Y: 5}
	browser.Update(motionMsg)

	// Cursor may or may not change depending on position
	// Main test is that it doesn't panic
	t.Logf("Cursor after hover: %d (was %d)", browser.cursor, initialCursor)
}

func TestServiceBrowserMouseClick(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())
	browser.SetSize(100, 50)

	// Simulate mouse click
	clickMsg := tea.MouseClickMsg{X: 30, Y: 5, Button: tea.MouseLeft}
	_, cmd := browser.Update(clickMsg)

	// Click might trigger navigation or do nothing depending on position
	t.Logf("Command after click: %v", cmd)
}

func TestServiceBrowserMouseWheel(t *testing.T) {
	ctx := context.Background()
	reg := registry.New()

	reg.RegisterCustom("ec2", "instances", registry.Entry{})
	reg.RegisterCustom("s3", "buckets", registry.Entry{})

	browser := NewServiceBrowser(ctx, reg)
	browser.Update(browser.Init()())
	browser.SetSize(100, 50)

	// Simulate mouse wheel
	wheelMsg := tea.MouseWheelMsg{X: 30, Y: 5, Button: tea.MouseWheelDown}
	browser.Update(wheelMsg)

	// Should not panic
}

func TestServiceBrowserScrollFollowsCursor(t *testing.T) {
	reg := registry.New()
	for _, svc := range []string{"ec2", "s3", "ecr", "glue", "vpc", "iam", "sqs", "codebuild", "cloudwatch", "backup", "ce", "gamelift"} {
		reg.RegisterCustom(svc, "items", registry.Entry{})
	}

	browser := NewServiceBrowser(context.Background(), reg)
	browser.Update(browser.Init()())
	browser.SetSize(100, 12)
	browser.updateViewport()

	cursorVisible := func() bool {
		top := browser.vp.Model.YOffset()
		bottom := top + browser.vp.Model.Height()

		for _, pos := range browser.itemPositions {
			if pos.itemIdx == browser.cursor {
				return pos.startLine >= top && pos.endLine <= bottom
			}
		}

		return false
	}

	for browser.cursor < len(browser.flatItems)-1 {
		browser.Update(tea.KeyPressMsg{Code: 'j'})

		if !cursorVisible() {
			t.Fatalf("cursor %d not visible after 'j' (YOffset=%d)", browser.cursor, browser.vp.Model.YOffset())
		}
	}

	if browser.vp.Model.YOffset() == 0 {
		t.Fatal("viewport did not scroll")
	}

	for browser.cursor > 0 {
		browser.Update(tea.KeyPressMsg{Code: 'k'})

		if !cursorVisible() {
			t.Fatalf("cursor %d not visible after 'k' (YOffset=%d)", browser.cursor, browser.vp.Model.YOffset())
		}
	}

	if got := browser.vp.Model.YOffset(); got != 0 {
		t.Errorf("YOffset at first item = %d, want 0", got)
	}
}
