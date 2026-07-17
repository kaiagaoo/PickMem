package cli

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"

	"github.com/kaiagaoo/PickMem/internal/userconf"
	"github.com/kaiagaoo/PickMem/internal/web"
	"github.com/spf13/cobra"
)

func newWebCmd() *cobra.Command {
	var (
		port   int
		host   string
		noOpen bool
	)
	cmd := &cobra.Command{
		Use:   "web",
		Short: "Launch the PickMem web app (local server)",
		Long: `Serve the PickMem management UI on localhost and open it in your browser.

The web app is a full curation surface: pick what the model sees, browse /
add / edit / delete notes, review the inbox, and save/switch lenses. It reads
and writes the same pickmem/active.json + lenses.json as the TUI picker and
the MCP server, so switching surfaces never changes what the model sees.

The server binds to 127.0.0.1 only and makes no network calls — your vault
stays on disk. It reloads the vault from disk on every request, so edits you
make in Obsidian or the CLI show up on the next click.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			// Record the launched vault as current + recent so the web
			// switcher lists it and future commands default to it.
			_ = userconf.SetCurrent(s.Root)
			srv := web.NewServer(s)

			addr := fmt.Sprintf("%s:%d", host, port)
			ln, err := net.Listen("tcp", addr)
			if err != nil {
				return fmt.Errorf("listen on %s: %w (try a different --port)", addr, err)
			}
			url := fmt.Sprintf("http://%s", ln.Addr().String())

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "PickMem web UI: %s\n", url)
			fmt.Fprintf(out, "Vault:          %s\n", s.Root)
			fmt.Fprintln(out, "Press Ctrl-C to stop.")

			if !noOpen {
				// Open after a short delay so the listener is definitely
				// serving by the time the browser requests the page.
				go func() {
					time.Sleep(300 * time.Millisecond)
					if err := openBrowser(url); err != nil {
						fmt.Fprintf(cmd.ErrOrStderr(), "could not open browser automatically: %v\n", err)
					}
				}()
			}

			httpSrv := &http.Server{Handler: srv.Handler()}
			if err := httpSrv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
				return err
			}
			return nil
		},
	}
	cmd.Flags().IntVar(&port, "port", 4577, "port to listen on")
	cmd.Flags().StringVar(&host, "host", "127.0.0.1", "host/interface to bind (localhost by default)")
	cmd.Flags().BoolVar(&noOpen, "no-open", false, "don't open a browser automatically")
	return cmd
}

// openBrowser opens url in the platform's default browser. Best-effort:
// callers treat a failure as non-fatal (the URL is already printed).
func openBrowser(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	default: // linux, bsd, …
		return exec.Command("xdg-open", url).Start()
	}
}
