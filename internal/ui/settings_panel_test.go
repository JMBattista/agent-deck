package ui

import (
	"reflect"
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
	tea "github.com/charmbracelet/bubbletea"
)

func setSettingsPanelHotkeyConfigForTest(t *testing.T, tomlBody string) {
	t.Helper()

	homeDir := setXDGTestHome(t)
	writeXDGTestConfig(t, homeDir, tomlBody)
}

// toolValueIndex returns the index of value in panel.toolValues (for name-based tests).
func toolValueIndex(t *testing.T, panel *SettingsPanel, value string) int {
	t.Helper()
	for i, v := range panel.toolValues {
		if v == value {
			return i
		}
	}
	t.Fatalf("tool value %q not found in %#v", value, panel.toolValues)
	return -1
}

func TestSettingsPanel_InitialState(t *testing.T) {
	panel := NewSettingsPanel()

	if panel.IsVisible() {
		t.Error("Panel should not be visible initially")
	}

	panel.Show()
	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show()")
	}
}

func TestSettingsPanel_Hide(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Hide()

	if panel.IsVisible() {
		t.Error("Panel should not be visible after Hide()")
	}
}

func TestSettingsPanel_LoadConfig(t *testing.T) {
	panel := NewSettingsPanel()

	// Load a config with specific values
	dangerousModeBool := true
	config := &session.UserConfig{
		DefaultTool: "gemini",
		Claude: session.ClaudeSettings{
			DangerousMode: &dangerousModeBool,
			ConfigDir:     "~/.claude-work",
		},
		Updates: session.UpdateSettings{
			CheckEnabled: false,
			AutoUpdate:   true,
		},
		Logs: session.LogSettings{
			MaxSizeMB:     20,
			MaxLines:      5000,
			RemoveOrphans: false,
		},
		GlobalSearch: session.GlobalSearchSettings{
			Enabled:    true,
			Tier:       "instant",
			RecentDays: 60,
		},
	}
	panel.LoadConfig(config)

	if got := panel.toolValues[panel.selectedTool]; got != "gemini" {
		t.Errorf("default tool: got %q, want %q", got, "gemini")
	}
	if !panel.dangerousMode {
		t.Error("dangerousMode should be true")
	}
	if panel.claudeConfigDir != "~/.claude-work" {
		t.Errorf("claudeConfigDir: got %q, want %q", panel.claudeConfigDir, "~/.claude-work")
	}
	if panel.checkForUpdates {
		t.Error("checkForUpdates should be false")
	}
	if !panel.autoUpdate {
		t.Error("autoUpdate should be true")
	}
	if panel.logMaxSizeMB != 20 {
		t.Errorf("logMaxSizeMB: got %d, want 20", panel.logMaxSizeMB)
	}
	if panel.logMaxLines != 5000 {
		t.Errorf("logMaxLines: got %d, want 5000", panel.logMaxLines)
	}
	if panel.removeOrphans {
		t.Error("removeOrphans should be false")
	}
	if !panel.globalSearchEnabled {
		t.Error("globalSearchEnabled should be true")
	}
	// "instant" should be index 1
	if panel.searchTier != 1 {
		t.Errorf("searchTier: got %d, want 1 (instant)", panel.searchTier)
	}
	if panel.recentDays != 60 {
		t.Errorf("recentDays: got %d, want 60", panel.recentDays)
	}
}

func TestSettingsPanel_LoadConfig_ProfileClaudeOverride(t *testing.T) {
	panel := NewSettingsPanel()
	panel.SetProfile("work")

	dangerousModeBool := true
	config := &session.UserConfig{
		Claude: session.ClaudeSettings{
			DangerousMode: &dangerousModeBool,
			ConfigDir:     "~/.claude-global",
		},
		Profiles: map[string]session.ProfileSettings{
			"work": {
				Claude: session.ProfileClaudeSettings{
					ConfigDir: "~/.claude-work",
				},
			},
		},
	}

	panel.LoadConfig(config)

	if panel.claudeConfigDir != "~/.claude-work" {
		t.Errorf("claudeConfigDir: got %q, want %q", panel.claudeConfigDir, "~/.claude-work")
	}
	if !panel.claudeConfigIsScope {
		t.Error("claudeConfigIsScope should be true when profile override exists")
	}
}

func TestSettingsPanel_LoadConfig_DefaultTool(t *testing.T) {
	panel := NewSettingsPanel()

	tests := []struct {
		name      string
		tool      string
		wantValue string // selected slot's config value ("", "" for None / unknown)
	}{
		{"claude", "claude", "claude"},
		{"gemini", "gemini", "gemini"},
		{"opencode", "opencode", "opencode"},
		{"codex", "codex", "codex"},
		{"pi", "pi", "pi"},
		{"copilot", "copilot", "copilot"},
		{"crush", "crush", "crush"},
		{"cursor", "cursor", "cursor"},
		{"hermes", "hermes", "hermes"},
		{"empty", "", ""}, // None
		{"unknown", "unknown-tool", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &session.UserConfig{
				DefaultTool: tt.tool,
			}
			panel.LoadConfig(config)
			if got := panel.toolValues[panel.selectedTool]; got != tt.wantValue {
				t.Errorf("LoadConfig(%q): toolValues[selectedTool] = %q, want %q",
					tt.tool, got, tt.wantValue)
			}
		})
	}
}

