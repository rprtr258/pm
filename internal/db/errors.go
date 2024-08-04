package db

import (
	"fmt"

	"github.com/rprtr258/pm/internal/core"
)

type ProcNotFoundError struct{ ProcID core.PMID }

func (err ProcNotFoundError) Error() string {
	return fmt.Sprintf("proc #%s not found", err.ProcID)
}

type FlushError struct{ Err error }

func (err FlushError) Error() string {
	return fmt.Sprintf("db flush: %s", err.Err.Error())
}
