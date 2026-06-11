package ui

import (
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// SettingType identifies which setting is being edited
type SettingType int

const (
	SettingTheme SettingType = iota // Theme must be first (index 0)
	SettingDefaultTool
	SettingDangerousMode
	SettingClaudeConfigDir
	SettingGeminiYoloMode
	SettingCodexYoloMode
	SettingHermesYoloMode
	SettingCheckForUpdates
	SettingAutoUpdate
	SettingLogMaxSize
	SettingLogMaxLines
	SettingRemoveOrphans
	SettingGlobalSearchEnabled
	SettingSearchTier
	SettingRecentDays
	SettingShowOutput
	SettingShowAnalytics
	SettingShowNotes
	SettingNotesOutputSplit
	SettingMaintenanceEnabled
	SettingStatsEnabled
	SettingStatsRefresh
	SettingStatsFormat
	SettingStatsShowCPU
	SettingStatsShowRAM
	SettingStatsShowDisk
	SettingStatsShowNetwork
	SettingStatsShowGPU
	SettingStatsShowLoad
	SettingSyncTitle
	SettingShowSessionTimestamps

	// --- Registry-driven settings (see settings_defs.go). These were
	// previously editable only by hand-editing config.toml; they are wired into
	// the TUI through the declarative settingDefs table rather than bespoke
	// switch cases. Appending here keeps the legacy indices (0..30) stable.

	// Interface ([ui])
	SettingPreviewPct
	SettingFooter
	SettingITermOpenAs
	SettingShowOnlyInstalledTools
	SettingNewSessionEnterAdvances
	SettingRemoteLatencyRefreshSecs
	SettingRemoteSessionRefreshSecs

	// Agents — Claude (additional)
	SettingClaudeCommand
	SettingClaudeAllowDangerousMode
	SettingClaudeAutoMode
	SettingClaudeDefaultModel
	SettingClaudeUseChrome
	SettingClaudeUseTeammateMode
	SettingClaudeEnvFile
	SettingClaudeHooksEnabled
	SettingClaudeAutoResumeSummary
	SettingClaudeVimMode

	// Agents — Gemini (additional)
	SettingGeminiDefaultModel
	SettingGeminiEnvFile
	SettingGeminiCommand

	// Agents — OpenCode
	SettingOpenCodeDefaultModel
	SettingOpenCodeDefaultAgent
	SettingOpenCodeEnvFile
	SettingOpenCodeCommand

	// Agents — Codex (additional)
	SettingCodexCommand
	SettingCodexConfigDir
	SettingCodexEnvFile

	// Agents — Copilot
	SettingCopilotEnvFile
	SettingCopilotCommand
	SettingCopilotDefaultModel
	SettingCopilotAllowAll

	// Agents — Hermes (additional)
	SettingHermesCommand
	SettingHermesEnvFile
	SettingHermesGatewayURL
	SettingHermesDashboardURL
	SettingHermesAPITokenEnv
	SettingHermesWorkspaceDir

	// Agents — Crush
	SettingCrushCommand
	SettingCrushEnvFile
	SettingCrushYoloMode

	// Worktree
	SettingWorktreeAutoCleanup
	SettingWorktreeDefaultEnabled
	SettingWorktreeDefaultLocation
	SettingWorktreePathTemplate
	SettingWorktreeBranchPrefix
	SettingWorktreeSetupTimeout

	// Tmux
	SettingTmuxInjectStatusLine
	SettingTmuxMouse
	SettingTmuxLaunchInUserScope
	SettingTmuxLaunchAs
	SettingTmuxWindowStyleOverride
	SettingTmuxClearOnRestart
	SettingTmuxDetachKey
	SettingTmuxSocketName

	// Instances
	SettingInstancesAllowMultiple
	SettingInstancesFollowCwdOnAttach
)

// Total number of navigable settings (legacy 31 + registry-driven expansion).
const settingsCount = 87

// settingsSection groups related settings for the left (master) pane. Each
// section's settings are shown in the right (detail) pane when the section is
// selected. This is a pure reorganization of the existing settingsCount
// settings — no config keys or on-disk schema change.
type settingsSection struct {
	title    string
	settings []SettingType
	info     bool // informational only (no editable settings, e.g. MCP & Tools)
}

// settingsSections is the master list rendered in the left pane. Together the
// editable sections must cover exactly the settingsCount SettingType entries,
// each appearing in exactly one section (verified by TestSettingsSectionsCoverAllSettings).
var settingsSections = []settingsSection{
	{title: "Appearance", settings: []SettingType{
		SettingTheme,
		SettingShowOutput,
		SettingShowAnalytics,
		SettingShowNotes,
		SettingNotesOutputSplit,
	}},
	{title: "Interface", settings: []SettingType{
		SettingPreviewPct,
		SettingFooter,
		SettingITermOpenAs,
		SettingShowOnlyInstalledTools,
		SettingNewSessionEnterAdvances,
		SettingRemoteLatencyRefreshSecs,
		SettingRemoteSessionRefreshSecs,
	}},
	{title: "Agents — Shared", settings: []SettingType{
		SettingDefaultTool,
		SettingDangerousMode,
	}},
	{title: "Agents — Claude", settings: []SettingType{
		SettingClaudeConfigDir,
		SettingClaudeCommand,
		SettingClaudeDefaultModel,
		SettingClaudeEnvFile,
		SettingClaudeAllowDangerousMode,
		SettingClaudeAutoMode,
		SettingClaudeUseChrome,
		SettingClaudeUseTeammateMode,
		SettingClaudeHooksEnabled,
		SettingClaudeAutoResumeSummary,
		SettingClaudeVimMode,
	}},
	{title: "Agents — Gemini", settings: []SettingType{
		SettingGeminiYoloMode,
		SettingGeminiDefaultModel,
		SettingGeminiEnvFile,
		SettingGeminiCommand,
	}},
	{title: "Agents — OpenCode", settings: []SettingType{
		SettingOpenCodeDefaultModel,
		SettingOpenCodeDefaultAgent,
		SettingOpenCodeEnvFile,
		SettingOpenCodeCommand,
	}},
	{title: "Agents — Codex", settings: []SettingType{
		SettingCodexYoloMode,
		SettingCodexCommand,
		SettingCodexConfigDir,
		SettingCodexEnvFile,
	}},
	{title: "Agents — Copilot", settings: []SettingType{
		SettingCopilotDefaultModel,
		SettingCopilotEnvFile,
		SettingCopilotCommand,
		SettingCopilotAllowAll,
	}},
	{title: "Agents — Hermes", settings: []SettingType{
		SettingHermesYoloMode,
		SettingHermesCommand,
		SettingHermesEnvFile,
		SettingHermesGatewayURL,
		SettingHermesDashboardURL,
		SettingHermesAPITokenEnv,
		SettingHermesWorkspaceDir,
	}},
	{title: "Agents — Crush", settings: []SettingType{
		SettingCrushCommand,
		SettingCrushEnvFile,
		SettingCrushYoloMode,
	}},
	{title: "Worktree", settings: []SettingType{
		SettingWorktreeAutoCleanup,
		SettingWorktreeDefaultEnabled,
		SettingWorktreeDefaultLocation,
		SettingWorktreePathTemplate,
		SettingWorktreeBranchPrefix,
		SettingWorktreeSetupTimeout,
	}},
	{title: "Tmux", settings: []SettingType{
		SettingTmuxInjectStatusLine,
		SettingTmuxMouse,
		SettingTmuxLaunchInUserScope,
		SettingTmuxLaunchAs,
		SettingTmuxWindowStyleOverride,
		SettingTmuxClearOnRestart,
		SettingTmuxDetachKey,
		SettingTmuxSocketName,
	}},
	{title: "Instances", settings: []SettingType{
		SettingInstancesAllowMultiple,
		SettingInstancesFollowCwdOnAttach,
	}},
	{title: "Updates", settings: []SettingType{
		SettingCheckForUpdates,
		SettingAutoUpdate,
	}},
	{title: "Logs", settings: []SettingType{
		SettingLogMaxSize,
		SettingLogMaxLines,
	}},
	{title: "Search", settings: []SettingType{
		SettingGlobalSearchEnabled,
		SettingSearchTier,
		SettingRecentDays,
	}},
	{title: "Maintenance", settings: []SettingType{
		SettingRemoveOrphans,
		SettingMaintenanceEnabled,
	}},
	{title: "System Stats", settings: []SettingType{
		SettingStatsEnabled,
		SettingStatsRefresh,
		SettingStatsFormat,
		SettingStatsShowCPU,
		SettingStatsShowRAM,
		SettingStatsShowDisk,
		SettingStatsShowNetwork,
		SettingStatsShowGPU,
		SettingStatsShowLoad,
	}},
	{title: "Sessions", settings: []SettingType{
		SettingSyncTitle,
	}},
	{title: "Display", settings: []SettingType{
		SettingShowSessionTimestamps,
	}},
	{title: "MCP & Tools", info: true},
}

// adjustableSettings are the radio/number settings that left/right (h/l) tune
// in-place in the detail pane. Toggles and text fields are not in this set, so
// left/right over them moves focus back to the section list instead.
var adjustableSettings = map[SettingType]bool{
	SettingTheme:            true,
	SettingDefaultTool:      true,
	SettingSearchTier:       true,
	SettingLogMaxSize:       true,
	SettingLogMaxLines:      true,
	SettingRecentDays:       true,
	SettingNotesOutputSplit: true,
	SettingStatsRefresh:     true,
	SettingStatsFormat:      true,
}

// SettingsPanel displays and edits user configuration
type SettingsPanel struct {
	visible       bool
	width         int
	height        int
	cursor        int  // Current setting index within the selected section (a SettingType)
	sectionCursor int  // Selected section index into settingsSections (left pane)
	focusRight    bool // false = section list (left pane) focused, true = detail (right pane)
	scrollOffset  int  // Scroll offset when content overflows terminal height
	profile       string

	// Dynamic tool lists (built-in + custom tools from config)
	toolNames  []string
	toolValues []string

	// Setting values
	selectedTheme       int // 0=dark, 1=light, 2=system
	selectedTool        int // index into toolNames/toolValues
	dangerousMode       bool
	claudeConfigDir     string
	claudeConfigIsScope bool // true = profile override, false = global [claude]
	geminiYoloMode      bool
	codexYoloMode       bool
	hermesYoloMode      bool
	checkForUpdates     bool
	autoUpdate          bool
	logMaxSizeMB        int
	logMaxLines         int
	removeOrphans       bool
	globalSearchEnabled bool
	searchTier          int // 0=auto, 1=instant, 2=balanced
	recentDays          int
	showOutput          bool
	showAnalytics       bool
	showNotes           bool
	syncTitle           bool // global: let the agent rename the session (off = keep your title)
	notesOutputSplit    int  // percentage 10-90 (displayed as %, stored as 0.10-0.90)
	maintenanceEnabled  bool
	statsEnabled        bool
	statsRefreshSecs    int
	statsFormat         int // 0=compact, 1=full, 2=minimal
	statsShowCPU        bool
	statsShowRAM        bool
	statsShowDisk       bool
	statsShowNetwork    bool
	statsShowGPU        bool
	statsShowLoad       bool

	showSessionTimestamps bool

	// Registry-driven value stores (settings_defs.go). Keyed by SettingType.
	// Legacy settings keep their dedicated fields above; everything in the
	// settingDefs table is backed by these generic maps.
	newBool map[SettingType]bool
	newInt  map[SettingType]int
	newEnum map[SettingType]int // selected radio index
	newText map[SettingType]string

	// Text input state
	editingText bool
	textBuffer  string

	// Track if global search settings changed (requires restart)
	needsRestart bool

	// Original config for detecting changes
	originalConfig *session.UserConfig
}

// builtinToolNames and builtinToolValues are the built-in tools. Custom tools
// from config are appended dynamically in LoadConfig.
var (
	builtinToolNames  = []string{"Claude", "Gemini", "OpenCode", "Codex", "Pi", "Copilot", "Crush", "Cursor", "Hermes"}
	builtinToolValues = []string{"claude", "gemini", "opencode", "codex", "pi", "copilot", "crush", "cursor", "hermes"}
)

// Search tier names for radio selection
var (
	tierNames  = []string{"Auto", "Instant", "Balanced"}
	tierValues = []string{"auto", "instant", "balanced"}
)

// Theme names for radio selection
var (
	themeNames  = []string{"Dark", "Light", "System"}
	themeValues = []string{"dark", "light", "system"}
)

// Stats format names for radio selection
var (
	statsFormatNames  = []string{"Compact", "Full", "Minimal"}
	statsFormatValues = []string{"compact", "full", "minimal"}
)

// NewSettingsPanel creates a new settings panel
func NewSettingsPanel() *SettingsPanel {
	return &SettingsPanel{
		newBool:             make(map[SettingType]bool),
		newInt:              make(map[SettingType]int),
		newEnum:             make(map[SettingType]int),
		newText:             make(map[SettingType]string),
		toolNames:           append(append([]string{}, builtinToolNames...), "None"),
		toolValues:          append(append([]string{}, builtinToolValues...), ""),
		logMaxSizeMB:        10,
		logMaxLines:         10000,
		removeOrphans:       true,
		checkForUpdates:     true,
		globalSearchEnabled: true,
		recentDays:          90,
		showOutput:          true,  // Default: output ON (shows launch animation)
		showAnalytics:       false, // Default: analytics OFF (opt-in)
		syncTitle:           true,  // Default: title sync ON (matches GetSyncTitle default)
		notesOutputSplit:    33,    // Default: 33%
		statsEnabled:        true,  // Default: stats ON
		statsRefreshSecs:    5,     // Default: 5 seconds
		statsShowCPU:        true,
		statsShowRAM:        true,
		statsShowDisk:       true,
		statsShowNetwork:    true,
	}
}

// Show displays the settings panel and loads current config
func (s *SettingsPanel) Show() {
	s.visible = true
	s.sectionCursor = 0
	s.focusRight = false
	s.cursor = int(settingsSections[0].settings[0])
	s.scrollOffset = 0
	s.editingText = false
	s.needsRestart = false

	// Load current config
	config, _ := session.LoadUserConfig()
	if config != nil {
		s.LoadConfig(config)
		s.originalConfig = config
	}
}

// Hide hides the settings panel
func (s *SettingsPanel) Hide() {
	s.visible = false
	s.editingText = false
}

// IsVisible returns whether the panel is visible
func (s *SettingsPanel) IsVisible() bool {
	return s.visible
}

// NeedsRestart returns true if changes require a restart
func (s *SettingsPanel) NeedsRestart() bool {
	return s.needsRestart
}

// ScrollUp moves the cursor up by one within the focused pane (mouse wheel support).
func (s *SettingsPanel) ScrollUp() {
	if !s.visible {
		return
	}
	if s.focusRight {
		s.moveDetail(-1)
	} else {
		s.moveSection(-1)
	}
}

// ScrollDown moves the cursor down by one within the focused pane (mouse wheel support).
func (s *SettingsPanel) ScrollDown() {
	if !s.visible {
		return
	}
	if s.focusRight {
		s.moveDetail(1)
	} else {
		s.moveSection(1)
	}
}

// currentSection returns the section selected in the left pane.
func (s *SettingsPanel) currentSection() settingsSection {
	if s.sectionCursor < 0 || s.sectionCursor >= len(settingsSections) {
		return settingsSections[0]
	}
	return settingsSections[s.sectionCursor]
}

// moveSection changes the selected section (left pane) and snaps the detail
// cursor to that section's first editable setting.
func (s *SettingsPanel) moveSection(delta int) {
	newIdx := s.sectionCursor + delta
	if newIdx < 0 || newIdx >= len(settingsSections) {
		return
	}
	s.sectionCursor = newIdx
	s.scrollOffset = 0
	if settings := settingsSections[newIdx].settings; len(settings) > 0 {
		s.cursor = int(settings[0])
	}
}

// moveDetail moves the cursor within the current section's settings (right pane).
func (s *SettingsPanel) moveDetail(delta int) {
	settings := s.currentSection().settings
	if len(settings) == 0 {
		return
	}
	pos := 0
	for i, st := range settings {
		if int(st) == s.cursor {
			pos = i
			break
		}
	}
	newPos := pos + delta
	if newPos < 0 || newPos >= len(settings) {
		return
	}
	s.cursor = int(settings[newPos])
}

// enterRightPane focuses the detail pane, snapping the cursor to a valid
// setting in the current section. Informational sections stay on the left.
func (s *SettingsPanel) enterRightPane() {
	settings := s.currentSection().settings
	if len(settings) == 0 {
		return
	}
	inSection := false
	for _, st := range settings {
		if int(st) == s.cursor {
			inSection = true
			break
		}
	}
	if !inSection {
		s.cursor = int(settings[0])
	}
	s.focusRight = true
}

// isAdjustable reports whether the current detail setting is tuned with left/right.
// Legacy radios/numbers are listed in adjustableSettings; registry enum/int
// entries are adjustable by virtue of their kind.
func (s *SettingsPanel) isAdjustable() bool {
	key := SettingType(s.cursor)
	if adjustableSettings[key] {
		return true
	}
	if def := defFor(key); def != nil {
		return def.kind == inputEnum || def.kind == inputInt
	}
	return false
}

// focusSetting positions the panel on the given setting: selects its section,
// places the detail cursor on it, and focuses the right pane. Used by tests and
// any caller that wants to jump straight to a specific setting.
func (s *SettingsPanel) focusSetting(setting SettingType) {
	for si, sec := range settingsSections {
		for _, st := range sec.settings {
			if st == setting {
				s.sectionCursor = si
				s.cursor = int(setting)
				s.focusRight = true
				return
			}
		}
	}
}

// SetSize sets the panel dimensions
func (s *SettingsPanel) SetSize(width, height int) {
	s.width = width
	s.height = height
}

// SetProfile sets the active profile for profile-aware settings.
func (s *SettingsPanel) SetProfile(profile string) {
	s.profile = profile
}

// LoadConfig populates panel values from a UserConfig
func (s *SettingsPanel) LoadConfig(config *session.UserConfig) {
	// Load theme
	switch config.Theme {
	case "light":
		s.selectedTheme = 1
	case "system":
		s.selectedTheme = 2
	default:
		s.selectedTheme = 0
	}

	// Rebuild tool lists: built-ins + custom tools + "None".
	s.buildToolLists(config)

	// Default tool
	s.selectedTool = len(s.toolValues) - 1 // None by default
	for i, val := range s.toolValues {
		if val == config.DefaultTool {
			s.selectedTool = i
			break
		}
	}

	// Claude settings
	s.dangerousMode = config.Claude.GetDangerousMode()
	s.claudeConfigDir = config.Claude.ConfigDir
	s.claudeConfigIsScope = false
	if s.profile != "" && config.Profiles != nil {
		if profileCfg, ok := config.Profiles[s.profile]; ok && profileCfg.Claude.ConfigDir != "" {
			s.claudeConfigDir = profileCfg.Claude.ConfigDir
			s.claudeConfigIsScope = true
		}
	}

	// Gemini settings
	s.geminiYoloMode = config.Gemini.YoloMode

	// Codex settings
	s.codexYoloMode = config.Codex.YoloMode

	// Hermes settings
	s.hermesYoloMode = config.Hermes.YoloMode

	// Update settings
	s.checkForUpdates = config.Updates.CheckEnabled
	s.autoUpdate = config.Updates.AutoUpdate

	// Log settings
	s.logMaxSizeMB = config.Logs.MaxSizeMB
	if s.logMaxSizeMB <= 0 {
		s.logMaxSizeMB = 10
	}
	s.logMaxLines = config.Logs.MaxLines
	if s.logMaxLines <= 0 {
		s.logMaxLines = 10000
	}
	s.removeOrphans = config.Logs.RemoveOrphans

	// Global search settings
	s.globalSearchEnabled = config.GlobalSearch.Enabled
	s.searchTier = 0 // auto by default
	for i, val := range tierValues {
		if val == config.GlobalSearch.Tier {
			s.searchTier = i
			break
		}
	}
	s.recentDays = config.GlobalSearch.RecentDays
	if s.recentDays < 0 {
		s.recentDays = 90
	}

	// Preview settings
	s.showOutput = config.GetShowOutput()
	s.showAnalytics = config.GetShowAnalytics()
	s.showNotes = config.GetShowNotes()

	// Session settings
	s.syncTitle = config.GetSyncTitle()
	split := config.Preview.GetNotesOutputSplit()
	s.notesOutputSplit = int(split * 100)
	if s.notesOutputSplit < 10 {
		s.notesOutputSplit = 10
	} else if s.notesOutputSplit > 90 {
		s.notesOutputSplit = 90
	}

	// Maintenance settings.
	s.maintenanceEnabled = config.Maintenance.Enabled

	// System stats settings
	s.statsEnabled = config.SystemStats.GetEnabled()
	s.statsRefreshSecs = config.SystemStats.GetRefreshSeconds()
	s.statsFormat = 0 // compact by default
	for i, val := range statsFormatValues {
		if val == config.SystemStats.GetFormat() {
			s.statsFormat = i
			break
		}
	}
	showSet := make(map[string]bool)
	for _, stat := range config.SystemStats.GetShow() {
		showSet[stat] = true
	}
	s.statsShowCPU = showSet["cpu"]
	s.statsShowRAM = showSet["ram"]
	s.statsShowDisk = showSet["disk"]
	s.statsShowNetwork = showSet["network"]
	s.statsShowGPU = showSet["gpu"]
	s.statsShowLoad = showSet["load"]

	// Display settings
	s.showSessionTimestamps = config.Display.ShowSessionTimestamps

	// Registry-driven settings (settings_defs.go): each entry binds itself to
	// a config field via its load closure.
	s.ensureValueMaps()
	for i := range settingDefs {
		settingDefs[i].load(s, config)
	}
}

// ensureValueMaps lazily allocates the registry value stores so LoadConfig /
// GetConfig are safe even if the panel was constructed without NewSettingsPanel.
func (s *SettingsPanel) ensureValueMaps() {
	if s.newBool == nil {
		s.newBool = make(map[SettingType]bool)
	}
	if s.newInt == nil {
		s.newInt = make(map[SettingType]int)
	}
	if s.newEnum == nil {
		s.newEnum = make(map[SettingType]int)
	}
	if s.newText == nil {
		s.newText = make(map[SettingType]string)
	}
}

func (s *SettingsPanel) buildToolLists(config *session.UserConfig) {
	names := append([]string{}, builtinToolNames...)
	values := append([]string{}, builtinToolValues...)

	if len(config.Tools) > 0 {
		builtins := map[string]bool{
			"claude": true, "gemini": true, "opencode": true,
			"codex": true, "pi": true, "crush": true, "copilot": true,
			"shell": true, "cursor": true, "aider": true, "hermes": true,
		}
		var custom []string
		for name := range config.Tools {
			if !builtins[name] {
				custom = append(custom, name)
			}
		}
		sort.Strings(custom)
		for _, name := range custom {
			display := strings.ToUpper(name[:1]) + name[1:]
			names = append(names, display)
			values = append(values, name)
		}
	}

	names = append(names, "None")
	values = append(values, "")

	s.toolNames = names
	s.toolValues = values
}

// GetConfig returns a UserConfig with current panel values
func (s *SettingsPanel) GetConfig() *session.UserConfig {
	config := &session.UserConfig{
		DefaultTool: "",
		Tools:       make(map[string]session.ToolDef),
		MCPs:        make(map[string]session.MCPDef),
	}

	// Base-copy the sub-structs the panel now partially edits so fields NOT
	// surfaced in the TUI (Claude.ExtraArgs, Tmux.Options, Worktree internals,
	// etc.) survive a save. The explicit legacy writes and the registry save
	// loop below overwrite only the exposed fields. Same #710/#584 preservation
	// contract, applied per-struct now that these tables are partly editable.
	if s.originalConfig != nil {
		config.Claude = s.originalConfig.Claude
		config.Gemini = s.originalConfig.Gemini
		config.OpenCode = s.originalConfig.OpenCode
		config.Codex = s.originalConfig.Codex
		config.Copilot = s.originalConfig.Copilot
		config.Hermes = s.originalConfig.Hermes
		config.Crush = s.originalConfig.Crush
		config.Worktree = s.originalConfig.Worktree
		config.Tmux = s.originalConfig.Tmux
		config.UI = s.originalConfig.UI
		config.Instances = s.originalConfig.Instances
	}

	// Theme
	if s.selectedTheme < len(themeValues) {
		config.Theme = themeValues[s.selectedTheme]
	}

	// Default tool
	if s.selectedTool >= 0 && s.selectedTool < len(s.toolValues) {
		config.DefaultTool = s.toolValues[s.selectedTool]
	}

	// Claude settings
	dangerousModeVal := s.dangerousMode
	config.Claude.DangerousMode = &dangerousModeVal
	if !s.claudeConfigIsScope {
		config.Claude.ConfigDir = s.claudeConfigDir
	}

	// Gemini settings
	config.Gemini.YoloMode = s.geminiYoloMode

	// Codex settings
	config.Codex.YoloMode = s.codexYoloMode

	// Hermes settings
	config.Hermes.YoloMode = s.hermesYoloMode

	// Update settings
	config.Updates.CheckEnabled = s.checkForUpdates
	config.Updates.AutoUpdate = s.autoUpdate

	// Log settings
	config.Logs.MaxSizeMB = s.logMaxSizeMB
	config.Logs.MaxLines = s.logMaxLines
	config.Logs.RemoveOrphans = s.removeOrphans

	// Global search settings
	config.GlobalSearch.Enabled = s.globalSearchEnabled
	if s.searchTier >= 0 && s.searchTier < len(tierValues) {
		config.GlobalSearch.Tier = tierValues[s.searchTier]
	}
	config.GlobalSearch.RecentDays = s.recentDays

	// Preview settings
	showOutput := s.showOutput
	config.Preview.ShowOutput = &showOutput
	showAnalytics := s.showAnalytics
	config.Preview.ShowAnalytics = &showAnalytics
	showNotes := s.showNotes
	config.Preview.ShowNotes = &showNotes
	config.Preview.NotesOutputSplit = float64(s.notesOutputSplit) / 100.0

	// Session settings
	syncTitle := s.syncTitle
	config.SyncTitle = &syncTitle

	// Maintenance settings.
	config.Maintenance.Enabled = s.maintenanceEnabled

	// System stats settings
	statsEnabled := s.statsEnabled
	config.SystemStats.Enabled = &statsEnabled
	config.SystemStats.RefreshSeconds = s.statsRefreshSecs
	if s.statsFormat >= 0 && s.statsFormat < len(statsFormatValues) {
		config.SystemStats.Format = statsFormatValues[s.statsFormat]
	}
	var showStats []string
	if s.statsShowCPU {
		showStats = append(showStats, "cpu")
	}
	if s.statsShowRAM {
		showStats = append(showStats, "ram")
	}
	if s.statsShowDisk {
		showStats = append(showStats, "disk")
	}
	if s.statsShowNetwork {
		showStats = append(showStats, "network")
	}
	if s.statsShowGPU {
		showStats = append(showStats, "gpu")
	}
	if s.statsShowLoad {
		showStats = append(showStats, "load")
	}
	config.SystemStats.Show = showStats

	// Display settings
	config.Display.ShowSessionTimestamps = s.showSessionTimestamps

	// Preserve original MCPs, Tools, and Docker settings.
	if s.originalConfig != nil {
		config.MCPs = s.originalConfig.MCPs
		config.Tools = s.originalConfig.Tools
		config.MCPPool = s.originalConfig.MCPPool
		config.Docker = s.originalConfig.Docker
		config.Preview.Analytics = s.originalConfig.Preview.Analytics
		config.Profiles = s.originalConfig.Profiles
		// Worktree and Tmux are base-copied above and their exposed fields are
		// re-applied by the registry save loop below; non-exposed siblings
		// (Tmux.Options, Worktree path internals) ride through on the base copy.
		// Fork settings are not exposed in SettingsPanel; preserve the whole
		// [fork] table so saving visible settings cannot reset quick-fork defaults.
		config.Fork = s.originalConfig.Fork
		// Keep global Claude config when editing profile-specific override.
		if s.claudeConfigIsScope {
			config.Claude.ConfigDir = s.originalConfig.Claude.ConfigDir
		}
	}

	// Apply profile-specific Claude override after original profile map is restored.
	if s.claudeConfigIsScope && s.profile != "" {
		if config.Profiles == nil {
			config.Profiles = make(map[string]session.ProfileSettings)
		}
		profileCfg := config.Profiles[s.profile]
		profileCfg.Claude.ConfigDir = s.claudeConfigDir
		config.Profiles[s.profile] = profileCfg
	}

	// Registry-driven settings (settings_defs.go): write current values back
	// onto the base-copied sub-structs. Runs last so exposed fields win while
	// hidden sibling fields (preserved by the base copy) ride through.
	s.ensureValueMaps()
	for i := range settingDefs {
		settingDefs[i].save(s, config)
	}

	return config
}

// Update handles input and returns (panel, cmd, valueChanged)
func (s *SettingsPanel) Update(msg tea.KeyMsg) (*SettingsPanel, tea.Cmd, bool) {
	if !s.visible {
		return s, nil, false
	}

	// Handle text editing mode
	if s.editingText {
		return s.handleTextEdit(msg)
	}

	valueChanged := false
	key := msg.String()

	switch key {
	case "esc", "S":
		s.Hide()
		return s, nil, false

	case "tab", "shift+tab":
		// Toggle focus between the section list and the detail pane.
		if s.focusRight {
			s.focusRight = false
		} else {
			s.enterRightPane()
		}

	case "up", "k":
		if s.focusRight {
			s.moveDetail(-1)
		} else {
			s.moveSection(-1)
		}

	case "down", "j":
		if s.focusRight {
			s.moveDetail(1)
		} else {
			s.moveSection(1)
		}

	case "left", "h":
		// In the detail pane, left tunes adjustable values; on a toggle/text
		// setting it steps focus back to the section list.
		if s.focusRight {
			if s.isAdjustable() {
				valueChanged = s.adjustValue(-1)
			} else {
				s.focusRight = false
			}
		}

	case "right", "l":
		// From the section list, right enters the detail pane. In the detail
		// pane, right tunes adjustable values.
		if !s.focusRight {
			s.enterRightPane()
		} else if s.isAdjustable() {
			valueChanged = s.adjustValue(1)
		}

	case " ":
		if s.focusRight {
			valueChanged = s.toggleValue()
		} else {
			// Space from the section list opens the section.
			s.enterRightPane()
		}

	case "enter":
		if !s.focusRight {
			s.enterRightPane()
		} else if s.isTextSetting() {
			s.startTextEdit()
		}
	}

	return s, nil, valueChanged
}

// adjustValue changes a radio or number value by delta
func (s *SettingsPanel) adjustValue(delta int) bool {
	setting := SettingType(s.cursor)

	// Registry-driven enum/int settings tune generically from their metadata.
	if def := defFor(setting); def != nil {
		switch def.kind {
		case inputEnum:
			newVal := s.newEnum[setting] + delta
			if newVal >= 0 && newVal < len(def.enumValues) {
				s.newEnum[setting] = newVal
				return true
			}
			return false
		case inputInt:
			step := def.step
			if step == 0 {
				step = 1
			}
			newVal := s.newInt[setting] + delta*step
			if newVal < def.min {
				newVal = def.min
			}
			if newVal > def.max {
				newVal = def.max
			}
			if newVal != s.newInt[setting] {
				s.newInt[setting] = newVal
				return true
			}
			return false
		}
	}

	changed := false

	switch setting {
	case SettingTheme:
		newVal := s.selectedTheme + delta
		if newVal >= 0 && newVal < len(themeNames) {
			s.selectedTheme = newVal
			changed = true
		}

	case SettingDefaultTool:
		newVal := s.selectedTool + delta
		if newVal >= 0 && newVal < len(s.toolNames) {
			s.selectedTool = newVal
			changed = true
		}

	case SettingSearchTier:
		newVal := s.searchTier + delta
		if newVal >= 0 && newVal < len(tierNames) {
			oldTier := s.searchTier
			s.searchTier = newVal
			changed = true
			if oldTier != newVal {
				s.needsRestart = true
			}
		}

	case SettingLogMaxSize:
		newVal := s.logMaxSizeMB + delta
		if newVal >= 1 {
			s.logMaxSizeMB = newVal
			changed = true
		}

	case SettingLogMaxLines:
		// Adjust by 1000 for lines
		newVal := s.logMaxLines + (delta * 1000)
		if newVal >= 1000 {
			s.logMaxLines = newVal
			changed = true
		}

	case SettingRecentDays:
		newVal := s.recentDays + (delta * 10)
		if newVal >= 0 {
			s.recentDays = newVal
			changed = true
			s.needsRestart = true
		}

	case SettingNotesOutputSplit:
		newVal := s.notesOutputSplit + (delta * 5)
		if newVal >= 10 && newVal <= 90 {
			s.notesOutputSplit = newVal
			changed = true
		}

	case SettingStatsRefresh:
		newVal := s.statsRefreshSecs + delta
		if newVal >= 2 && newVal <= 300 {
			s.statsRefreshSecs = newVal
			changed = true
		}

	case SettingStatsFormat:
		newVal := s.statsFormat + delta
		if newVal >= 0 && newVal < len(statsFormatNames) {
			s.statsFormat = newVal
			changed = true
		}
	}

	return changed
}

// toggleValue toggles a checkbox value
func (s *SettingsPanel) toggleValue() bool {
	setting := SettingType(s.cursor)

	// Registry-driven bool settings toggle generically.
	if def := defFor(setting); def != nil {
		if def.kind == inputBool {
			s.newBool[setting] = !s.newBool[setting]
			return true
		}
		return false
	}

	switch setting {
	case SettingDangerousMode:
		s.dangerousMode = !s.dangerousMode
		return true

	case SettingGeminiYoloMode:
		s.geminiYoloMode = !s.geminiYoloMode
		return true

	case SettingCodexYoloMode:
		s.codexYoloMode = !s.codexYoloMode
		return true

	case SettingHermesYoloMode:
		s.hermesYoloMode = !s.hermesYoloMode
		return true

	case SettingCheckForUpdates:
		s.checkForUpdates = !s.checkForUpdates
		return true

	case SettingAutoUpdate:
		s.autoUpdate = !s.autoUpdate
		return true

	case SettingRemoveOrphans:
		s.removeOrphans = !s.removeOrphans
		return true

	case SettingGlobalSearchEnabled:
		s.globalSearchEnabled = !s.globalSearchEnabled
		s.needsRestart = true
		return true

	case SettingShowOutput:
		s.showOutput = !s.showOutput
		return true

	case SettingShowAnalytics:
		s.showAnalytics = !s.showAnalytics
		return true

	case SettingShowNotes:
		s.showNotes = !s.showNotes
		return true

	case SettingSyncTitle:
		s.syncTitle = !s.syncTitle
		return true

	case SettingMaintenanceEnabled:
		s.maintenanceEnabled = !s.maintenanceEnabled
		return true

	case SettingStatsEnabled:
		s.statsEnabled = !s.statsEnabled
		return true

	case SettingStatsShowCPU:
		s.statsShowCPU = !s.statsShowCPU
		return true

	case SettingStatsShowRAM:
		s.statsShowRAM = !s.statsShowRAM
		return true

	case SettingStatsShowDisk:
		s.statsShowDisk = !s.statsShowDisk
		return true

	case SettingStatsShowNetwork:
		s.statsShowNetwork = !s.statsShowNetwork
		return true

	case SettingStatsShowGPU:
		s.statsShowGPU = !s.statsShowGPU
		return true

	case SettingStatsShowLoad:
		s.statsShowLoad = !s.statsShowLoad
		return true

	case SettingShowSessionTimestamps:
		s.showSessionTimestamps = !s.showSessionTimestamps
		return true
	}

	return false
}

// isTextSetting returns true if current setting uses text input. Covers the
// legacy Claude config dir plus every registry literal (string/path/command).
func (s *SettingsPanel) isTextSetting() bool {
	setting := SettingType(s.cursor)
	if setting == SettingClaudeConfigDir {
		return true
	}
	if def := defFor(setting); def != nil {
		return def.kind == inputLiteral
	}
	return false
}

// startTextEdit begins text editing for current setting
func (s *SettingsPanel) startTextEdit() {
	setting := SettingType(s.cursor)
	if setting == SettingClaudeConfigDir {
		s.textBuffer = s.claudeConfigDir
		s.editingText = true
		return
	}
	if def := defFor(setting); def != nil && def.kind == inputLiteral {
		s.textBuffer = s.newText[setting]
		s.editingText = true
	}
}

// handleTextEdit processes keys during text editing
func (s *SettingsPanel) handleTextEdit(msg tea.KeyMsg) (*SettingsPanel, tea.Cmd, bool) {
	key := msg.String()

	switch key {
	case "enter":
		// Save the text
		setting := SettingType(s.cursor)
		if setting == SettingClaudeConfigDir {
			s.claudeConfigDir = s.textBuffer
		} else if def := defFor(setting); def != nil && def.kind == inputLiteral {
			s.newText[setting] = s.textBuffer
		}
		s.editingText = false
		return s, nil, true

	case "esc":
		// Cancel editing
		s.editingText = false
		return s, nil, false

	case "backspace":
		if len(s.textBuffer) > 0 {
			s.textBuffer = s.textBuffer[:len(s.textBuffer)-1]
		}

	default:
		// Add character
		if len(key) == 1 {
			s.textBuffer += key
		}
	}

	return s, nil, false
}

// View renders the settings panel as a two-pane (sections + detail) layout.
func (s *SettingsPanel) View() string {
	if !s.visible {
		return ""
	}

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)
	warningStyle := lipgloss.NewStyle().Foreground(ColorYellow)

	// Dialog dimensions. Wider than the old single column to fit two panes;
	// shrinks (and falls back to a single pane) on narrow terminals.
	dialogWidth := 78
	if s.width > 0 && dialogWidth > s.width-4 {
		dialogWidth = s.width - 4
	}
	if dialogWidth < 44 {
		dialogWidth = 44
	}
	innerWidth := dialogWidth - 4 // dialog has Padding(1,2): 2 cells each side
	if innerWidth < 1 {
		innerWidth = 1
	}

	const leftPaneWidth = 22
	const sep = " │ "
	sepCells := cellWidth(sep)
	// Need room for both panes plus a usable (>=24 cell) detail column.
	twoPane := innerWidth >= leftPaneWidth+sepCells+24

	leftRenderWidth := leftPaneWidth
	rightWidth := innerWidth
	if twoPane {
		rightWidth = innerWidth - leftPaneWidth - sepCells
	} else {
		leftRenderWidth = innerWidth
	}

	// Build each pane as a slice of lines so we can window vertically and, in
	// two-pane mode, join them row by row at a fixed left-column width.
	leftLines, leftCursorRow := s.renderSectionList(leftRenderWidth)
	rightLines, rightCursorRow := s.renderDetailPane(rightWidth)

	var paneRows []string
	var focusRow int
	switch {
	case twoPane:
		n := len(leftLines)
		if len(rightLines) > n {
			n = len(rightLines)
		}
		for i := 0; i < n; i++ {
			var l, r string
			if i < len(leftLines) {
				l = leftLines[i]
			}
			if i < len(rightLines) {
				r = rightLines[i]
			}
			paneRows = append(paneRows, fitCellWidth(l, leftPaneWidth)+dimStyle.Render(sep)+r)
		}
		if s.focusRight {
			focusRow = rightCursorRow
		} else {
			focusRow = leftCursorRow
		}
	case s.focusRight:
		paneRows = rightLines
		focusRow = rightCursorRow
	default:
		paneRows = leftLines
		focusRow = leftCursorRow
	}

	// Header (title + optional restart notice), bounded to the content width.
	titleLine := titleStyle.Render("Settings")
	if s.needsRestart {
		titleLine += warningStyle.Render("  (restart required)")
	}
	header := cellTruncate(titleLine, innerWidth, "") + "\n" + strings.Repeat("-", innerWidth)

	helpBar := cellTruncate(
		dimStyle.Render("Tab/←→ Pane · ↑↓ Move · Space Toggle · Enter Edit · Esc Close"),
		innerWidth, "",
	)

	// Vertical windowing of the pane block when it overflows the viewport.
	const dialogChrome = 4                                                    // border (2) + vertical padding (2)
	const reserved = 2 /*header*/ + 1 /*blank*/ + 1 /*blank before help*/ + 1 /*help*/
	availHeight := s.height - dialogChrome - reserved
	if availHeight < 6 {
		availHeight = 6
	}
	paneBlock := s.windowRows(paneRows, focusRow, availHeight, dimStyle)

	content := header + "\n\n" + paneBlock + "\n\n" + helpBar

	dialogStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorCyan).
		Background(ColorBg).
		Padding(1, 2).
		Width(dialogWidth)

	dialog := dialogStyle.Render(content)

	return lipgloss.Place(s.width, s.height, lipgloss.Center, lipgloss.Center, dialog)
}