func TestSettingsPanel_LoadConfig_CustomTools(t *testing.T) {
	panel := NewSettingsPanel()

	config := &session.UserConfig{
		DefaultTool: "openclaw",
		Tools: map[string]session.ToolDef{
			"openclaw": {},
			"zeta":     {},
			"claude":   {},
		},
	}

	panel.LoadConfig(config)

	wantNames := []string{"Claude", "Gemini", "OpenCode", "Codex", "Pi", "Copilot", "Crush", "Cursor", "Hermes", "Openclaw", "Zeta", "None"}
	wantValues := []string{"claude", "gemini", "opencode", "codex", "pi", "copilot", "crush", "cursor", "hermes", "openclaw", "zeta", ""}

	if !reflect.DeepEqual(panel.toolNames, wantNames) {
		t.Fatalf("toolNames = %#v, want %#v", panel.toolNames, wantNames)
	}
	if !reflect.DeepEqual(panel.toolValues, wantValues) {
		t.Fatalf("toolValues = %#v, want %#v", panel.toolValues, wantValues)
	}
	wantIdx := toolValueIndex(t, panel, "openclaw")
	if panel.selectedTool != wantIdx {
		t.Fatalf("selectedTool = %d, want %d for openclaw", panel.selectedTool, wantIdx)
	}
}

func TestSettingsPanel_LoadConfig_SearchTier(t *testing.T) {
	panel := NewSettingsPanel()

	tests := []struct {
		name     string
		tier     string
		expected int
	}{
		{"auto", "auto", 0},
		{"instant", "instant", 1},
		{"balanced", "balanced", 2},
		{"empty", "", 0},
		{"unknown", "unknown", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &session.UserConfig{
				GlobalSearch: session.GlobalSearchSettings{
					Tier: tt.tier,
				},
			}
			panel.LoadConfig(config)
			if panel.searchTier != tt.expected {
				t.Errorf("LoadConfig tier %q: searchTier = %d, want %d",
					tt.tier, panel.searchTier, tt.expected)
			}
		})
	}
}

func TestSettingsPanel_GetConfig(t *testing.T) {
	panel := NewSettingsPanel()
	panel.selectedTool = toolValueIndex(t, panel, "opencode")
	panel.dangerousMode = true
	panel.claudeConfigDir = "~/.claude-custom"
	panel.checkForUpdates = false
	panel.autoUpdate = true
	panel.logMaxSizeMB = 15
	panel.logMaxLines = 8000
	panel.removeOrphans = false
	panel.globalSearchEnabled = true
	panel.searchTier = 2 // balanced
	panel.recentDays = 45

	config := panel.GetConfig()

	if config.DefaultTool != "opencode" {
		t.Errorf("DefaultTool: got %q, want %q", config.DefaultTool, "opencode")
	}
	if !config.Claude.GetDangerousMode() {
		t.Error("DangerousMode should be true")
	}
	if config.Claude.ConfigDir != "~/.claude-custom" {
		t.Errorf("ConfigDir: got %q, want %q", config.Claude.ConfigDir, "~/.claude-custom")
	}
	if config.Updates.CheckEnabled {
		t.Error("CheckEnabled should be false")
	}
	if !config.Updates.AutoUpdate {
		t.Error("AutoUpdate should be true")
	}
	if config.Logs.MaxSizeMB != 15 {
		t.Errorf("MaxSizeMB: got %d, want 15", config.Logs.MaxSizeMB)
	}
	if config.Logs.MaxLines != 8000 {
		t.Errorf("MaxLines: got %d, want 8000", config.Logs.MaxLines)
	}
	if config.Logs.RemoveOrphans {
		t.Error("RemoveOrphans should be false")
	}
	if !config.GlobalSearch.Enabled {
		t.Error("GlobalSearch.Enabled should be true")
	}
	if config.GlobalSearch.Tier != "balanced" {
		t.Errorf("Tier: got %q, want %q", config.GlobalSearch.Tier, "balanced")
	}
	if config.GlobalSearch.RecentDays != 45 {
		t.Errorf("RecentDays: got %d, want 45", config.GlobalSearch.RecentDays)
	}
}

