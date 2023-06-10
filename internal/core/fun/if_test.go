package fun

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func _const[T any](value T) func() T {
	return func() T {
		return value
	}
}

func TestAll(t *testing.T) {
	for name, test := range map[string]struct {
		got  string
		want string
	}{
		"true": {
			got: If[string](true).Then("1").
				Else("2"),
			want: "1",
		},
		"false": {
			got: If[string](false).Then("1").
				Else("2"),
			want: "2",
		},
		"ThenF ElseF true": {
			got: If[string](true).ThenF(_const("1")).
				ElseF(_const("2")),
			want: "1",
		},
		"ThenF ElseF false": {
			got: If[string](false).ThenF(_const("1")).
				ElseF(_const("2")),
			want: "2",
		},
		"ElseIf Else 1": {
			got: If[string](true).Then("1").
				ElseIf(true).Then("2").
				Else("3"),
			want: "1",
		},
		"ElseIf Else 2": {
			got: If[string](false).Then("1").
				ElseIf(true).Then("2").
				Else("3"),
			want: "2",
		},
		"ElseIf Else 3": {
			got: If[string](false).Then("1").
				ElseIf(false).Then("2").
				Else("3"),
			want: "3",
		},
		"ElseIf ElseIf 3": {
			got: If[string](false).Then("1").
				ElseIf(false).Then("2").
				ElseIf(true).Then("3").
				Else("4"),
			want: "3",
		},
		"ElseIf ElseIf 4": {
			got: If[string](false).Then("1").
				ElseIf(false).Then("2").
				ElseIf(false).Then("3").
				Else("4"),
			want: "4",
		},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.want, test.got)
		})
	}
}