// windowRows returns rows joined with newlines, scrolled so focusRow stays
// visible when the slice is taller than avail. Scroll indicators replace the
// edge rows. Mutates s.scrollOffset (the persisted window position), matching
// the pre-redesign scroll behavior.
func (s *SettingsPanel) windowRows(rows []string, focusRow, avail int, dimStyle lipgloss.Style) string {
	total := len(rows)
	if total <= avail {
		s.scrollOffset = 0
		return strings.Join(rows, "\n")
	}
	if focusRow-1 < s.scrollOffset {
		s.scrollOffset = focusRow - 1
	}
	if focusRow+1 >= s.scrollOffset+avail {
		s.scrollOffset = focusRow - avail + 2
	}
	if s.scrollOffset < 0 {
		s.scrollOffset = 0
	}
	if maxOff := total - avail; s.scrollOffset > maxOff {
		s.scrollOffset = maxOff
	}
	start := s.scrollOffset
	end := s.scrollOffset + avail
	if end > total {
		end = total
	}
	showUp := start > 0
	showDown := end < total
	if showUp {
		start++
	}
	if showDown {
		end--
	}
	var b strings.Builder
	if showUp {
		b.WriteString(dimStyle.Render("  ▲ more above") + "\n")
	}
	b.WriteString(strings.Join(rows[start:end], "\n"))
	if showDown {
		b.WriteString("\n" + dimStyle.Render("  ▼ more below"))
	}
	return b.String()
}

