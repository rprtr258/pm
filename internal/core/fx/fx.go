package fx

import (
	"context"

	"github.com/rprtr258/xerr"
)

type Lifecycle struct {
	Name       string
	Start      func(context.Context) error
	StartAsync func(context.Context)
	Close      func()
}

func Combine(name string, lcs ...Lifecycle) Lifecycle {
	return Lifecycle{
		Name: name,
		Start: func(ctx context.Context) error {
			for _, lc := range lcs {
				if err := lc.Run(ctx); err != nil {
					return err
				}
			}
			return nil
		},
		StartAsync: nil,
		Close:      nil,
	}
}

func (lc Lifecycle) close() {
	if lc.Close != nil {
		lc.Close()
	}
}

func (lc Lifecycle) Run(ctx context.Context) error {
	if lc.Start != nil {
		if err := lc.Start(ctx); err != nil {
			return xerr.NewWM(err, "start component", xerr.Fields{
				"component": lc.Name,
			})
		}
		lc.close()
	} else {
		go func() {
			lc.StartAsync(ctx)
			lc.close()
		}()
	}
	return nil
}
