package db

import (
	"fmt"

	"github.com/rprtr258/pm/internal/core"
)

type Error interface {
	isError()
	error
}

type GetTableError struct{ Table string }

func (GetTableError) isError() {}

func (err GetTableError) Error() string {
	return fmt.Sprintf("get table %q", err.Table)
}

type ProcNotFoundError struct{ ProcID core.ProcID }

func (ProcNotFoundError) isError() {}

func (err ProcNotFoundError) Error() string {
	return fmt.Sprintf("proc #%d not found", err.ProcID)
}

type FlushError struct{ Err error }

func (FlushError) isError() {}

func (err FlushError) Error() string {
	return fmt.Sprintf("db flush: %s", err.Err.Error())
}
