package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"nortezh-cli/internal/auth"
	"nortezh-cli/internal/config"
)

// openBrowser is the function used to launch the browser during login.
// Tests override this variable to intercept the login URL.
var openBrowser auth.OpenBrowserFunc = defaultOpenBrowser

func defaultOpenBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}

func newLoginCmd(g *Globals) *cobra.Command {
	var (
		serviceAccount string
		key            string
		keyFile        string
	)
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with the Nortezh platform",
		RunE: func(cmd *cobra.Command, args []string) error {
			if serviceAccount != "" || key != "" || keyFile != "" {
				return runServiceAccountLogin(cmd, serviceAccount, key, keyFile)
			}
			return runBrowserLogin(cmd, g)
		},
	}
	cmd.Flags().StringVar(&serviceAccount, "service-account", "", "service account email")
	cmd.Flags().StringVar(&key, "key", "", "service account key (inline)")
	cmd.Flags().StringVar(&keyFile, "key-file", "", "path to file containing the service account key (- for stdin)")
	return cmd
}

func runBrowserLogin(cmd *cobra.Command, g *Globals) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	server := config.ResolveServer(g.Server, cfg)
	fmt.Fprintf(cmd.OutOrStdout(), "Opening browser at %s...\n", server)
	creds, err := auth.Login(cmd.Context(), server, openBrowser)
	if err != nil {
		return err
	}
	if err := auth.Save(creds); err != nil {
		return err
	}
	fmt.Fprintln(cmd.OutOrStdout(), "Logged in.")
	return nil
}

func runServiceAccountLogin(cmd *cobra.Command, serviceAccount, key, keyFile string) error {
	if serviceAccount == "" {
		return fmt.Errorf("--service-account is required for service account login")
	}
	if key == "" && keyFile == "" {
		return fmt.Errorf("--key or --key-file is required for service account login")
	}
	if key != "" && keyFile != "" {
		return fmt.Errorf("--key and --key-file are mutually exclusive")
	}

	actualKey := key
	if keyFile != "" {
		k, err := readKeyFile(keyFile, cmd.InOrStdin())
		if err != nil {
			return err
		}
		actualKey = k
	}

	if err := auth.Save(&auth.ServiceAccountCreds{
		Email: serviceAccount,
		Key:   actualKey,
	}); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Logged in as service account %s.\n", serviceAccount)
	return nil
}

// readKeyFile reads the key from path. If path is "-", it reads from stdin.
func readKeyFile(path string, stdin io.Reader) (string, error) {
	var r io.Reader
	if path == "-" {
		r = stdin
	} else {
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		r = f
	}
	var sb strings.Builder
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		sb.WriteString(sc.Text())
	}
	if err := sc.Err(); err != nil {
		return "", err
	}
	return strings.TrimSpace(sb.String()), nil
}

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := auth.Wipe(); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Logged out.")
			return nil
		},
	}
}

func newWhoamiCmd(g *Globals) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Print the currently authenticated user",
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := buildClient(g)
			if err != nil {
				return err
			}
			var out struct {
				Email string `json:"email"`
				ID    string `json:"id"`
			}
			if err := c.Invoke(cmd.Context(), "auth.me", nil, &out); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), out.Email)
			return nil
		},
	}
}
