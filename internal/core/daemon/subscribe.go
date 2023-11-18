package daemon

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/daemon/eventbus"
)

func (s *Server) Subscribe(ctx context.Context, id core.PMID) (<-chan core.Proc, error) {
	// can't get incoming query in interceptor, so logging here also
	log.Info().Stringer("pmid", id).Msg("Subscribe method called")

	updCh := s.ebus.Subscribe(
		"sub"+time.Now().String(),
		eventbus.KindProcStarted,
		eventbus.KindProcStopped,
	)

	procs := s.db.GetProcs(core.WithIDs(id))

	ch := make(chan core.Proc)
	go func() {
		defer close(ch)

		for {
			select {
			case <-ctx.Done():
				return
			case <-updCh:
				proc, ok := procs[id]
				if !ok {
					log.Error().Stringer("pmid", id).Msg("failed to find proc")
					return
				}

				select {
				case <-ctx.Done():
					return
				case ch <- proc:
				}
			}
		}
	}()
	return ch, nil
}
