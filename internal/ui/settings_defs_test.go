package ui

import (
	"testing"

	"github.com/asheshgoplani/agent-deck/internal/session"
	tea "github.com/charmbracelet/bubbletea"
)

// boolPtrVal / intPtrVal / strPtrVal are local helpers for building configs.
func boolPtrVal(b bool) *bool    { return &b }
func intPtrVal(i int) *int       { return &i }
func strPtrVal(s string) *string { return &s }

// TestSettingsRegistry_RoundTrip loads a config exercising the new
// registry-driven fields across every input kind (bool / enum / int / literal)
// and asserts GetConfig writes the same values back.
func TestSettingsRegistry_RoundTrip(t *testing.T) {
	original := &session.UserConfig{
		UI: session.UISettings{
			PreviewPct:             40,
			Footer:                 session.FooterMinimal,
			ShowOnlyInstalledTools: true,
		},
		Claude: session.ClaudeSettings{
			Command:      "cdw",
			DefaultModel: "claude-opus-4-8",
			VimMode:      true,
			HooksEnabled: boolPtrVal(false),
		},
		Tmux: session.TmuxSettings{
			Mouse:      boolPtrVal(false),
			LaunchAs:   strPtrVal("service"),
			SocketName: "agentdeck",
		},
		Worktree: session.WorktreeSettings{
			DefaultLocation:     "subdirectory",
			BranchPrefix:        strPtrVal("dev/"),
			SetupTimeoutSeconds: intPtrVal(0), // explicit unlimited
		},
		Instances: session.InstanceSettings{
			AllowMultiple: boolPtrVal(true),
		},
	}

	panel := NewSettingsPanel()
	panel.LoadConfig(original)
	panel.originalConfig = original

	got := panel.GetConfig()

	if got.UI.PreviewPct != 40 {
		t.Errorf("UI.PreviewPct = %d, want 40", got.UI.PreviewPct)
	}
	if got.UI.Footer != session.FooterMinimal {
		t.Errorf("UI.Footer = %q, want %q", got.UI.Footer, session.FooterMinimal)
	}
	if !got.UI.ShowOnlyInstalledTools {
		t.Error("UI.ShowOnlyInstalledTools should be true")
	}
	if got.Claude.Command != "cdw" {
		t.Errorf("Claude.Command = %q, want %q", got.Claude.Command, "cdw")
	}
	if got.Claude.DefaultModel != "claude-opus-4-8" {
		t.Errorf("Claude.DefaultModel = %q, want %q", got.Claude.DefaultModel, "claude-opus-4-8")
	}
	if !got.Claude.VimMode {
		t.Error("Claude.VimMode should be true")
	}
	if got.Claude.HooksEnabled == nil || *got.Claude.HooksEnabled {
		t.Errorf("Claude.HooksEnabled should round-trip as explicit false, got %v", got.Claude.HooksEnabled)
	}
	if got.Tmux.Mouse == nil || *got.Tmux.Mouse {
		t.Errorf("Tmux.Mouse should round-trip as explicit false, got %v", got.Tmux.Mouse)
	}
	if got.Tmux.GetLaunchAs() != "service" {
		t.Errorf("Tmux.LaunchAs = %q, want %q", got.Tmux.GetLaunchAs(), "service")
	}
	if got.Tmux.SocketName != "agentdeck" {
		t.Errorf("Tmux.SocketName = %q, want %q", got.Tmux.SocketName, "agentdeck")
	}
	if got.Worktree.DefaultLocation != "subdirectory" {
		t.Errorf("Worktree.DefaultLocation = %q, want %q", got.Worktree.DefaultLocation, "subdirectory")
	}
	if got.Worktree.BranchPrefix == nil || *got.Worktree.BranchPrefix != "dev/" {
		t.Errorf("Worktree.BranchPrefix should round-trip as dev/, got %v", got.Worktree.BranchPrefix)
	}
	if got.Worktree.SetupTimeoutSeconds == nil || *got.Worktree.SetupTimeoutSeconds != 0 {
		t.Errorf("Worktree.SetupTimeoutSeconds should round-trip as explicit 0, got %v", got.Worktree.SetupTimeoutSeconds)
	}
	if got.Instances.GetAllowMultiple() != true {
		t.Error("Instances.AllowMultiple should round-trip as true")
	}
}

