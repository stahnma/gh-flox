package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) newClearCacheCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clearcache",
		Short: "Clear the cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			a.Cache.Flush()
			if err := a.Cache.SaveToFile(a.Config.CacheFile); err != nil {
				return fmt.Errorf("saving cache: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Cache cleared.")
			return nil
		},
	}
}
