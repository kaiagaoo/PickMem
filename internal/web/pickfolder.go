package web

import (
	"bytes"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
)

// handlePickFolder pops the operating system's native folder-chooser and
// returns the absolute path the user selected as {"path": "..."}.
//
// The browser sandbox can't hand a Go server a real filesystem path (a
// directory <input> only yields relative names), so the picker has to run
// server-side. That's fine here: pickmem web is a local process on the same
// machine as the user, exactly like the browser-opening in `pickmem web`.
//
// A user cancel is not an error — it returns {"path": ""} so the client just
// leaves the text field untouched. It runs outside withLock/withVault: the
// native dialog blocks until the user answers, and it never touches the
// store, so holding the process lock would needlessly stall other requests.
func (s *Server) handlePickFolder(w http.ResponseWriter, r *http.Request) {
	path, err := pickFolderNative()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"path": path})
}

// pickFolderNative shells out to the platform's native directory picker and
// returns the chosen absolute path, or "" if the user canceled. An error is
// reserved for "no picker available" or a real failure, so the client can
// fall back to manual path entry.
func pickFolderNative() (string, error) {
	switch runtime.GOOS {
	case "darwin":
		// choose folder yields an AppleScript alias; POSIX path of … turns it
		// into a plain absolute path. Cancel exits non-zero with -128.
		const script = `POSIX path of (choose folder with prompt "Choose a folder for PickMem")`
		out, stderr, err := run("osascript", "-e", script)
		if err != nil {
			if strings.Contains(stderr, "-128") {
				return "", nil // user canceled
			}
			return "", fmt.Errorf("folder picker failed: %s", firstLine(stderr, err))
		}
		return strings.TrimSpace(out), nil

	case "linux":
		if p, err := exec.LookPath("zenity"); err == nil {
			out, _, err := run(p, "--file-selection", "--directory",
				"--title=Choose a folder for PickMem")
			if err != nil {
				return "", nil // zenity exits 1 on cancel; treat any exit as cancel
			}
			return strings.TrimSpace(out), nil
		}
		if p, err := exec.LookPath("kdialog"); err == nil {
			out, _, err := run(p, "--getexistingdirectory", ".")
			if err != nil {
				return "", nil // cancel
			}
			return strings.TrimSpace(out), nil
		}
		return "", fmt.Errorf("no folder picker found (install zenity or kdialog) — type the path instead")

	case "windows":
		const ps = `Add-Type -AssemblyName System.Windows.Forms;` +
			`$d = New-Object System.Windows.Forms.FolderBrowserDialog;` +
			`if ($d.ShowDialog() -eq [System.Windows.Forms.DialogResult]::OK) { [Console]::Out.Write($d.SelectedPath) }`
		out, stderr, err := run("powershell", "-NoProfile", "-STA", "-Command", ps)
		if err != nil {
			return "", fmt.Errorf("folder picker failed: %s", firstLine(stderr, err))
		}
		return strings.TrimSpace(out), nil // empty output == canceled

	default:
		return "", fmt.Errorf("native folder picker isn't supported on %s — type the path instead", runtime.GOOS)
	}
}

// run executes cmd and returns its stdout, stderr, and error.
func run(name string, args ...string) (stdout, stderr string, err error) {
	cmd := exec.Command(name, args...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.String(), errBuf.String(), err
}

// firstLine picks the most useful one-line error message: the first line of
// stderr if present, otherwise the process error.
func firstLine(stderr string, err error) string {
	if s := strings.TrimSpace(stderr); s != "" {
		if i := strings.IndexByte(s, '\n'); i >= 0 {
			return s[:i]
		}
		return s
	}
	return err.Error()
}
