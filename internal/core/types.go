package core

import (
	"encoding/json"
	"fmt"

	"github.com/rprtr258/fun"
	"github.com/rprtr258/xerr"
	"github.com/samber/lo"
)

type RunConfig struct {
	Args    []string
	Tags    []string
	Command string
	Cwd     string
	Name    fun.Option[string]
}

func (cfg *RunConfig) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name    *string  `json:"name"`
		Cwd     string   `json:"cwd"`
		Command string   `json:"command"`
		Args    []any    `json:"args"`
		Tags    []string `json:"tags"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return xerr.NewWM(err, "json.unmarshal")
	}

	*cfg = RunConfig{
		Name:    fun.FromPtr(tmp.Name),
		Cwd:     tmp.Cwd,
		Command: tmp.Command,
		Args: lo.Map(
			tmp.Args,
			func(elem any, _ int) string {
				return fmt.Sprint(elem)
			},
		),
		Tags: tmp.Tags,
	}

	return nil
}