// renderSectionList builds the left (master) pane: a header plus one row per
// section. Returns the lines and the row index of the selected section.
func (s *SettingsPanel) renderSectionList(width int) ([]string, int) {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	selStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)
	activeStyle := lipgloss.NewStyle().Background(ColorSurface).Foreground(ColorCyan).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

	lines := []string{headerStyle.Render("SECTIONS")}
	cursorRow := 1
	for i, sec := range settingsSections {
		switch {
		case i == s.sectionCursor && !s.focusRight:
			lines = append(lines, activeStyle.Render(fitCellWidth("▸ "+sec.title, width)))
		case i == s.sectionCursor:
			lines = append(lines, selStyle.Render("▸ "+sec.title))
		default:
			lines = append(lines, dimStyle.Render("  "+sec.title))
		}
		if i == s.sectionCursor {
			cursorRow = 1 + i
		}
	}
	return lines, cursorRow
}

// renderDetailPane builds the right (detail) pane for the selected section.
// Returns the lines and the row index of the active setting.
func (s *SettingsPanel) renderDetailPane(width int) ([]string, int) {
	sectionStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)

	sec := s.currentSection()
	ruleLen := width
	if ruleLen > 28 {
		ruleLen = 28
	}
	lines := []string{
		sectionStyle.Render(strings.ToUpper(sec.title)),
		dimStyle.Render(strings.Repeat("─", ruleLen)),
	}
	cursorRow := 0

	if sec.info {
		hotkeys := resolveHotkeys(session.GetHotkeyOverrides())
		mcpKey := actionHotkey(hotkeys, hotkeyMCPManager)
		keyLine := "MCP Manager hotkey is unbound."
		if mcpKey != "" {
			keyLine = "MCP Manager hotkey: " + mcpKey
		}
		info := []string{
			"Edit your config file to add",
			"MCP servers and custom tools:",
			userConfigPathForDisplay(),
			"",
			keyLine,
			"",
			"Attach MCPs on any Claude,",
			"Gemini, or Cursor session.",
		}
		for _, l := range info {
			lines = append(lines, dimStyle.Render(cellTruncate(l, width, "")))
		}
		return lines, cursorRow
	}

	for _, st := range sec.settings {
		if int(st) == s.cursor {
			cursorRow = len(lines)
		}
		focused := s.focusRight && int(st) == s.cursor
		lines = append(lines, s.renderSettingRow(st, focused, width)...)
	}
	return lines, cursorRow
}

