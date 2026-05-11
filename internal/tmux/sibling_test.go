package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveParentPID_MatchesOSGetppid asserts the cross-platform
// parent-PID resolver returns the same value as os.Getppid() for the
// current process. Regression guard for #927: killStaleControlClients
// uses this to decide whether a control client is owned by a live
// sibling agent-deck instance, so a wrong PPID would either let
// dueling kills continue (false negative) or skip orphan reaps
// (false positive).
func TestResolveParentPID_MatchesOSGetppid(t *testing.T) {
	ppid, err := resolveParentPID(os.Getpid())
	require.NoError(t, err)
	assert.Equal(t, os.Getppid(), ppid)
}

// TestShouldSkipControlClientKill_NoHookKillsOrphan asserts the
// pre-#927 always-kill behaviour is preserved when no live-instance
// lookup is installed (e.g. tests, CLI tools that never wire one in).
func TestShouldSkipControlClientKill_NoHookKillsOrphan(t *testing.T) {
	resolve := func(int) (int, error) { return 1, nil }
	assert.False(t, shouldSkipControlClientKill(999, nil, resolve),
		"with no lookup installed, decision helper must say 'don't skip'")
}

// TestShouldSkipControlClientKill_EmptyLookupKillsOrphan asserts that
// an installed-but-empty lookup (e.g. no live siblings registered yet)
// still permits the orphan reap. Belt-and-braces in case the hook is
// wired before any instance has heartbeated.
func TestShouldSkipControlClientKill_EmptyLookupKillsOrphan(t *testing.T) {
	live := func() map[int]bool { return map[int]bool{} }
	resolve := func(int) (int, error) { return 1, nil }
	assert.False(t, shouldSkipControlClientKill(999, live, resolve),
		"empty live-PID set must not block the orphan reap")
}

// TestShouldSkipControlClientKill_PpidResolveFailureKillsOrphan asserts
// that when the parent PID cannot be determined (e.g. the client
// already exited and /proc/<pid>/status is gone), we fall back to the
// pre-#927 always-kill behaviour. Failing closed in this direction
// preserves the orphan-reap guarantee from #595/#737 in pathological
// cases where heartbeat lookup is moot anyway.
func TestShouldSkipControlClientKill_PpidResolveFailureKillsOrphan(t *testing.T) {
	live := func() map[int]bool { return map[int]bool{12345: true} }
	resolve := func(int) (int, error) { return 0, errors.New("not found") }
	assert.False(t, shouldSkipControlClientKill(999, live, resolve),
		"PPID-resolve failure must not block the orphan reap")
}

// TestShouldSkipControlClientKill_UnknownPpidKillsOrphan asserts the
// orphan path: a client whose PPID does not match any live agent-deck
// instance heartbeat is reaped (no live sibling owns it). This is the
// path #595/#737 originally added — cleaning up `tmux -C attach-session`
// children left over from crashed/SIGKILL'd previous runs.
func TestShouldSkipControlClientKill_UnknownPpidKillsOrphan(t *testing.T) {
	live := func() map[int]bool { return map[int]bool{12345: true} }
	resolve := func(int) (int, error) { return 99999, nil }
	assert.False(t, shouldSkipControlClientKill(777, live, resolve),
		"client whose PPID matches no live heartbeat must be reaped as orphan")
}

// TestShouldSkipControlClientKill_LiveSiblingIsPreserved is the core
// #927 regression guard: when the control client's PPID matches a
// live agent-deck instance heartbeat, the kill MUST be skipped. The
// duel between two TUIs against the same profile (#927) only ends
// when this decision flips from "always kill" to "skip if owned by a
// live sibling."
func TestShouldSkipControlClientKill_LiveSiblingIsPreserved(t *testing.T) {
	siblingPID := 12345
	live := func() map[int]bool { return map[int]bool{siblingPID: true} }
	resolve := func(int) (int, error) { return siblingPID, nil }
	assert.True(t, shouldSkipControlClientKill(777, live, resolve),
		"control client owned by live sibling TUI must be preserved (#927)")
}

// createSiblingTestSession spawns a fresh tmux session in the isolated
// TestMain socket and registers cleanup. Differs from createTestSession
// in that it gates on skipIfNoTmuxBinary only — the legacy
// skipIfNoTmuxServer gate (which requires a non-bootstrap session to
// already exist) would silent-skip these regression tests in CI where
// only the bootstrap session is present.
func createSiblingTestSession(t *testing.T, suffix string) string {
	t.Helper()
	skipIfNoTmuxBinary(t)
	name := SessionPrefix + "siblingtest-" + suffix
	require.NoError(t, exec.Command("tmux", "new-session", "-d", "-s", name).Run(),
		"failed to create test session %s", name)
	t.Cleanup(func() {
		_ = exec.Command("tmux", "kill-session", "-t", name).Run()
	})
	return name
}

