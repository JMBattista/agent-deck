package ui

// Data-driven settings registry.
//
// The two-pane Settings panel originally hand-wrote a switch case per setting
// in LoadConfig / GetConfig / adjustValue / toggleValue / renderSettingRow.
// That does not scale: agent-deck's config has dozens of [agent]/[worktree]/
// [tmux]/[ui] fields and adding each new one to the TUI meant touching five
// functions.
//
// This file makes the input system declarative, along the lines requested:
// every setting is one entry in settingDefs categorized as bool / enum /
// int / literal (and literals are further typed as an arbitrary string, a
// filesystem path, or a command). Each entry carries its label, a one-line
// description, the widget metadata for its kind, and a load/save closure that
// binds it to a field on session.UserConfig. The panel's generic dispatch then
// renders and edits any entry purely from its kind — so a brand-new setting
// shows up in the UI just by adding a row here and dropping its SettingType
// into a section.
//
// The original ("legacy") curated settings still use their bespoke handlers in
// settings_panel.go; everything added in the file-only expansion is registered
// here. Both paths share the same render/edit plumbing.

import "github.com/asheshgoplani/agent-deck/internal/session"

// inputKind is the top-level categorization of how a setting is edited.
type inputKind int

const (
	inputBool    inputKind = iota // checkbox (on/off)
	inputEnum                     // radio group over a fixed set of choices
	inputInt                      // bounded number stepped with left/right
	inputLiteral                  // free-form value typed in (see literalKind)
)

// literalKind sub-types a free-form literal so the UI (and future
// string-comprehension helpers like path expansion or command validation) can
// treat them differently. They all edit through the same text widget today.
type literalKind int

const (
	litString    literalKind = iota // arbitrary string
	litDirectory                    // filesystem path (dir or file)
	litCommand                      // shell command / invocation
)

// settingDef is one declarative row in the registry.
type settingDef struct {
	key   SettingType
	kind  inputKind
	lit   literalKind // inputLiteral only
	label string
	desc  string // shown as the dim hint line under the row

	// inputEnum: parallel name/value slices. enumValues are what is stored in
	// config; enumNames are what the radio group displays.
	enumNames  []string
	enumValues []string

	// inputInt: inclusive bounds and the per-step delta applied by left/right.
	min, max, step int
	suffix         string

	// inputLiteral: dim text shown when the value is empty.
	placeholder string

	// load copies the value out of cfg into the panel's value maps.
	// save writes the panel's current value back into cfg.
	load func(s *SettingsPanel, cfg *session.UserConfig)
	save func(s *SettingsPanel, cfg *session.UserConfig)
}

// boolPtr / intPtr return pointers for the *bool / *int config fields whose
// "field absent" state is meaningful. We always materialize an explicit value
// on save: the field is now user-editable, so persisting the chosen value
// (rather than leaving it implicit) is the correct round-trip.
func boolPtr(b bool) *bool { return &b }
func intPtr(i int) *int    { return &i }

// strPtrOrNil maps "" to nil so clearing a literal that overrides a default
// (e.g. worktree path_template) restores the default instead of pinning empty.
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// enumIndex returns the index of v in values, or 0 (the default choice) when
// absent. Used by enum load closures to map a stored string to a radio index.
func enumIndex(values []string, v string) int {
	for i, x := range values {
		if x == v {
			return i
		}
	}
	return 0
}

var (
	footerNames  = []string{"Full", "Curated", "Compact", "Minimal"}
	footerValues = []string{session.FooterFull, session.FooterCurated, session.FooterCompact, session.FooterMinimal}

	itermNames  = []string{"Tab", "Window"}
	itermValues = []string{session.ITermOpenAsTab, session.ITermOpenAsWindow}

	// launchAs "" means unset (defer to launch_in_user_scope).
	launchAsNames  = []string{"Default", "Auto", "Scope", "Service", "Direct"}
	launchAsValues = []string{"", "auto", "scope", "service", "direct"}
)

// boolDef builds a checkbox entry bound to a plain bool config field.
func boolDef(key SettingType, label, desc string,
	get func(*session.UserConfig) bool, set func(*session.UserConfig, bool)) settingDef {
	return settingDef{
		key: key, kind: inputBool, label: label, desc: desc,
		load: func(s *SettingsPanel, c *session.UserConfig) { s.newBool[key] = get(c) },
		save: func(s *SettingsPanel, c *session.UserConfig) { set(c, s.newBool[key]) },
	}
}

