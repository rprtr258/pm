package table

import (
	"strings"

	"github.com/muesli/ansi"
	"github.com/muesli/reflow/wrap"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
)

func safeRepeat(s string, n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat(s, n)
}

type Table struct {
	Headers               []string
	Rows                  [][]string
	HaveInnerRowsDividers bool
}

const (
	E = 1 << iota
	W
	S
	N
)

// NSWE
var borders = [1 << 4]string{
	N | S | W | E: scuf.String("┼", scuf.FgBlack),
	N | S | E:     scuf.String("├", scuf.FgBlack),
	N | S | W:     scuf.String("┤", scuf.FgBlack),
	N | W | E:     scuf.String("┴", scuf.FgBlack),
	S | W | E:     scuf.String("┬", scuf.FgBlack),
	N | E:         scuf.String("╰", scuf.FgBlack),
	N | W:         scuf.String("╯", scuf.FgBlack),
	S | W:         scuf.String("╮", scuf.FgBlack),
	S | E:         scuf.String("╭", scuf.FgBlack),
	W | E:         scuf.String("─", scuf.FgBlack),
	N | S:         scuf.String("│", scuf.FgBlack),
}

func mywordwrap(w int, s string) []string {
	res := []string{}
	for _, line := range strings.Split(s, "\n") {
		part := ""
		for _, word := range strings.Fields(line) {
			newPart := part
			if part != "" {
				newPart += " "
			}
			newPart += word

			if ansi.PrintableRuneWidth(newPart) <= w {
				part = newPart
			} else {
				res = append(res, part)
				part = word
			}
		}
		if part != "" {
			res = append(res, part)
		}
	}
	return res
}

// NOTE: stupid piece of shit
func mywrap(w int, s string) []string {
	{
		// try softwrap first, as wrap doesnt do it
		softs := mywordwrap(w, s)
		for _, part := range softs {
			if ansi.PrintableRuneWidth(part) > w {
				goto HARD
			}
		}
		return softs
	}
HARD:

	wrapper := wrap.NewWriter(w)
	wrapper.KeepNewlines = true
	wrapper.PreserveSpace = false
	_, _ = wrapper.Write([]byte(s))
	return strings.Split(wrapper.String(), "\n")
}

// determine columns widths
func cols(t Table, w int) []int {
	res := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		if res[i] != 0 {
			continue
		}

		res[i] = ansi.PrintableRuneWidth(h) + 2
		for _, row := range t.Rows {
			for _, line := range strings.Split(row[i], "\n") {
				res[i] = max(res[i], ansi.PrintableRuneWidth(line)+2)
			}
		}
	}

	colsTotal := 0
	for _, c := range res {
		colsTotal += c
	}

	if wContent := w - (len(t.Headers)*1 + 1); colsTotal > wContent {
		res = fun.Map[int](func(col int) int {
			return col * wContent / colsTotal
		}, res...)
	}

	return res
}

func renderShort(t Table, w int) string {
	res := []string{}
	for _, row := range t.Rows {
		res = append(res, strings.Join(fun.Map[string](func(r string, i int) string {
			header := t.Headers[i]
			subLen := ansi.PrintableRuneWidth(header) +
				ansi.PrintableRuneWidth(r)
			if subLen <= w { // single line
				return header + safeRepeat(" ", w-subLen) + r
			}
			return t.Headers[i] + " " + r
		}, row...), "\n"))
	}
	return strings.Join(res, "\n"+safeRepeat(borders[W|E], w)+"\n")
}

func Render(t Table, w int) string {
	// if should go short, go short
	{
		width := len(t.Headers) + 1
		for i, header := range t.Headers {
			colWidth := ansi.PrintableRuneWidth(header)
			for _, row := range t.Rows {
				for _, line := range strings.Split(row[i], "\n") {
					colWidth = max(colWidth, ansi.PrintableRuneWidth(line))
				}
			}
			width += colWidth
		}
		if width >= w*2 {
			return renderShort(t, w)
		}
	}

	cols := cols(t, w)

	_we, _ns, _nswe := borders[W|E], borders[N|S], borders[N|S|W|E]

	line0 := make([]string, len(cols))
	line1 := make([]string, len(cols))
	for i, col := range cols {
		if col == 0 {
			continue
		}

		line0[i] = safeRepeat(_we, col)

		header := mywrap(col, t.Headers[i])[0]
		totalPadding := col - ansi.PrintableRuneWidth(header)
		line1[i] = safeRepeat(" ", totalPadding/2) + header + safeRepeat(" ", totalPadding-totalPadding/2)
	}
	lines := []string{
		borders[S|E] + strings.Join(line0, borders[S|W|E]) + borders[S|W],
		_ns + strings.Join(line1, _ns) + _ns,
		borders[N|S|E] + strings.Join(line0, _nswe) + borders[N|S|W],
	}
	for _, row := range t.Rows {
		wraps := fun.Map[[]string](func(col, j int) []string {
			return mywrap(col, row[j])
		}, cols...)

		linesTotal := 0
		for _, wrapLines := range wraps {
			linesTotal = max(linesTotal, len(wrapLines))
		}

		for k := 0; k < linesTotal; k++ {
			line := fun.Map[string](func(col, j int) string {
				part := ""
				if k < len(wraps[j]) {
					part = wraps[j][k]
				}

				totalPadding := col - ansi.PrintableRuneWidth(part) - 1
				return " " + part + safeRepeat(" ", totalPadding)
			}, cols...)
			lines = append(lines, _ns+strings.Join(line, _ns)+_ns)
		}
	}
	lines = append(lines, borders[N|E]+strings.Join(line0, borders[N|W|E])+borders[N|W])

	return strings.Join(lines, "\n")
}