// renderSettingRow renders a single detail-pane setting (one main line plus an
// optional dim hint line), truncated to width and highlighted when focused.
func (s *SettingsPanel) renderSettingRow(setting SettingType, focused bool, width int) []string {
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)
	highlightStyle := lipgloss.NewStyle().Background(ColorSurface)

	var main, hint string
	if def := defFor(setting); def != nil {
		main, hint = s.renderRegistryRow(def, focused)
	} else {
		switch setting {
		case SettingTheme:
			main = "Theme: " + s.renderRadioGroup(themeNames, s.selectedTheme, focused)
		case SettingShowOutput:
			main = s.renderCheckbox("Show Output", s.showOutput) + " - Terminal output"
		case SettingShowAnalytics:
			main = s.renderCheckbox("Show Analytics", s.showAnalytics) + " - Analytics panel"
		case SettingShowNotes:
			main = s.renderCheckbox("Show Notes", s.showNotes) + " - Session notes"
		case SettingNotesOutputSplit:
			main = s.renderNumber("Notes/Output split:", s.notesOutputSplit, "%")
		case SettingDefaultTool:
			main = "Default tool: " + s.renderRadioGroup(s.toolNames, s.selectedTool, focused)
			hint = "Pre-selected for new sessions"
		case SettingDangerousMode:
			main = s.renderCheckbox("Dangerous mode", s.dangerousMode) + " - Skip permission prompts"
		case SettingClaudeConfigDir:
			line := "Config dir"
			if s.claudeConfigIsScope && s.profile != "" {
				line += " (" + s.profile + ")"
			}
			line += ": "
			if s.editingText && s.cursor == int(SettingClaudeConfigDir) {
				line += "[" + s.textBuffer + "|]"
			} else if s.claudeConfigDir == "" {
				line += "~/.claude (default)"
			} else {
				line += s.claudeConfigDir
			}
			main = line
		case SettingGeminiYoloMode:
			main = s.renderCheckbox("YOLO mode", s.geminiYoloMode) + " - Auto-approve all actions"
		case SettingCodexYoloMode:
			main = s.renderCheckbox("YOLO mode", s.codexYoloMode) + " - Bypass approvals and sandbox"
		case SettingHermesYoloMode:
			main = s.renderCheckbox("YOLO mode", s.hermesYoloMode) + " - Auto-approve all tool calls"
		case SettingCheckForUpdates:
			main = s.renderCheckbox("Check for updates on startup", s.checkForUpdates)
		case SettingAutoUpdate:
			main = s.renderCheckbox("Auto-install updates", s.autoUpdate)
		case SettingLogMaxSize:
			main = s.renderNumber("Max file size:", s.logMaxSizeMB, "MB")
		case SettingLogMaxLines:
			main = s.renderNumber("Lines to keep:", s.logMaxLines, "")
		case SettingGlobalSearchEnabled:
			main = s.renderCheckbox("Enabled", s.globalSearchEnabled)
		case SettingSearchTier:
			main = "Search tier: " + s.renderRadioGroup(tierNames, s.searchTier, focused)
		case SettingRecentDays:
			main = s.renderNumber("Recent days:", s.recentDays, "(0 = all)")
		case SettingRemoveOrphans:
			main = s.renderCheckbox("Remove orphan logs", s.removeOrphans)
		case SettingMaintenanceEnabled:
			main = s.renderCheckbox("Auto-maintenance", s.maintenanceEnabled)
			hint = "Prune logs, clean backups, archive"
		case SettingStatsEnabled:
			main = s.renderCheckbox("Enabled", s.statsEnabled) + " - Status bar stats"
		case SettingStatsRefresh:
			main = s.renderNumber("Refresh interval:", s.statsRefreshSecs, "sec")
		case SettingStatsFormat:
			main = "Format: " + s.renderRadioGroup(statsFormatNames, s.statsFormat, focused)
		case SettingStatsShowCPU:
			main = s.renderCheckbox("Show CPU", s.statsShowCPU)
		case SettingStatsShowRAM:
			main = s.renderCheckbox("Show RAM", s.statsShowRAM)
		case SettingStatsShowDisk:
			main = s.renderCheckbox("Show Disk", s.statsShowDisk)
		case SettingStatsShowNetwork:
			main = s.renderCheckbox("Show Network", s.statsShowNetwork)
		case SettingStatsShowGPU:
			main = s.renderCheckbox("Show GPU", s.statsShowGPU)
		case SettingStatsShowLoad:
			main = s.renderCheckbox("Show Load", s.statsShowLoad)
		case SettingSyncTitle:
			main = s.renderCheckbox("Sync session title", s.syncTitle)
			hint = "Let the agent rename the session"
		case SettingShowSessionTimestamps:
			main = s.renderCheckbox("Show session timestamps", s.showSessionTimestamps)
			hint = "Last activity per row"
		}
	}

	if focused {
		main = highlightStyle.Render(main)
	}
	out := []string{cellTruncate("  "+labelStyle.Render(main), width, "")}
	if hint != "" {
		out = append(out, cellTruncate("    "+dimStyle.Render(hint), width, ""))
	}
	return out
}