// TestSettingsRegistry_PreservesHiddenSiblingFields guards the base-copy
// contract: fields NOT surfaced in the TUI (Claude.ExtraArgs, Tmux.Options,
// Worktree.PathTemplate) must survive a save unchanged even though their
// sibling fields are now editable.
func TestSettingsRegistry_PreservesHiddenSiblingFields(t *testing.T) {
	pathTemplate := "~/wt/{repo-name}/{branch}"
	original := &session.UserConfig{
		Claude: session.ClaudeSettings{
			Command:   "claude",
			ExtraArgs: []string{"--verbose", "--foo"},
		},
		Tmux: session.TmuxSettings{
			Options: map[string]string{"history-limit": "50000"},
		},
		Worktree: session.WorktreeSettings{
			PathTemplate: &pathTemplate,
		},
		Gemini: session.GeminiSettings{DefaultModel: "gemini-2.5-flash"},
	}

	panel := NewSettingsPanel()
	panel.LoadConfig(original)
	panel.originalConfig = original

	got := panel.GetConfig()

	if len(got.Claude.ExtraArgs) != 2 || got.Claude.ExtraArgs[0] != "--verbose" || got.Claude.ExtraArgs[1] != "--foo" {
		t.Errorf("Claude.ExtraArgs not preserved: %#v", got.Claude.ExtraArgs)
	}
	if got.Tmux.Options["history-limit"] != "50000" {
		t.Errorf("Tmux.Options not preserved: %#v", got.Tmux.Options)
	}
	if got.Worktree.PathTemplate == nil || *got.Worktree.PathTemplate != pathTemplate {
		t.Errorf("Worktree.PathTemplate not preserved: %v", got.Worktree.PathTemplate)
	}
	if got.Gemini.DefaultModel != "gemini-2.5-flash" {
		t.Errorf("Gemini.DefaultModel not preserved: %q", got.Gemini.DefaultModel)
	}
}

// TestSettingsRegistry_EditFlows exercises the generic edit dispatch for each
// kind via the public Update path: toggle a bool, step an enum, step an int,
// and confirm each reports changed=true and mutates the value store.
func TestSettingsRegistry_EditFlows(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	session.ClearUserConfigCache()
	t.Cleanup(session.ClearUserConfigCache)

	panel := NewSettingsPanel()
	panel.Show()

	// Bool: Crush YOLO mode.
	panel.focusSetting(SettingCrushYoloMode)
	before := panel.newBool[SettingCrushYoloMode]
	_, _, changed := panel.Update(tea.KeyMsg{Type: tea.KeySpace})
	if !changed || panel.newBool[SettingCrushYoloMode] == before {
		t.Error("toggling SettingCrushYoloMode should flip the value and report changed")
	}

	// Enum: Footer style steps right.
	panel.focusSetting(SettingFooter)
	panel.newEnum[SettingFooter] = 0
	_, _, changed = panel.Update(tea.KeyMsg{Type: tea.KeyRight})
	if !changed || panel.newEnum[SettingFooter] != 1 {
		t.Errorf("right on SettingFooter should advance enum index to 1, got %d", panel.newEnum[SettingFooter])
	}

	// Int: preview width steps by 5 and clamps at the max.
	panel.focusSetting(SettingPreviewPct)
	panel.newInt[SettingPreviewPct] = session.MaxPreviewPct - 5
	_, _, changed = panel.Update(tea.KeyMsg{Type: tea.KeyRight})
	if !changed || panel.newInt[SettingPreviewPct] != session.MaxPreviewPct {
		t.Errorf("right on SettingPreviewPct should reach max %d, got %d", session.MaxPreviewPct, panel.newInt[SettingPreviewPct])
	}
	_, _, changed = panel.Update(tea.KeyMsg{Type: tea.KeyRight})
	if changed || panel.newInt[SettingPreviewPct] != session.MaxPreviewPct {
		t.Error("right at max preview width should not change the value")
	}
}

// TestSettingsRegistry_LiteralIsTextSetting confirms literals route through the
// text-edit path and other kinds do not.
func TestSettingsRegistry_LiteralIsTextSetting(t *testing.T) {
	panel := NewSettingsPanel()
	panel.visible = true // Update() early-returns when not visible

	panel.focusSetting(SettingTmuxSocketName)
	if !panel.isTextSetting() {
		t.Error("SettingTmuxSocketName (literal) should be a text setting")
	}
	panel.focusSetting(SettingTmuxMouse)
	if panel.isTextSetting() {
		t.Error("SettingTmuxMouse (bool) should not be a text setting")
	}

	// Editing a literal updates its value store via the text buffer.
	panel.focusSetting(SettingTmuxSocketName)
	panel.startTextEdit()
	if !panel.editingText {
		t.Fatal("startTextEdit should enter editing mode for a literal")
	}
	panel.textBuffer = "mydeck"
	panel.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if panel.newText[SettingTmuxSocketName] != "mydeck" {
		t.Errorf("literal edit should store the buffer, got %q", panel.newText[SettingTmuxSocketName])
	}
}
