package db

import (
	"fmt"

	"github.com/rprtr258/pm/internal/core"
)

type Error interface {
	isError()
	error
}

type ProcNotFoundError struct{ ProcID core.PMID }

func (ProcNotFoundError) isError() {}

func (err ProcNotFoundError) Error() string {
	return fmt.Sprintf("proc #%s not found", err.ProcID)
}

type FlushError struct{ Err error }

func (FlushError) isError() {}

func (err FlushError) Error() string {
	return fmt.Sprintf("db flush: %s", err.Err.Error())
}