func TestSettingsPanel_GetConfig_PreservesProfileOverrides(t *testing.T) {
	panel := NewSettingsPanel()
	panel.originalConfig = &session.UserConfig{
		Profiles: map[string]session.ProfileSettings{
			"work": {
				Claude: session.ProfileClaudeSettings{
					ConfigDir: "~/.claude-work",
				},
			},
		},
	}

	config := panel.GetConfig()
	profileCfg, ok := config.Profiles["work"]
	if !ok {
		t.Fatal("expected profile override for work to be preserved")
	}
	if profileCfg.Claude.ConfigDir != "~/.claude-work" {
		t.Errorf("profile override config dir: got %q, want %q", profileCfg.Claude.ConfigDir, "~/.claude-work")
	}
}

func TestSettingsPanel_GetConfig_UpdatesProfileClaudeOverride(t *testing.T) {
	panel := NewSettingsPanel()
	panel.SetProfile("work")
	panel.claudeConfigDir = "~/.claude-work-updated"
	panel.claudeConfigIsScope = true
	panel.originalConfig = &session.UserConfig{
		Claude: session.ClaudeSettings{
			ConfigDir: "~/.claude-global",
		},
		Profiles: map[string]session.ProfileSettings{
			"work": {
				Claude: session.ProfileClaudeSettings{
					ConfigDir: "~/.claude-work",
				},
			},
		},
	}

	config := panel.GetConfig()
	if config.Claude.ConfigDir != "~/.claude-global" {
		t.Errorf("global Claude.ConfigDir changed unexpectedly: got %q, want %q", config.Claude.ConfigDir, "~/.claude-global")
	}
	if got := config.Profiles["work"].Claude.ConfigDir; got != "~/.claude-work-updated" {
		t.Errorf("profile override config dir: got %q, want %q", got, "~/.claude-work-updated")
	}
}

func TestSettingsPanel_GetConfig_ToolMapping(t *testing.T) {
	panel := NewSettingsPanel()

	tests := []struct {
		name     string
		value    string
		expected string
	}{
		{"claude", "claude", "claude"},
		{"gemini", "gemini", "gemini"},
		{"opencode", "opencode", "opencode"},
		{"codex", "codex", "codex"},
		{"pi", "pi", "pi"},
		{"copilot", "copilot", "copilot"},
		{"crush", "crush", "crush"},
		{"cursor", "cursor", "cursor"},
		{"hermes", "hermes", "hermes"},
		{"none", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel.selectedTool = toolValueIndex(t, panel, tt.value)
			config := panel.GetConfig()
			if config.DefaultTool != tt.expected {
				t.Errorf("GetConfig for tool %q: DefaultTool = %q, want %q",
					tt.value, config.DefaultTool, tt.expected)
			}
		})
	}
}

func TestSettingsPanel_GetConfig_CustomToolMapping(t *testing.T) {
	panel := NewSettingsPanel()
	panel.LoadConfig(&session.UserConfig{
		Tools: map[string]session.ToolDef{
			"openclaw": {},
		},
	})

	panel.selectedTool = toolValueIndex(t, panel, "openclaw")
	config := panel.GetConfig()
	if config.DefaultTool != "openclaw" {
		t.Fatalf("DefaultTool: got %q, want %q", config.DefaultTool, "openclaw")
	}
}

func TestSettingsPanel_GetConfig_TierMapping(t *testing.T) {
	panel := NewSettingsPanel()

	tests := []struct {
		name     string
		index    int
		expected string
	}{
		{"auto", 0, "auto"},
		{"instant", 1, "instant"},
		{"balanced", 2, "balanced"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel.searchTier = tt.index
			config := panel.GetConfig()
			if config.GlobalSearch.Tier != tt.expected {
				t.Errorf("GetConfig for tier index %d: Tier = %q, want %q",
					tt.index, config.GlobalSearch.Tier, tt.expected)
			}
		})
	}
}

func TestSettingsPanel_SetSize(t *testing.T) {
	panel := NewSettingsPanel()
	panel.SetSize(120, 60)

	if panel.width != 120 {
		t.Errorf("Width = %d, want 120", panel.width)
	}
	if panel.height != 60 {
		t.Errorf("Height = %d, want 60", panel.height)
	}
}

