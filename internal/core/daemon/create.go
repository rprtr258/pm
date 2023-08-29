package daemon

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"

	pb "github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/core/namegen"
	"github.com/rprtr258/pm/internal/infra/db"
)

type CreateQuery struct {
	Command    string
	Args       []string
	Name       fun.Option[string]
	Cwd        string
	Tags       []string
	Env        map[string]string
	Watch      fun.Option[string]
	StdoutFile fun.Option[string]
	StderrFile fun.Option[string]
}

func (srv *daemonServer) create(query CreateQuery) (core.ProcID, error) {
	// try to find by name and update
	if name, ok := query.Name.Unpack(); ok {
		procs := srv.db.GetProcs(core.WithAllIfNoFilters)

		if procID, ok := fun.FindKeyBy(
			procs,
			func(_ core.ProcID, procData core.Proc) bool {
				return procData.Name == name
			},
		); ok { // TODO: early exit from outer if block
			procData := core.Proc{
				ID:         procID,
				Status:     core.NewStatusCreated(),
				Name:       name,
				Cwd:        query.Cwd,
				Tags:       fun.Uniq(append(query.Tags, "all")),
				Command:    query.Command,
				Args:       query.Args,
				Watch:      query.Watch,
				Env:        query.Env,
				StdoutFile: query.StdoutFile.OrDefault(filepath.Join(srv.logsDir, fmt.Sprintf("%d.stdout", procID))),
				StderrFile: query.StderrFile.OrDefault(filepath.Join(srv.logsDir, fmt.Sprintf("%d.stderr", procID))),
			}

			proc := procs[procID]
			if proc.Status.Status != core.StatusRunning ||
				proc.Cwd == procData.Cwd &&
					len(proc.Tags) == len(procData.Tags) && // TODO: compare lists, not lengths
					proc.Command == procData.Command &&
					len(proc.Args) == len(procData.Args) && // TODO: compare lists, not lengths
					proc.Watch == procData.Watch {
				// not updated, do nothing
				return procID, nil
			}

			if errUpdate := srv.db.UpdateProc(procData); errUpdate != nil {
				return 0, xerr.NewWM(errUpdate, "update proc", xerr.Fields{
					// "procData": procFields(procData),
				})
			}

			return procID, nil
		}
	}

	procID, err := srv.db.AddProc(db.CreateQuery{
		Name:       query.Name.OrDefault(namegen.New()),
		Cwd:        query.Cwd,
		Tags:       fun.Uniq(append(query.Tags, "all")),
		Command:    query.Command,
		Args:       query.Args,
		Watch:      query.Watch,
		Env:        query.Env,
		StdoutFile: query.StdoutFile,
		StderrFile: query.StderrFile,
	}, srv.logsDir)
	if err != nil {
		return 0, xerr.NewWM(err, "save proc")
	}

	return procID, nil
}

func (srv *daemonServer) Create(_ context.Context, req *pb.CreateRequest) (*pb.ProcID, error) {
	procID, err := srv.create(CreateQuery{
		Name:       fun.FromPtr(req.Name),
		Cwd:        req.GetCwd(),
		Tags:       req.GetTags(),
		Command:    req.GetCommand(),
		Args:       req.GetArgs(),
		Watch:      fun.FromPtr(req.Watch),
		Env:        req.GetEnv(),
		StdoutFile: fun.FromPtr(req.StdoutFile),
		StderrFile: fun.FromPtr(req.StderrFile),
	})
	if err != nil {
		return nil, err
	}

	return &pb.ProcID{
		Id: procID,
	}, nil
}
