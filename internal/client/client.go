package client

import (
	"context"
	"fmt"
	"net"

	"github.com/samber/lo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"

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
		internal.SocketDaemonRpc,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(ctx context.Context, s string) (net.Conn, error) {
			return net.Dial("unix", s)
		}),
	)
	if err != nil {
		return Client{}, fmt.Errorf("connecting to grpc server failed: %w", err)
	}

	return Client{
		client: api.NewDaemonClient(conn),
		conn:   conn,
	}, nil
}

func (c Client) Close() error {
	return c.conn.Close()
}

func (c Client) Create(ctx context.Context, r *api.ProcessOptions) (uint64, error) {
	resp, err := c.client.Create(ctx, r)
	if err != nil {
		return 0, err
	}

	return resp.GetId(), nil
}

func (c Client) List(ctx context.Context) (map[db.ProcID]db.ProcData, error) {
	resp, err := c.client.List(ctx, &emptypb.Empty{})
	if err != nil {
		return nil, err
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
			Cpu:       st.GetCpu(),
			Memory:    st.GetMemory(),
		}
	default:
		return db.Status{Status: db.StatusInvalid}
	}
}

func (c Client) Delete(ctx context.Context, ids []uint64) error {
	_, err := c.client.Delete(ctx, mapIDs(ids))
	return err
}

func (c Client) Start(ctx context.Context, ids []uint64) error {
	_, err := c.client.Start(ctx, mapIDs(ids))
	return err
}

func (c Client) Stop(ctx context.Context, ids []uint64) error {
	_, err := c.client.Stop(ctx, mapIDs(ids))
	return err
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
	_, err := c.client.HealthCheck(ctx, &emptypb.Empty{})
	return err
}