// litDef builds a free-form literal entry bound to a string config field.
func litDef(key SettingType, lit literalKind, label, desc, placeholder string,
	get func(*session.UserConfig) string, set func(*session.UserConfig, string)) settingDef {
	return settingDef{
		key: key, kind: inputLiteral, lit: lit, label: label, desc: desc, placeholder: placeholder,
		load: func(s *SettingsPanel, c *session.UserConfig) { s.newText[key] = get(c) },
		save: func(s *SettingsPanel, c *session.UserConfig) { set(c, s.newText[key]) },
	}
}

// enumDef builds a radio entry bound to a string config field.
func enumDef(key SettingType, label, desc string, names, values []string,
	get func(*session.UserConfig) string, set func(*session.UserConfig, string)) settingDef {
	return settingDef{
		key: key, kind: inputEnum, label: label, desc: desc, enumNames: names, enumValues: values,
		load: func(s *SettingsPanel, c *session.UserConfig) { s.newEnum[key] = enumIndex(values, get(c)) },
		save: func(s *SettingsPanel, c *session.UserConfig) {
			idx := s.newEnum[key]
			if idx < 0 || idx >= len(values) {
				idx = 0
			}
			set(c, values[idx])
		},
	}
}

// intDef builds a number entry bound to an int config field.
func intDef(key SettingType, label, desc, suffix string, min, max, step int,
	get func(*session.UserConfig) int, set func(*session.UserConfig, int)) settingDef {
	return settingDef{
		key: key, kind: inputInt, label: label, desc: desc, suffix: suffix, min: min, max: max, step: step,
		load: func(s *SettingsPanel, c *session.UserConfig) { s.newInt[key] = get(c) },
		save: func(s *SettingsPanel, c *session.UserConfig) { set(c, s.newInt[key]) },
	}
}