// renderRegistryRow builds the main line and hint for a registry-driven
// setting (settings_defs.go) purely from its kind, reusing the same checkbox /
// radio / number widgets as the legacy settings. Literals render their typed
// value (or a dim placeholder), with an inline edit caret while typing.
func (s *SettingsPanel) renderRegistryRow(def *settingDef, focused bool) (string, string) {
	switch def.kind {
	case inputBool:
		return s.renderCheckbox(def.label, s.newBool[def.key]), def.desc
	case inputEnum:
		return def.label + ": " + s.renderRadioGroup(def.enumNames, s.newEnum[def.key], focused), def.desc
	case inputInt:
		return s.renderNumber(def.label+":", s.newInt[def.key], def.suffix), def.desc
	case inputLiteral:
		line := def.label + ": "
		switch {
		case s.editingText && s.cursor == int(def.key):
			line += "[" + s.textBuffer + "|]"
		case s.newText[def.key] != "":
			line += s.newText[def.key]
		default:
			line += def.placeholder
		}
		return line, def.desc
	}
	return def.label, def.desc
}

// renderCheckbox renders a checkbox with label
func (s *SettingsPanel) renderCheckbox(label string, checked bool) string {
	box := "[ ]"
	if checked {
		box = "[x]"
	}
	return box + " " + label
}

// renderRadioGroup renders a group of radio options
func (s *SettingsPanel) renderRadioGroup(options []string, selected int, focused bool) string {
	var parts []string
	for i, opt := range options {
		if i == selected {
			style := lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
			parts = append(parts, style.Render(">"+opt))
		} else {
			style := lipgloss.NewStyle().Foreground(ColorTextDim)
			parts = append(parts, style.Render(" "+opt))
		}
	}
	return strings.Join(parts, "  ")
}

// renderNumber renders a number input with label and suffix
func (s *SettingsPanel) renderNumber(label string, value int, suffix string) string {
	numStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	valueStr := strconv.Itoa(value)
	result := label + " [" + numStyle.Render(valueStr) + "]"
	if suffix != "" {
		result += " " + suffix
	}
	return result
}