func TestSettingsPanel_Update_PaneNavigation(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	// Initial focus is the section list (left pane), first section selected.
	if panel.focusRight {
		t.Error("Initial focus should be the section list (left pane)")
	}
	if panel.sectionCursor != 0 {
		t.Errorf("Initial sectionCursor = %d, want 0", panel.sectionCursor)
	}

	// Down moves the section selection in the left pane.
	panel.Update(tea.KeyMsg{Type: tea.KeyDown})
	if panel.sectionCursor != 1 {
		t.Errorf("After down: sectionCursor = %d, want 1", panel.sectionCursor)
	}
	// Selecting a section snaps the detail cursor to its first setting.
	if panel.cursor != int(settingsSections[1].settings[0]) {
		t.Errorf("After section move: cursor = %d, want %d", panel.cursor, settingsSections[1].settings[0])
	}

	// 'k' moves the section selection back up.
	panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if panel.sectionCursor != 0 {
		t.Errorf("After k: sectionCursor = %d, want 0", panel.sectionCursor)
	}

	// Should not go above the first section.
	panel.Update(tea.KeyMsg{Type: tea.KeyUp})
	if panel.sectionCursor != 0 {
		t.Errorf("After up at 0: sectionCursor = %d, want 0", panel.sectionCursor)
	}

	// Right enters the detail pane; up/down then move within the section.
	// Appearance has multiple settings, so detail navigation is observable.
	panel.Update(tea.KeyMsg{Type: tea.KeyRight})
	if !panel.focusRight {
		t.Error("Right from section list should focus the detail pane")
	}
	if panel.cursor != int(settingsSections[0].settings[0]) {
		t.Errorf("Detail cursor = %d, want first setting %d", panel.cursor, settingsSections[0].settings[0])
	}
	panel.Update(tea.KeyMsg{Type: tea.KeyDown})
	if panel.cursor != int(settingsSections[0].settings[1]) {
		t.Errorf("After down in detail: cursor = %d, want %d", panel.cursor, settingsSections[0].settings[1])
	}

	// Tab toggles focus back to the section list.
	panel.Update(tea.KeyMsg{Type: tea.KeyTab})
	if panel.focusRight {
		t.Error("Tab should toggle focus back to the section list")
	}
}

func TestSettingsPanel_Update_ToggleCheckbox(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	// Focus the dangerous-mode setting in the detail pane.
	panel.focusSetting(SettingDangerousMode)
	initialValue := panel.dangerousMode

	// Toggle with space
	_, _, changed := panel.Update(tea.KeyMsg{Type: tea.KeySpace})
	if panel.dangerousMode == initialValue {
		t.Error("dangerousMode should have toggled")
	}
	if !changed {
		t.Error("Update should return changed=true when value changes")
	}

	// Toggle back
	panel.Update(tea.KeyMsg{Type: tea.KeySpace})
	if panel.dangerousMode != initialValue {
		t.Error("dangerousMode should have toggled back")
	}
}

func TestSettingsPanel_Update_RadioSelection(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	// Focus the default-tool radio in the detail pane.
	panel.focusSetting(SettingDefaultTool)
	panel.selectedTool = 0 // claude

	// Move right
	panel.Update(tea.KeyMsg{Type: tea.KeyRight})
	if panel.selectedTool != 1 {
		t.Errorf("After right: selectedTool = %d, want 1", panel.selectedTool)
	}

	// Move right with 'l'
	panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if panel.selectedTool != 2 {
		t.Errorf("After l: selectedTool = %d, want 2", panel.selectedTool)
	}

	// Move left
	panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if panel.selectedTool != 1 {
		t.Errorf("After left: selectedTool = %d, want 1", panel.selectedTool)
	}

	// Move left with 'h'
	panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	if panel.selectedTool != 0 {
		t.Errorf("After h: selectedTool = %d, want 0", panel.selectedTool)
	}

	// Should not go below 0
	panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if panel.selectedTool != 0 {
		t.Errorf("After left at 0: selectedTool = %d, want 0", panel.selectedTool)
	}
}

func TestSettingsPanel_Update_NumberAdjustment(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	// Focus the log max-size number in the detail pane.
	panel.focusSetting(SettingLogMaxSize)
	panel.logMaxSizeMB = 10

	// Increase with right
	panel.Update(tea.KeyMsg{Type: tea.KeyRight})
	if panel.logMaxSizeMB != 11 {
		t.Errorf("After right: logMaxSizeMB = %d, want 11", panel.logMaxSizeMB)
	}

	// Decrease with left
	panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if panel.logMaxSizeMB != 10 {
		t.Errorf("After left: logMaxSizeMB = %d, want 10", panel.logMaxSizeMB)
	}

	// Should not go below 1
	panel.logMaxSizeMB = 1
	panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if panel.logMaxSizeMB != 1 {
		t.Errorf("After left at 1: logMaxSizeMB = %d, want 1", panel.logMaxSizeMB)
	}
}

func TestSettingsPanel_Update_Escape(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Update(tea.KeyMsg{Type: tea.KeyEsc})

	if panel.IsVisible() {
		t.Error("Panel should be hidden after Escape")
	}
}

func TestSettingsPanel_Update_SKey(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	if !panel.IsVisible() {
		t.Error("Panel should be visible after Show()")
	}

	panel.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})

	if panel.IsVisible() {
		t.Error("Panel should be hidden after pressing S")
	}
}

func TestSettingsPanel_NeedsRestart(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	// Initially should not need restart
	if panel.NeedsRestart() {
		t.Error("Should not need restart initially")
	}

	// Focus and toggle the global-search enabled setting.
	panel.focusSetting(SettingGlobalSearchEnabled)
	panel.Update(tea.KeyMsg{Type: tea.KeySpace})

	if !panel.NeedsRestart() {
		t.Error("Should need restart after changing global search setting")
	}
}