// settingDefs is the declarative registry covering every file-only config field
// promoted to the TUI in the settings-pane expansion. Order is informational;
// the section list (settingsSections) controls display grouping.
var settingDefs = []settingDef{
	// ---- Interface ([ui]) -------------------------------------------------
	intDef(SettingPreviewPct, "Preview width", "Percent of width given to the preview pane", "%",
		session.MinPreviewPct, session.MaxPreviewPct, 5,
		func(c *session.UserConfig) int { return c.UI.GetPreviewPct() },
		func(c *session.UserConfig, v int) { c.UI.PreviewPct = v }),
	enumDef(SettingFooter, "Footer style", "Bottom hint-bar verbosity", footerNames, footerValues,
		func(c *session.UserConfig) string { return c.UI.GetFooter() },
		func(c *session.UserConfig, v string) { c.UI.Footer = v }),
	enumDef(SettingITermOpenAs, "iTerm open as", "Shift+Enter pop-out target (macOS)", itermNames, itermValues,
		func(c *session.UserConfig) string { return c.UI.GetITermOpenAs() },
		func(c *session.UserConfig, v string) { c.UI.ITermOpenAs = v }),
	boolDef(SettingShowOnlyInstalledTools, "Show only installed tools", "Hide tools whose command is not on PATH",
		func(c *session.UserConfig) bool { return c.UI.ShowOnlyInstalledTools },
		func(c *session.UserConfig, v bool) { c.UI.ShowOnlyInstalledTools = v }),
	boolDef(SettingNewSessionEnterAdvances, "Enter advances new-session fields", "Enter moves to next field (Ctrl+S submits)",
		func(c *session.UserConfig) bool { return c.UI.NewSessionEnterAdvances },
		func(c *session.UserConfig, v bool) { c.UI.NewSessionEnterAdvances = v }),
	intDef(SettingRemoteLatencyRefreshSecs, "Remote latency refresh", "How often remote round-trip latency is measured", "sec",
		2, 300, 1,
		func(c *session.UserConfig) int {
			if c.UI.RemoteLatencyRefreshSecs <= 0 {
				return 5
			}
			return c.UI.RemoteLatencyRefreshSecs
		},
		func(c *session.UserConfig, v int) { c.UI.RemoteLatencyRefreshSecs = v }),
	intDef(SettingRemoteSessionRefreshSecs, "Remote session refresh", "How often the remote session list is re-fetched", "sec",
		session.MinRemoteSessionRefreshSecs, session.MaxRemoteSessionRefreshSecs, 5,
		func(c *session.UserConfig) int { return c.UI.GetRemoteSessionRefreshSecs() },
		func(c *session.UserConfig, v int) { c.UI.RemoteSessionRefreshSecs = v }),

	// ---- Agents — Claude (additional) -------------------------------------
	litDef(SettingClaudeCommand, litCommand, "Command", "CLI command/alias for Claude sessions", "claude",
		func(c *session.UserConfig) string { return c.Claude.Command },
		func(c *session.UserConfig, v string) { c.Claude.Command = v }),
	boolDef(SettingClaudeAllowDangerousMode, "Allow dangerous mode", "Unlock bypass as an option without enabling it",
		func(c *session.UserConfig) bool { return c.Claude.AllowDangerousMode },
		func(c *session.UserConfig, v bool) { c.Claude.AllowDangerousMode = v }),
	boolDef(SettingClaudeAutoMode, "Auto mode", "Classifier reviews commands before they run",
		func(c *session.UserConfig) bool { return c.Claude.AutoMode },
		func(c *session.UserConfig, v bool) { c.Claude.AutoMode = v }),
	litDef(SettingClaudeDefaultModel, litString, "Default model", "Preselected model for new Claude sessions", "(Claude default)",
		func(c *session.UserConfig) string { return c.Claude.DefaultModel },
		func(c *session.UserConfig, v string) { c.Claude.DefaultModel = v }),
	boolDef(SettingClaudeUseChrome, "Use Chrome", "Pass --chrome by default",
		func(c *session.UserConfig) bool { return c.Claude.UseChrome },
		func(c *session.UserConfig, v bool) { c.Claude.UseChrome = v }),
	boolDef(SettingClaudeUseTeammateMode, "Use teammate mode", "Pass --teammate-mode tmux by default",
		func(c *session.UserConfig) bool { return c.Claude.UseTeammateMode },
		func(c *session.UserConfig, v bool) { c.Claude.UseTeammateMode = v }),
	litDef(SettingClaudeEnvFile, litDirectory, "Env file", "Claude-specific .env, sourced after [shell]", "(none)",
		func(c *session.UserConfig) string { return c.Claude.EnvFile },
		func(c *session.UserConfig, v string) { c.Claude.EnvFile = v }),
	boolDef(SettingClaudeHooksEnabled, "Hooks enabled", "Lifecycle hooks for instant status detection",
		func(c *session.UserConfig) bool { return c.Claude.GetHooksEnabled() },
		func(c *session.UserConfig, v bool) { c.Claude.HooksEnabled = boolPtr(v) }),
	boolDef(SettingClaudeAutoResumeSummary, "Auto resume summary", "Auto-confirm the resume-from-summary picker",
		func(c *session.UserConfig) bool { return c.Claude.GetAutoResumeSummary() },
		func(c *session.UserConfig, v bool) { c.Claude.AutoResumeSummary = boolPtr(v) }),
	boolDef(SettingClaudeVimMode, "Vim mode", "Inner prompt uses vim keybindings (insert-guard sends)",
		func(c *session.UserConfig) bool { return c.Claude.VimMode },
		func(c *session.UserConfig, v bool) { c.Claude.VimMode = v }),

	// ---- Agents — Gemini (additional) -------------------------------------
	litDef(SettingGeminiDefaultModel, litString, "Default model", "Model for new Gemini sessions", "(Gemini default)",
		func(c *session.UserConfig) string { return c.Gemini.DefaultModel },
		func(c *session.UserConfig, v string) { c.Gemini.DefaultModel = v }),
	litDef(SettingGeminiEnvFile, litDirectory, "Env file", "Gemini-specific .env", "(none)",
		func(c *session.UserConfig) string { return c.Gemini.EnvFile },
		func(c *session.UserConfig, v string) { c.Gemini.EnvFile = v }),
	litDef(SettingGeminiCommand, litCommand, "Command", "CLI command/invocation for Gemini", "gemini",
		func(c *session.UserConfig) string { return c.Gemini.Command },
		func(c *session.UserConfig, v string) { c.Gemini.Command = v }),

	// ---- Agents — OpenCode ------------------------------------------------
	litDef(SettingOpenCodeDefaultModel, litString, "Default model", "provider/model for new OpenCode sessions", "(OpenCode default)",
		func(c *session.UserConfig) string { return c.OpenCode.DefaultModel },
		func(c *session.UserConfig, v string) { c.OpenCode.DefaultModel = v }),
	litDef(SettingOpenCodeDefaultAgent, litString, "Default agent", "Agent for new OpenCode sessions", "(OpenCode default)",
		func(c *session.UserConfig) string { return c.OpenCode.DefaultAgent },
		func(c *session.UserConfig, v string) { c.OpenCode.DefaultAgent = v }),
	litDef(SettingOpenCodeEnvFile, litDirectory, "Env file", "OpenCode-specific .env", "(none)",
		func(c *session.UserConfig) string { return c.OpenCode.EnvFile },
		func(c *session.UserConfig, v string) { c.OpenCode.EnvFile = v }),
	litDef(SettingOpenCodeCommand, litCommand, "Command", "CLI command/invocation for OpenCode", "opencode",
		func(c *session.UserConfig) string { return c.OpenCode.Command },
		func(c *session.UserConfig, v string) { c.OpenCode.Command = v }),

	// ---- Agents — Codex (additional) --------------------------------------
	litDef(SettingCodexCommand, litCommand, "Command", "CLI command/alias for Codex", "codex",
		func(c *session.UserConfig) string { return c.Codex.Command },
		func(c *session.UserConfig, v string) { c.Codex.Command = v }),
	litDef(SettingCodexConfigDir, litDirectory, "Config dir", "Codex home directory (CODEX_HOME)", "~/.codex (default)",
		func(c *session.UserConfig) string { return c.Codex.ConfigDir },
		func(c *session.UserConfig, v string) { c.Codex.ConfigDir = v }),
	litDef(SettingCodexEnvFile, litDirectory, "Env file", "Codex-specific .env", "(none)",
		func(c *session.UserConfig) string { return c.Codex.EnvFile },
		func(c *session.UserConfig, v string) { c.Codex.EnvFile = v }),

	// ---- Agents — Copilot -------------------------------------------------
	litDef(SettingCopilotEnvFile, litDirectory, "Env file", "Copilot-specific .env", "(none)",
		func(c *session.UserConfig) string { return c.Copilot.EnvFile },
		func(c *session.UserConfig, v string) { c.Copilot.EnvFile = v }),
	litDef(SettingCopilotCommand, litCommand, "Command", "CLI command/invocation for Copilot", "copilot",
		func(c *session.UserConfig) string { return c.Copilot.Command },
		func(c *session.UserConfig, v string) { c.Copilot.Command = v }),
	litDef(SettingCopilotDefaultModel, litString, "Default model", "Model for new Copilot sessions", "(Copilot default)",
		func(c *session.UserConfig) string { return c.Copilot.DefaultModel },
		func(c *session.UserConfig, v string) { c.Copilot.DefaultModel = v }),
	boolDef(SettingCopilotAllowAll, "Allow all", "--allow-all by default (tools, paths, urls)",
		func(c *session.UserConfig) bool { return c.Copilot.AllowAll },
		func(c *session.UserConfig, v bool) { c.Copilot.AllowAll = v }),

	// ---- Agents — Hermes (additional) -------------------------------------
	litDef(SettingHermesCommand, litCommand, "Command", "CLI command/invocation for Hermes", "hermes",
		func(c *session.UserConfig) string { return c.Hermes.Command },
		func(c *session.UserConfig, v string) { c.Hermes.Command = v }),
	litDef(SettingHermesEnvFile, litDirectory, "Env file", "Hermes-specific .env", "(none)",
		func(c *session.UserConfig) string { return c.Hermes.EnvFile },
		func(c *session.UserConfig, v string) { c.Hermes.EnvFile = v }),
	litDef(SettingHermesGatewayURL, litString, "Gateway URL", "WebSocket URL for Hermes gateway health checks", "(none)",
		func(c *session.UserConfig) string { return c.Hermes.GatewayURL },
		func(c *session.UserConfig, v string) { c.Hermes.GatewayURL = v }),
	litDef(SettingHermesDashboardURL, litString, "Dashboard URL", "Hermes dashboard API endpoint", "(none)",
		func(c *session.UserConfig) string { return c.Hermes.DashboardURL },
		func(c *session.UserConfig, v string) { c.Hermes.DashboardURL = v }),
	litDef(SettingHermesAPITokenEnv, litString, "API token env var", "Env var holding the Hermes API token", "HERMES_API_TOKEN",
		func(c *session.UserConfig) string { return c.Hermes.APITokenEnv },
		func(c *session.UserConfig, v string) { c.Hermes.APITokenEnv = v }),
	litDef(SettingHermesWorkspaceDir, litDirectory, "Workspace dir", "Base dir for Hermes shared workspaces", "(temp dir)",
		func(c *session.UserConfig) string { return c.Hermes.WorkspaceDir },
		func(c *session.UserConfig, v string) { c.Hermes.WorkspaceDir = v }),

	// ---- Agents — Crush ---------------------------------------------------
	litDef(SettingCrushCommand, litCommand, "Command", "CLI command/invocation for Crush", "crush",
		func(c *session.UserConfig) string { return c.Crush.Command },
		func(c *session.UserConfig, v string) { c.Crush.Command = v }),
	litDef(SettingCrushEnvFile, litDirectory, "Env file", "Crush-specific .env", "(none)",
		func(c *session.UserConfig) string { return c.Crush.EnvFile },
		func(c *session.UserConfig, v string) { c.Crush.EnvFile = v }),
	boolDef(SettingCrushYoloMode, "YOLO mode", "--yolo: auto-accept all permission prompts",
		func(c *session.UserConfig) bool { return c.Crush.YoloMode },
		func(c *session.UserConfig, v bool) { c.Crush.YoloMode = v }),

	// ---- Worktree ---------------------------------------------------------
	boolDef(SettingWorktreeAutoCleanup, "Auto cleanup", "Remove worktree when its session is deleted",
		func(c *session.UserConfig) bool { return c.Worktree.AutoCleanup },
		func(c *session.UserConfig, v bool) { c.Worktree.AutoCleanup = v }),
	boolDef(SettingWorktreeDefaultEnabled, "Default enabled", "Pre-select worktree creation in new-session/fork",
		func(c *session.UserConfig) bool { return c.Worktree.DefaultEnabled },
		func(c *session.UserConfig, v bool) { c.Worktree.DefaultEnabled = v }),
	litDef(SettingWorktreeDefaultLocation, litDirectory, "Default location", "sibling, subdirectory, or a custom path", "sibling",
		func(c *session.UserConfig) string { return c.Worktree.DefaultLocation },
		func(c *session.UserConfig, v string) { c.Worktree.DefaultLocation = v }),
	litDef(SettingWorktreePathTemplate, litString, "Path template", "Overrides location; vars: {repo-name} {branch} …", "(use location)",
		func(c *session.UserConfig) string {
			if c.Worktree.PathTemplate == nil {
				return ""
			}
			return *c.Worktree.PathTemplate
		},
		func(c *session.UserConfig, v string) { c.Worktree.PathTemplate = strPtrOrNil(v) }),
	litDef(SettingWorktreeBranchPrefix, litString, "Branch prefix", `Prefix for auto branch names (empty disables)`, "feature/",
		func(c *session.UserConfig) string {
			if c.Worktree.BranchPrefix == nil {
				return "feature/"
			}
			return *c.Worktree.BranchPrefix
		},
		func(c *session.UserConfig, v string) { p := v; c.Worktree.BranchPrefix = &p }),
	intDef(SettingWorktreeSetupTimeout, "Setup timeout", "Cap on worktree-setup.sh (0 = unlimited)", "sec",
		0, 3600, 5,
		func(c *session.UserConfig) int {
			if c.Worktree.SetupTimeoutSeconds == nil || *c.Worktree.SetupTimeoutSeconds < 0 {
				return 60
			}
			return *c.Worktree.SetupTimeoutSeconds
		},
		func(c *session.UserConfig, v int) { c.Worktree.SetupTimeoutSeconds = intPtr(v) }),

	// ---- Tmux -------------------------------------------------------------
	boolDef(SettingTmuxInjectStatusLine, "Inject status line", "Inject the agent-deck tmux status bar",
		func(c *session.UserConfig) bool { return c.Tmux.GetInjectStatusLine() },
		func(c *session.UserConfig, v bool) { c.Tmux.InjectStatusLine = boolPtr(v) }),
	boolDef(SettingTmuxMouse, "Mouse mode", "Enable tmux mouse mode on new sessions",
		func(c *session.UserConfig) bool { return c.Tmux.GetMouse() },
		func(c *session.UserConfig, v bool) { c.Tmux.Mouse = boolPtr(v) }),
	boolDef(SettingTmuxLaunchInUserScope, "Launch in user scope", "Run tmux under the systemd --user manager",
		func(c *session.UserConfig) bool { return c.Tmux.GetLaunchInUserScope() },
		func(c *session.UserConfig, v bool) { c.Tmux.LaunchInUserScope = boolPtr(v) }),
	enumDef(SettingTmuxLaunchAs, "Launch as", "Spawn form for new tmux servers", launchAsNames, launchAsValues,
		func(c *session.UserConfig) string { return c.Tmux.GetLaunchAs() },
		func(c *session.UserConfig, v string) {
			// "" clears the override (defer to launch_in_user_scope).
			if v == "" {
				c.Tmux.LaunchAs = nil
				return
			}
			p := v
			c.Tmux.LaunchAs = &p
		}),
	litDef(SettingTmuxWindowStyleOverride, litString, "Window style override", `tmux window-style; "default" shows terminal bg`, "(theme)",
		func(c *session.UserConfig) string { return c.Tmux.WindowStyleOverride },
		func(c *session.UserConfig, v string) { c.Tmux.WindowStyleOverride = v }),
	boolDef(SettingTmuxClearOnRestart, "Clear on restart", "Wipe scrollback when a session is restarted",
		func(c *session.UserConfig) bool { return c.Tmux.ClearOnRestart },
		func(c *session.UserConfig, v bool) { c.Tmux.ClearOnRestart = v }),
	litDef(SettingTmuxDetachKey, litString, "Detach key", `PTY-attach detach key, e.g. "ctrl+d"`, "ctrl+q (default)",
		func(c *session.UserConfig) string { return c.Tmux.DetachKey },
		func(c *session.UserConfig, v string) { c.Tmux.DetachKey = v }),
	litDef(SettingTmuxSocketName, litString, "Socket name", "Isolate agent-deck onto its own tmux server", "(shared)",
		func(c *session.UserConfig) string { return c.Tmux.SocketName },
		func(c *session.UserConfig, v string) { c.Tmux.SocketName = v }),

	// ---- Instances --------------------------------------------------------
	boolDef(SettingInstancesAllowMultiple, "Allow multiple instances", "Run multiple TUI instances per profile",
		func(c *session.UserConfig) bool { return c.Instances.GetAllowMultiple() },
		func(c *session.UserConfig, v bool) { c.Instances.AllowMultiple = boolPtr(v) }),
	boolDef(SettingInstancesFollowCwdOnAttach, "Follow cwd on attach", "Persist pane cwd as the session path after attach",
		func(c *session.UserConfig) bool { return c.Instances.GetFollowCwdOnAttach() },
		func(c *session.UserConfig, v bool) { c.Instances.FollowCwdOnAttach = boolPtr(v) }),
}

// settingDefByKey indexes settingDefs for O(1) lookup by SettingType. Built
// once at init. A setting present here is "registry-driven"; one absent uses
// the legacy bespoke handlers in settings_panel.go.
var settingDefByKey = func() map[SettingType]*settingDef {
	m := make(map[SettingType]*settingDef, len(settingDefs))
	for i := range settingDefs {
		m[settingDefs[i].key] = &settingDefs[i]
	}
	return m
}()

// defFor returns the registry entry for key, or nil if the setting uses the
// legacy bespoke handlers.
func defFor(key SettingType) *settingDef {
	return settingDefByKey[key]
}
