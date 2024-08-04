package cli

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/infra/config"
	"github.com/rprtr258/pm/internal/infra/db"
)

var dbb, cfg = func() (db.Handle, core.Config) {
	db, config, errNewApp := config.New()
	if errNewApp != nil {
		log.Fatal().Err(errNewApp).Msg("new app")
	}
	return db, config
}()

func printProcs(procs ...core.ProcStat) {
	for _, proc := range procs {
		fmt.Println(proc.Name)
	}
}

func addFlagConfig(cmd *cobra.Command, config *string) {
	cmd.Flags().StringVarP(config, "config", "f", "", "config file to use")
}

func addFlagStrings(
	cmd *cobra.Command,
	dest *[]string,
	long string,
	description string,
	completeFunc func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective),
) {
	cmd.Flags().StringSliceVar(dest, long, nil, description)
	registerFlagCompletionFunc(cmd, long, completeFunc)
}

func addFlagNames(cmd *cobra.Command, names *[]string) {
	addFlagStrings(cmd, names, "name", "name(s) of process(es)", completeFlagName)
}

func addFlagTags(cmd *cobra.Command, tags *[]string) {
	addFlagStrings(cmd, tags, "tag", "tag(s) of process(es)", completeFlagTag)
}

func addFlagIDs(cmd *cobra.Command, ids *[]string) {
	addFlagStrings(cmd, ids, "id", "id(s) of process(es) to list", completeFlagIDs)
}

func addFlagInteractive(cmd *cobra.Command, dest *bool) {
	cmd.Flags().BoolVarP(dest, "interactive", "i", false, "prompt before taking action")
}

func confirmProc(ps core.ProcStat, action string) bool {
	var result bool
	if err := huh.NewConfirm().
		Title(fmt.Sprintf(
			"Do you really want to %s process %q id=%s ? ",
			action, ps.Name, ps.ID.String(),
		)).
		Inline(true).
		Value(&result).
		WithTheme(theme()).
		Run(); err != nil {
		log.Fatal().Msg(err.Error())
	}
	return result
}
