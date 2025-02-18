package cli

import (
	"io"
	"net"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/errors"
)

var _cmdAttach = func() *cobra.Command {
	const filter = filterAll
	var names, ids, tags []string
	cmd := &cobra.Command{
		Use:               "attach [name|tag|id]",
		Short:             "attach to process stdin/stdout",
		Aliases:           []string{"a"},
		GroupID:           "management",
		ValidArgsFunction: completeArgGenericSelector(filter),
		RunE: func(_ *cobra.Command, args []string) error {
			filterFunc := core.FilterFunc(
				core.WithAllIfNoFilters,
				core.WithGeneric(args...),
				core.WithIDs(ids...),
				core.WithNames(names...),
				core.WithTags(tags...),
			)
			procs := listProcs(dbb).
				Filter(func(ps core.ProcStat) bool { return filterFunc(ps.Proc) }).
				Slice()
			if len(procs) != 1 {
				return errors.Newf("expected 1 process, found %d", len(procs))
			}
			proc := procs[0]

			conn, errIn := net.Dial("unix", filepath.Join(core.DirHome, proc.ID.String()+".sock"))
			if errIn != nil {
				return errors.Wrap(errIn, "connect to proc socket")
			}

			go func() {
				if _, errOut := io.Copy(os.Stdout, conn); errOut != nil {
					log.Error().Err(errOut).Msg("copy stdout")
				}
			}()
			_, errIn = io.Copy(conn, os.Stdin)
			return errors.Wrap(errIn, "copy stdin")
		},
	}
	addFlagGenerics(cmd, filter, &names, &tags, &ids)
	return cmd
}()