// TestKillStaleControlClients_PreservesSiblingTUIControlClient is the
// end-to-end #927 regression guard. Without the fix, two concurrent
// agent-deck TUIs against the same profile SIGTERM each other's live
// `tmux -C attach-session` clients on every reconnect cycle, flipping
// every managed session into StatusError within ~20s.
//
// We stand in for the second TUI by installing a live-instance lookup
// that includes os.Getpid() — the PPID of the control client we
// register below. killStaleControlClients should now see the client
// as sibling-owned and skip it. The complementary case (orphan still
// reaped) stays covered by the pre-#927 TestKillStaleControlClients
// test, where no lookup is installed.
func TestKillStaleControlClients_PreservesSiblingTUIControlClient(t *testing.T) {
	name := createSiblingTestSession(t, "preserve")

	pipe, err := NewControlPipe(name, "")
	require.NoError(t, err)
	siblingPID := pipe.cmd.Process.Pid
	t.Cleanup(func() { pipe.Close() })

	// Wait for the client to register with tmux.
	require.Eventually(t, func() bool {
		out, _ := exec.Command("tmux", "list-clients", "-t", name,
			"-F", "#{client_control_mode} #{client_pid}").Output()
		return strings.Contains(string(out), fmt.Sprintf("1 %d", siblingPID))
	}, 3*time.Second, 100*time.Millisecond, "control client should register with tmux")

	// Pretend the test runner is a live sibling TUI. The control client
	// we just spawned has PPID = os.Getpid(); registering os.Getpid()
	// as live forces the killStale path to treat the client as
	// sibling-owned and skip the kill.
	SetLiveInstanceLookup(func() map[int]bool {
		return map[int]bool{os.Getpid(): true}
	})
	t.Cleanup(func() { SetLiveInstanceLookup(nil) })

	killStaleControlClients(name, "")

	// Give the would-be SIGTERM time to land (or, with the fix, not).
	// 300ms is well past softKillProcess's 500ms grace would have begun.
	time.Sleep(300 * time.Millisecond)

	out, _ := exec.Command("tmux", "list-clients", "-t", name,
		"-F", "#{client_control_mode} #{client_pid}").Output()
	assert.Contains(t, string(out), fmt.Sprintf("1 %d", siblingPID),
		"sibling-owned control client must NOT be killed (#927 regression)")
}

// TestKillStaleControlClients_StillReapsOrphanWithHookInstalled asserts
// that wiring up the live-instance lookup does NOT regress the orphan
// reap behaviour from #595/#737 — a control client whose PPID is NOT
// in the live set must still be SIGTERMed. Belt for the suspenders of
// the unit test on shouldSkipControlClientKill: this exercises the
// real /proc-based resolver against a real tmux server.
func TestKillStaleControlClients_StillReapsOrphanWithHookInstalled(t *testing.T) {
	name := createSiblingTestSession(t, "orphan")

	pipe, err := NewControlPipe(name, "")
	require.NoError(t, err)
	stalePID := pipe.cmd.Process.Pid
	t.Cleanup(func() { pipe.Close() })

	require.Eventually(t, func() bool {
		out, _ := exec.Command("tmux", "list-clients", "-t", name,
			"-F", "#{client_control_mode} #{client_pid}").Output()
		return strings.Contains(string(out), fmt.Sprintf("1 %d", stalePID))
	}, 3*time.Second, 100*time.Millisecond, "control client should register with tmux")

	// Hook returns a live set that does NOT include os.Getpid(), so
	// the control client's PPID (= os.Getpid()) is treated as an
	// orphan — the pre-#927 behaviour must hold.
	SetLiveInstanceLookup(func() map[int]bool {
		return map[int]bool{99999999: true}
	})
	t.Cleanup(func() { SetLiveInstanceLookup(nil) })

	killStaleControlClients(name, "")

	require.Eventually(t, func() bool {
		out, _ := exec.Command("tmux", "list-clients", "-t", name,
			"-F", "#{client_control_mode} #{client_pid}").Output()
		return !strings.Contains(string(out), fmt.Sprintf("1 %d", stalePID))
	}, 2*time.Second, 100*time.Millisecond,
		"orphaned control client (PPID not in live set) must still be reaped")
}
