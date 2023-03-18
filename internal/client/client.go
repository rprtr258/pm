package client

import (
	"context"
	"net"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/rprtr258/xerr"

	"github.com/rprtr258/pm/api"
	"github.com/rprtr258/pm/internal"
	"github.com/rprtr258/pm/internal/db"
)

type Client struct {
	client api.DaemonClient
	conn   *grpc.ClientConn
}

func NewGrpcClient() (Client, error) {
	conn, err := grpc.Dial(
		internal.SocketDaemonRPC,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, addr string) (net.Conn, error) {
			conn, err := net.Dial("unix", addr)
			if err != nil {
				return nil, xerr.NewWM(err, "dial unix socket",
					xerr.Field("addr", addr))
			}

			return conn, nil
		}),
	)
	if err != nil {
		return Client{}, xerr.NewWM(err, "connect to grpc server")
	}

	return Client{
		client: api.NewDaemonClient(conn),
		conn:   conn,
	}, nil
}

func (c Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return xerr.NewWM(err, "close client")
	}

	return nil
}

func (c Client) Create(ctx context.Context, r *api.ProcessOptions) (uint64, error) {
	resp, err := c.client.Create(ctx, r)
	if err != nil {
		return 0, xerr.NewWM(err, "server.create")
	}

	return resp.GetId(), nil
}

func (c Client) List(ctx context.Context) (map[db.ProcID]db.ProcData, error) {
	resp, err := c.client.List(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, xerr.NewWM(err, "server.list")
	}

	return lo.SliceToMap(
		resp.GetList(),
		func(proc *api.Process) (db.ProcID, db.ProcData) {
			procID := db.ProcID(proc.GetId().GetId())
			return procID, db.ProcData{
				ProcID:  procID,
				Name:    proc.GetName(),
				Command: proc.GetCommand(),
				Args:    proc.GetArgs(),
				Status:  mapPbStatus(proc.GetStatus()),
				Tags:    proc.GetTags(),
				Cwd:     proc.GetCwd(),
				Watch:   nil,
			}
		},
	), nil
}

func mapPbStatus(status *api.ProcessStatus) db.Status {
	switch {
	case status.GetErrored() != nil:
		return db.Status{Status: db.StatusErrored}
	case status.GetInvalid() != nil:
		return db.Status{Status: db.StatusInvalid}
	case status.GetStarting() != nil:
		return db.Status{Status: db.StatusStarting}
	case status.GetStopped() != nil:
		return db.Status{Status: db.StatusStopped}
	case status.GetRunning() != nil:
		st := status.GetRunning()
		return db.Status{
			Status:    db.StatusRunning,
			Pid:       int(st.GetPid()),
			StartTime: st.GetStartTime().AsTime(),
			CPU:       st.GetCpu(),
			Memory:    st.GetMemory(),
		}
	default:
		return db.Status{Status: db.StatusInvalid}
	}
}

func (c Client) Delete(ctx context.Context, ids []uint64) error {
	if _, err := c.client.Delete(ctx, mapIDs(ids)); err != nil {
		return xerr.NewWM(err, "server.delete", xerr.Field("ids", ids))
	}
	return nil
}

func (c Client) Start(ctx context.Context, ids []uint64) error {
	if _, err := c.client.Start(ctx, mapIDs(ids)); err != nil {
		return xerr.NewWM(err, "server.start", xerr.Field("ids", ids))
	}
	return nil
}

func (c Client) Stop(ctx context.Context, ids []uint64) error {
	if _, err := c.client.Stop(ctx, mapIDs(ids)); err != nil {
		return xerr.NewWM(err, "server.stop", xerr.Field("ids", ids))
	}
	return nil
}

func mapIDs(ids []uint64) *api.IDs {
	return &api.IDs{
		Ids: lo.Map(
			ids,
			func(procID uint64, _ int) *api.ProcessID {
				return &api.ProcessID{
					Id: procID,
				}
			},
		),
	}
}

func (c Client) HealthCheck(ctx context.Context) error {
	if _, err := c.client.HealthCheck(ctx, &emptypb.Empty{}); err != nil {
		return xerr.NewW(err)
	}
	return nil
}
