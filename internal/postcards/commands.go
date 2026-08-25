package postcards

import (
	"fmt"

	"github.com/blackfyre/wga/internal/config"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/spf13/cobra"
)

// RegisterCommands adds postcard delivery inspection and resolution commands.
func RegisterCommands(app *pocketbase.PocketBase, keyringProvider func() (config.PostcardTokenKeyring, error)) {
	app.RootCmd.AddCommand(newPostcardsCommand(app, keyringProvider))
}

func newPostcardsCommand(app core.App, keyringProvider func() (config.PostcardTokenKeyring, error)) *cobra.Command {
	var unresolved bool
	inspect := &cobra.Command{
		Use:   "inspect [attempt-id]",
		Short: "Inspect postcard delivery attempts",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := ""
			if len(args) == 1 {
				filter = `id = {:id}`
			}
			if unresolved {
				if filter != "" {
					filter += " && "
				}
				filter += `status = 'dead_lettered' && resolved_at = ''`
			}
			records, err := app.FindRecordsByFilter(collectionDeliveryAttempts, filter, "-updated", 0, 0, map[string]any{"id": firstArg(args)})
			if err != nil {
				return err
			}
			for _, record := range records {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s message_id=%s status=%s attempts=%d outcome=%s\n", record.Id, record.GetString("message_id"), record.GetString("status"), record.GetInt("attempt_count"), record.GetString("last_error_class")); err != nil {
					return err
				}
			}
			return nil
		},
	}
	inspect.Flags().BoolVar(&unresolved, "unresolved", false, "show unresolved dead-lettered attempts")

	var resolutionCode string
	var resolutionSummary string
	resolve := &cobra.Command{
		Use:   "resolve <attempt-id>",
		Short: "Resolve a dead-lettered postcard delivery attempt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return ResolveAttempt(app, args[0], resolutionCode, resolutionSummary)
		},
	}
	resolve.Flags().StringVar(&resolutionCode, "code", "", "resolved_manually, closed_without_replay, or ignored_duplicate")
	resolve.Flags().StringVar(&resolutionSummary, "summary", "", "operator resolution summary")

	var confirmSafe bool
	replay := &cobra.Command{
		Use:   "replay <attempt-id>",
		Short: "Create a linked retry after external reconciliation",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !confirmSafe {
				return fmt.Errorf("--confirm-safe is required before replaying a delivery")
			}
			attempt, err := ReplayAttempt(app, args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), attempt.Id)
			return err
		},
	}
	replay.Flags().BoolVar(&confirmSafe, "confirm-safe", false, "confirm that the prior delivery was reconciled")

	var sourceKeyID string
	var rewrapLimit int
	rewrap := &cobra.Command{
		Use:   "rewrap-token-key --from <old-id> [--limit N]",
		Short: "Rewrap live postcard recipient tokens to the active key",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if keyringProvider == nil {
				return errInvalidRecipientTokenKeyring
			}
			keyring, err := keyringProvider()
			if err != nil {
				return err
			}
			count, err := RewrapTokenKey(app, keyring, sourceKeyID, rewrapLimit)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "rewrapped=%d\n", count)
			return err
		},
	}
	rewrap.Flags().StringVar(&sourceKeyID, "from", "", "source key identifier")
	rewrap.Flags().IntVar(&rewrapLimit, "limit", defaultTokenRewrapLimit, "maximum deliveries to rewrap")

	group := &cobra.Command{Use: "postcards", Short: "Operate postcard delivery attempts"}
	group.AddCommand(inspect, resolve, replay, rewrap)
	return group
}

// firstArg returns the first argument or an empty string when none was supplied.
func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
