package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/tonydisco/claude-usage/internal/auth"
)

func newLoginCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Paste your claude.ai session cookie; stored in the OS keychain",
		Long: `Prompts for the value of your claude.ai sessionKey cookie and stores it
securely in the OS keychain (macOS Keychain / Windows Credential Manager /
Secret Service on Linux).

How to grab the cookie:
  1. Open https://claude.ai in your browser and make sure you're signed in.
  2. DevTools -> Application -> Cookies -> https://claude.ai
  3. Copy the value of the cookie named "sessionKey".
  4. Paste it when prompted below.

Your cookie is never written to disk in plaintext.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprint(cmd.OutOrStderr(), "Paste sessionKey cookie value: ")
			cookie, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				return err
			}
			cookie = strings.TrimSpace(cookie)
			if cookie == "" {
				return fmt.Errorf("empty input; aborting")
			}
			if err := auth.SaveCookie(cookie); err != nil {
				return fmt.Errorf("save to keychain: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Saved. Try: claude-usage status")
			return nil
		},
	}
}