func TestSettingsPanel_View_NotVisible(t *testing.T) {
	panel := NewSettingsPanel()

	view := panel.View()
	if view != "" {
		t.Errorf("View() should return empty string when not visible, got %q", view)
	}
}

func TestSettingsPanel_View_Visible(t *testing.T) {
	panel := NewSettingsPanel()
	panel.SetSize(100, 80)
	panel.Show()

	view := panel.View()
	if view == "" {
		t.Error("View() should return non-empty string when visible")
	}

	// The left pane always shows the section list; the right pane shows the
	// selected (first: Appearance) section's settings.
	expectedElements := []string{
		"Settings",        // title
		"SECTIONS",        // left-pane header
		"Appearance",      // a section name in the left pane
		"Agents — Claude", // grouped agent section
		"System Stats",
		"MCP & Tools",
		"APPEARANCE", // right-pane detail header for the selected section
		"Theme",
		"Dark",
		"Light",
	}

	for _, elem := range expectedElements {
		if !containsString(view, elem) {
			t.Errorf("View() should contain %q", elem)
		}
	}
}

func TestSettingsPanel_View_HighlightsCursor(t *testing.T) {
	panel := NewSettingsPanel()
	panel.SetSize(80, 50) // Increased height for new settings
	panel.Show()

	// Render every section with focus on each setting; verify no crash.
	for i := 0; i < settingsCount; i++ {
		panel.focusSetting(SettingType(i))
		view := panel.View()
		if view == "" {
			t.Errorf("View() should return non-empty for setting %d", i)
		}
	}

	// Also render each section with focus on the left (section list) pane.
	for si := range settingsSections {
		panel.sectionCursor = si
		panel.focusRight = false
		if view := panel.View(); view == "" {
			t.Errorf("View() should return non-empty for section %d", si)
		}
	}
}

// Helper function to check if a string contains a substring
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestSettingsPanel_ThemeToggle(t *testing.T) {
	panel := NewSettingsPanel()
	// Set dark explicitly (avoid loading config from disk)
	panel.selectedTheme = 0
	panel.visible = true

	// Focus the theme radio, then navigate right to select light.
	panel.focusSetting(SettingTheme)
	panel, _, shouldSave := panel.Update(tea.KeyMsg{Type: tea.KeyRight})

	if panel.selectedTheme != 1 {
		t.Errorf("Theme should be 1 (light) after right, got %d", panel.selectedTheme)
	}
	if !shouldSave {
		t.Error("Should trigger save on theme change")
	}

	// Navigate right to select system
	panel, _, shouldSave = panel.Update(tea.KeyMsg{Type: tea.KeyRight})
	if panel.selectedTheme != 2 {
		t.Errorf("Theme should be 2 (system) after right, got %d", panel.selectedTheme)
	}
	if !shouldSave {
		t.Error("Should trigger save on theme change to system")
	}

	// Theme changes should not require restart (applied live)
	if panel.needsRestart {
		t.Error("Theme change should not require restart")
	}
}

func TestSettingsPanel_LoadConfig_Theme(t *testing.T) {
	tests := []struct {
		name     string
		theme    string
		expected int
	}{
		{"dark", "dark", 0},
		{"light", "light", 1},
		{"system", "system", 2},
		{"empty defaults to dark", "", 0},
		{"invalid defaults to dark", "invalid", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := NewSettingsPanel()
			config := &session.UserConfig{Theme: tt.theme}
			panel.LoadConfig(config)

			if panel.selectedTheme != tt.expected {
				t.Errorf("selectedTheme: got %d, want %d", panel.selectedTheme, tt.expected)
			}
		})
	}
}

func TestSettingsPanel_GetConfig_Theme(t *testing.T) {
	tests := []struct {
		name          string
		selectedTheme int
		expected      string
	}{
		{"dark", 0, "dark"},
		{"light", 1, "light"},
		{"system", 2, "system"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			panel := NewSettingsPanel()
			panel.selectedTheme = tt.selectedTheme
			config := panel.GetConfig()

			if config.Theme != tt.expected {
				t.Errorf("Theme: got %q, want %q", config.Theme, tt.expected)
			}
		})
	}
}

func TestSettingsPanelPreviewSettings(t *testing.T) {
	sp := NewSettingsPanel()

	// Find Show Output setting
	foundOutput := false
	foundAnalytics := false
	for i := 0; i < settingsCount; i++ {
		sp.cursor = i
		setting := SettingType(i)
		switch setting {
		case SettingShowOutput:
			foundOutput = true
		case SettingShowAnalytics:
			foundAnalytics = true
		}
	}

	if !foundOutput {
		t.Error("Show Output setting should exist (SettingShowOutput constant)")
	}
	if !foundAnalytics {
		t.Error("Show Analytics setting should exist (SettingShowAnalytics constant)")
	}
}

