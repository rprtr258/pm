package cli

import (
	"context"
	"strings"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/rprtr258/tea"
	"github.com/rprtr258/tea/components/headless/list"
	"github.com/rprtr258/tea/components/help"
	"github.com/rprtr258/tea/components/key"
	"github.com/rprtr258/tea/components/tablebox"
	"github.com/rprtr258/tea/styles"
	"github.com/rs/zerolog/log"

	"github.com/rprtr258/pm/internal/core"
	"github.com/rprtr258/pm/internal/db"
)

const _refreshInterval = time.Second

var (
	listItemStyleSelected = styles.Style{}.
				Background(scuf.BgANSI(61)).
				Foreground(scuf.FgANSI(230)).
				Bold(true)
	listItemStyle = styles.Style{}
)

type keyMap struct {
	Up, Down key.Binding
	Quit     key.Binding
}

type model struct {
	ids    []core.PMID
	db     db.Handle
	cfg    core.Config
	logsCh <-chan core.LogLine

	list     *list.List[core.Proc]
	keys     keyMap
	keysMap  help.KeyMap
	help     help.Model
	logsList *list.List[core.LogLine]
}

type msgRefresh struct{}

type msgLog struct{ line core.LogLine }

func (m *model) Init(f func(...tea.Cmd)) {
	m.keys = keyMap{
		Up: key.Binding{
			Keys: []string{"up", "k"},
			Help: key.Help{"↑/k", "move up"},
		},
		Down: key.Binding{
			Keys: []string{"down", "j"},
			Help: key.Help{"↓/j", "move down"},
		},
		Quit: key.Binding{
			Keys: []string{"q", "esc", "ctrl+c"},
			Help: key.Help{"q", "quit"},
		},
	}
	m.keysMap = help.KeyMap{
		ShortHelp: []key.Binding{m.keys.Up, m.keys.Down, m.keys.Quit},
		FullHelp:  nil,
	}

	m.help = help.New()
	m.help.ShortSeparator = "  " // TODO: crumbs like in zellij

	m.list = list.New([]core.Proc{})        /* func(i core.Proc) string { return i.Name }*/
	m.logsList = list.New([]core.LogLine{}) /* func(i core.LogLine) string { return i.Line }*/

	go func() {
		for line := range m.logsCh {
			f(func() tea.Msg { return msgLog{line} })
		}
	}()
	f(
		tea.EnterAltScreen,
		func() tea.Msg { return msgRefresh{} },
	)
}

func (m *model) Update(message tea.Msg, f func(...tea.Cmd)) {
	switch msg := message.(type) {
	case tea.MsgKey:
		switch {
		case key.Matches(msg, m.keys.Up):
			m.list.SelectPrev()
		case key.Matches(msg, m.keys.Down):
			m.list.SelectNext()
		case key.Matches(msg, m.keys.Quit):
			f(tea.Quit)
			return
		}
	case msgRefresh:
		procs, err := m.db.List(core.WithIDs(m.ids...))
		if err != nil {
			log.Error().Err(err).Msg("get procs")
			f(tea.Quit)
			return
		}

		m.list.ItemsSet(fun.FilterMap[core.Proc](func(id core.PMID) (core.Proc, bool) {
			proc, ok := procs[id]
			return proc, ok
		}, m.ids...))
		if _, ok := m.list.Selected(); !ok {
			m.list.Select(0)
		}

		f(tea.Tick(_refreshInterval, func(t time.Time) tea.Msg {
			return msgRefresh{}
		}))
		return
	case msgLog:
		m.logsList.ItemsSet(append(m.logsList.ItemsAll(), msg.line))
	}
}

func (m *model) View(vb tea.Viewbox) {
	vbPane, vbHelp := vb.SplitY2(tea.Auto(), tea.Fixed(1))
	tablebox.Box(
		vbPane,
		[]tea.Layout{tea.Flex(1)},
		[]tea.Layout{tea.Flex(1), tea.Flex(4)},
		func(vb tea.Viewbox, y, x int) {
			switch {
			case x == 0:
				for i := 0; i < min(vb.Height, m.list.Total()); i++ {
					vb := vb.Row(i)

					style := fun.IF(i == m.list.SelectedIndex(), listItemStyleSelected, listItemStyle)
					vb.Styled(style).WriteLine(m.list.ItemsAll()[i].Name)
				}
			case x == 1:
				// TODO: show last N log lines, update channels on refresh, update log lines on new log lines
				for i := 0; i < min(vb.Height, m.logsList.Total()); i++ {
					vb := vb.Row(i)

					// style := fun.IF(index == m.Index(), listItemStyleSelected, listItemStyle)
					item := m.logsList.ItemsAll()[i]
					cleanLine := strings.ReplaceAll(stripansi.Strip(item.Line), "\r", "")
					x0 := vb.
						Styled(styles.Style{}.Foreground(styles.FgColor("238"))).
						WriteLine(item.ProcName)
					x1 := vb.
						PaddingLeft(x0).
						Styled(styles.Style{}.Foreground(styles.FgColor(fun.IF(
							item.Type == core.LogTypeStdout,
							"#00FF00",
							"#FF0000",
						)))).
						WriteLine(" | ")
					vb.
						PaddingLeft(x0 + x1).
						WriteLineX(cleanLine)
				}
			}
		},
		tablebox.NormalBorder,
		styles.Style{}.Foreground(styles.ANSIColor(238)),
	)
	m.help.View(vbHelp, m.keysMap)
}

func tui(
	ctx context.Context,
	db db.Handle,
	cfg core.Config,
	logsCh <-chan core.LogLine,
	ids ...core.PMID,
) error {
	_, err := tea.NewProgram(ctx, &model{
		ids:    ids,
		db:     db,
		cfg:    cfg,
		logsCh: logsCh,
	}).Run()
	return err
}