func TestSettingsPanel_PreviewSettings_Toggle(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	// Test Show Output toggle
	panel.focusSetting(SettingShowOutput)
	initialOutput := panel.showOutput

	_, _, changed := panel.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !changed {
		t.Error("Show Output toggle should report changed=true")
	}
	if panel.showOutput == initialOutput {
		t.Error("Show Output should have toggled")
	}

	// Test Show Analytics toggle
	panel.focusSetting(SettingShowAnalytics)
	initialAnalytics := panel.showAnalytics

	_, _, changed = panel.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !changed {
		t.Error("Show Analytics toggle should report changed=true")
	}
	if panel.showAnalytics == initialAnalytics {
		t.Error("Show Analytics should have toggled")
	}
}

func TestSettingsPanel_PreviewSettings_LoadConfig(t *testing.T) {
	panel := NewSettingsPanel()

	// Test loading with explicit values
	showOutputTrue := true
	showAnalyticsFalse := false
	config := &session.UserConfig{
		Preview: session.PreviewSettings{
			ShowOutput:    &showOutputTrue,
			ShowAnalytics: &showAnalyticsFalse,
		},
	}
	panel.LoadConfig(config)

	if !panel.showOutput {
		t.Error("showOutput should be true after loading config")
	}
	if panel.showAnalytics {
		t.Error("showAnalytics should be false after loading config")
	}

	// Test loading with nil ShowAnalytics (should default to false)
	showOutputFalse := false
	config2 := &session.UserConfig{
		Preview: session.PreviewSettings{
			ShowOutput:    &showOutputFalse,
			ShowAnalytics: nil,
		},
	}
	panel.LoadConfig(config2)

	if panel.showOutput {
		t.Error("showOutput should be false after loading config2")
	}
	if panel.showAnalytics {
		t.Error("showAnalytics should default to false when nil")
	}
}

func TestSettingsPanel_PreviewSettings_GetConfig(t *testing.T) {
	panel := NewSettingsPanel()
	panel.showOutput = true
	panel.showAnalytics = false

	config := panel.GetConfig()

	if config.Preview.ShowOutput == nil || !*config.Preview.ShowOutput {
		t.Error("Preview.ShowOutput should be true")
	}
	if config.Preview.ShowAnalytics == nil {
		t.Error("Preview.ShowAnalytics should not be nil")
	} else if *config.Preview.ShowAnalytics {
		t.Error("Preview.ShowAnalytics should be false")
	}
}

func TestSettingsPanel_PreviewSettings_GetConfigPreservesHiddenFields(t *testing.T) {
	panel := NewSettingsPanel()
	panel.showOutput = false
	panel.showAnalytics = true

	showNotes := false
	showTools := false
	original := &session.UserConfig{
		Preview: session.PreviewSettings{
			ShowNotes:        &showNotes,
			NotesOutputSplit: 0.42,
			Analytics: session.AnalyticsDisplaySettings{
				ShowTools: &showTools,
			},
		},
	}
	panel.LoadConfig(original)
	panel.originalConfig = original

	config := panel.GetConfig()

	// ShowNotes is now editable; LoadConfig sets it from original
	if config.Preview.ShowNotes == nil || *config.Preview.ShowNotes {
		t.Fatal("Preview.ShowNotes should be false after loading config with false")
	}
	// NotesOutputSplit is now editable; LoadConfig converts 0.42 -> 42% -> int(42)
	// GetConfig converts back: 42 / 100.0 = 0.42
	if config.Preview.NotesOutputSplit != 0.42 {
		t.Fatalf("Preview.NotesOutputSplit = %v, want 0.42", config.Preview.NotesOutputSplit)
	}
	if config.Preview.Analytics.ShowTools == nil || *config.Preview.Analytics.ShowTools {
		t.Fatal("Preview.Analytics should preserve original hidden settings")
	}
}

func TestSettingsPanel_Worktree_GetConfigPreservesHiddenFields(t *testing.T) {
	panel := NewSettingsPanel()

	branchPrefix := "dev/"
	pathTemplate := "~/worktrees/{repo-name}/{branch}"
	original := &session.UserConfig{
		Worktree: session.WorktreeSettings{
			AutoCleanup:     true,
			DefaultEnabled:  true,
			DefaultLocation: "sibling",
			PathTemplate:    &pathTemplate,
			BranchPrefix:    &branchPrefix,
		},
	}
	panel.LoadConfig(original)
	panel.originalConfig = original

	config := panel.GetConfig()

	if !config.Worktree.AutoCleanup {
		t.Fatal("Worktree.AutoCleanup should be preserved")
	}
	if !config.Worktree.DefaultEnabled {
		t.Fatal("Worktree.DefaultEnabled should be preserved")
	}
	if config.Worktree.DefaultLocation != "sibling" {
		t.Fatalf("Worktree.DefaultLocation = %q, want %q", config.Worktree.DefaultLocation, "sibling")
	}
	if config.Worktree.PathTemplate == nil || *config.Worktree.PathTemplate != pathTemplate {
		t.Fatalf("Worktree.PathTemplate should be preserved, got %v", config.Worktree.PathTemplate)
	}
	if config.Worktree.BranchPrefix == nil || *config.Worktree.BranchPrefix != branchPrefix {
		t.Fatalf("Worktree.BranchPrefix should be preserved, got %v", config.Worktree.BranchPrefix)
	}
}

func TestSettingsPanel_Fork_GetConfigPreservesHiddenFields(t *testing.T) {
	panel := NewSettingsPanel()

	worktree := false
	withState := false
	withIgnored := false
	dockerMode := "off"
	original := &session.UserConfig{
		Fork: session.ForkSettings{
			InheritFromParent: true,
			Worktree:          &worktree,
			WithState:         &withState,
			WithIgnored:       &withIgnored,
			Docker:            &dockerMode,
			BranchPrefix:      "wip/",
		},
	}
	panel.LoadConfig(original)
	panel.originalConfig = original

	config := panel.GetConfig()

	if !config.Fork.InheritFromParent {
		t.Fatal("Fork.InheritFromParent should be preserved")
	}
	if config.Fork.Worktree == nil || *config.Fork.Worktree {
		t.Fatalf("Fork.Worktree should preserve explicit false, got %v", config.Fork.Worktree)
	}
	if config.Fork.WithState == nil || *config.Fork.WithState {
		t.Fatalf("Fork.WithState should preserve explicit false, got %v", config.Fork.WithState)
	}
	if config.Fork.WithIgnored == nil || *config.Fork.WithIgnored {
		t.Fatalf("Fork.WithIgnored should preserve explicit false, got %v", config.Fork.WithIgnored)
	}
	if config.Fork.Docker == nil || *config.Fork.Docker != "off" {
		t.Fatalf("Fork.Docker should preserve off, got %v", config.Fork.Docker)
	}
	if config.Fork.BranchPrefix != "wip/" {
		t.Fatalf("Fork.BranchPrefix = %q, want %q", config.Fork.BranchPrefix, "wip/")
	}
}

// TestSettingsPanel_Tmux_GetConfigPreservesHiddenFields guards #710.
// The Settings TUI does not expose [tmux] fields, so GetConfig() must copy
// them through from originalConfig — same as MCPs/Tools/Worktree. Before the
// fix, saving from the TUI silently dropped the entire [tmux] table, which
// also explained the original #687 inject_status_line report we couldn't
// reproduce by editing config.toml directly.
func TestSettingsPanel_Tmux_GetConfigPreservesHiddenFields(t *testing.T) {
	panel := NewSettingsPanel()

	injectFalse := false
	launchScopeTrue := true
	original := &session.UserConfig{
		Tmux: session.TmuxSettings{
			InjectStatusLine:  &injectFalse,
			LaunchInUserScope: &launchScopeTrue,
			DetachKey:         "C-q",
		},
	}
	panel.LoadConfig(original)
	panel.originalConfig = original

	config := panel.GetConfig()

	if config.Tmux.InjectStatusLine == nil || *config.Tmux.InjectStatusLine != false {
		t.Fatalf("Tmux.InjectStatusLine should be preserved as false, got %v", config.Tmux.InjectStatusLine)
	}
	if config.Tmux.LaunchInUserScope == nil || *config.Tmux.LaunchInUserScope != true {
		t.Fatalf("Tmux.LaunchInUserScope should be preserved as true, got %v", config.Tmux.LaunchInUserScope)
	}
	if config.Tmux.DetachKey != "C-q" {
		t.Fatalf("Tmux.DetachKey = %q, want %q", config.Tmux.DetachKey, "C-q")
	}
}

func TestSettingsPanel_PreviewSettings_ViewContains(t *testing.T) {
	panel := NewSettingsPanel()
	panel.SetSize(80, 50)
	panel.Show()

	// Show Output / Show Analytics live in the Appearance section (selected by
	// default after Show()).
	view := panel.View()

	expectedElements := []string{
		"APPEARANCE",
		"Show Output",
		"Show Analytics",
	}

	for _, elem := range expectedElements {
		if !containsString(view, elem) {
			t.Errorf("View() should contain %q", elem)
		}
	}
}

func TestSettingsPanel_ViewUsesConfiguredMCPHotkeyHint(t *testing.T) {
	setSettingsPanelHotkeyConfigForTest(t, "[hotkeys]\nmcp_manager = \"ctrl+m\"\n")

	panel := NewSettingsPanel()
	panel.SetSize(120, 200)
	panel.Show()
	selectSectionByTitle(t, panel, "MCP & Tools")

	view := panel.View()
	// Hint may wrap across dialog lines; assert on stable fragments rather than one contiguous string.
	if !containsString(view, "MCP Manager hotkey: ctrl+m") ||
		!containsString(view, "Attach MCPs") {
		t.Fatalf("settings view should show configured MCP key hint, got %q", view)
	}
}

func TestSettingsPanel_ViewShowsUnboundMCPHotkeyHint(t *testing.T) {
	setSettingsPanelHotkeyConfigForTest(t, "[hotkeys]\nmcp_manager = \"\"\n")

	panel := NewSettingsPanel()
	panel.SetSize(100, 80)
	panel.Show()
	selectSectionByTitle(t, panel, "MCP & Tools")

	view := panel.View()
	if !containsString(view, "MCP Manager hotkey is unbound.") {
		t.Fatalf("settings view should show unbound MCP key hint, got %q", view)
	}
}

// selectSectionByTitle points the left pane at the named section (focus stays
// on the section list).
func selectSectionByTitle(t *testing.T, panel *SettingsPanel, title string) {
	t.Helper()
	for i, sec := range settingsSections {
		if sec.title == title {
			panel.sectionCursor = i
			panel.focusRight = false
			return
		}
	}
	t.Fatalf("section %q not found", title)
}

// TestSettingsSectionsCoverAllSettings guards the core invariant of the
// two-pane redesign: the editable sections together cover exactly the
// settingsCount SettingType entries, each in exactly one section.
func TestSettingsSectionsCoverAllSettings(t *testing.T) {
	seen := make(map[SettingType]int)
	for _, sec := range settingsSections {
		if sec.info {
			if len(sec.settings) != 0 {
				t.Errorf("info section %q should have no settings, got %d", sec.title, len(sec.settings))
			}
			continue
		}
		for _, st := range sec.settings {
			seen[st]++
		}
	}

	if len(seen) != settingsCount {
		t.Errorf("sections cover %d distinct settings, want %d", len(seen), settingsCount)
	}
	for i := 0; i < settingsCount; i++ {
		st := SettingType(i)
		switch seen[st] {
		case 0:
			t.Errorf("setting %d is not in any section", i)
		case 1:
			// exactly once — good
		default:
			t.Errorf("setting %d appears in %d sections, want exactly 1", i, seen[st])
		}
	}
}

// TestSettingsPanel_AllSettingsReachable confirms every SettingType can be
// reached by navigating the section list and entering the detail pane — no
// setting is orphaned by the reorganization.
func TestSettingsPanel_AllSettingsReachable(t *testing.T) {
	reachable := make(map[int]bool)
	for si := range settingsSections {
		panel := NewSettingsPanel()
		panel.Show()
		// Walk the section list down to section si.
		for panel.sectionCursor < si {
			panel.Update(tea.KeyMsg{Type: tea.KeyDown})
		}
		if panel.sectionCursor != si {
			t.Fatalf("could not navigate to section %d (stuck at %d)", si, panel.sectionCursor)
		}
		// Enter the detail pane and walk every setting in the section.
		panel.Update(tea.KeyMsg{Type: tea.KeyRight})
		settings := settingsSections[si].settings
		if len(settings) == 0 {
			continue // informational section
		}
		if !panel.focusRight {
			t.Fatalf("section %d (%q) should be focusable", si, settingsSections[si].title)
		}
		reachable[panel.cursor] = true
		for j := 1; j < len(settings); j++ {
			panel.Update(tea.KeyMsg{Type: tea.KeyDown})
			reachable[panel.cursor] = true
		}
	}

	for i := 0; i < settingsCount; i++ {
		if !reachable[i] {
			t.Errorf("setting %d is not reachable via pane navigation", i)
		}
	}
}

// TestSettingsPanel_LeftAdjustVsBack verifies the dual role of left/h in the
// detail pane: it tunes adjustable settings but steps focus back to the
// section list on toggle/text settings.
func TestSettingsPanel_LeftAdjustVsBack(t *testing.T) {
	panel := NewSettingsPanel()
	panel.Show()

	// Adjustable setting: left tunes the value, focus stays on the right.
	panel.focusSetting(SettingLogMaxSize)
	panel.logMaxSizeMB = 5
	panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if panel.logMaxSizeMB != 4 {
		t.Errorf("left on adjustable setting should decrement: got %d, want 4", panel.logMaxSizeMB)
	}
	if !panel.focusRight {
		t.Error("left on adjustable setting should keep focus on the detail pane")
	}

	// Toggle setting: left steps focus back to the section list.
	panel.focusSetting(SettingDangerousMode)
	before := panel.dangerousMode
	panel.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if panel.focusRight {
		t.Error("left on a toggle setting should return focus to the section list")
	}
	if panel.dangerousMode != before {
		t.Error("left on a toggle setting should not change its value")
	}
}
