package cli

import (
	"context"
	"strings"
	"time"

	"github.com/acarl005/stripansi"
	"github.com/charmbracelet/bubbles/key"
	bubbletea "github.com/charmbracelet/bubbletea"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/scuf"
	"github.com/rprtr258/tea"
	"github.com/rprtr258/tea/components/headless/list"
	"github.com/rprtr258/tea/components/help"
	key2 "github.com/rprtr258/tea/components/key"
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

var keymap = struct {
	Up, Down, Quit, Switch key2.Binding
}{
	Up: key2.Binding{
		[]string{"up", "k"},
		key2.Help{"↑/k", "move up"},
		false,
	},
	Down: key2.Binding{
		[]string{"down", "j"},
		key2.Help{"↓/j", "move down"},
		false,
	},
	Quit: key2.Binding{
		[]string{"q", "esc", "ctrl+c"},
		key2.Help{"q", "quit"},
		false,
	},
	Switch: key2.Binding{
		[]string{"h"},
		key2.Help{"h", "Show all/stderr/stdout logs"},
		false,
	},
}

func compatMatches(msg bubbletea.KeyMsg, keyy key2.Binding) bool {
	return key.Matches(msg, key.NewBinding(
		key.WithKeys(keyy.Keys...),
		key.WithHelp(keyy.Help[0], keyy.Help[1]),
	))
}

type logTypeShow int8

const (
	logTypeShowAll logTypeShow = 0
	logTypeShowErr logTypeShow = 1
	logTypeShowOut logTypeShow = 2
)

type model struct {
	ids    []core.PMID
	db     db.Handle
	cfg    core.Config
	logsCh <-chan core.LogLine

	dispatch func(msg bubbletea.Msg)
	size     [2]int

	list    *list.List[core.Proc]
	keysMap help.KeyMap
	help    help.Model

	logsList    map[core.PMID][]core.LogLine
	logTypeShow logTypeShow
}

type msgRefresh struct{}

type msgLog struct{ line core.LogLine }

func (m *model) Init() bubbletea.Cmd {
	m.keysMap = help.KeyMap{
		ShortHelp: []key2.Binding{keymap.Up, keymap.Down, keymap.Quit, keymap.Switch},
	}

	m.help = help.New()
	m.help.ShortSeparator = "  " // TODO: crumbs like in zellij

	m.list = list.New([]core.Proc{}) /* func(i core.Proc) string { return i.Name }*/
	m.logsList = map[core.PMID][]core.LogLine{}

	go func() {
		for line := range m.logsCh {
			m.dispatch(msgLog{line})
		}
	}()
	go m.dispatch(tea.EnterAltScreen())
	go m.dispatch(msgRefresh{})
	return nil
}

func (m *model) update(msg bubbletea.Msg) bubbletea.Cmd {
	switch msg := msg.(type) {
	case bubbletea.KeyMsg:
		switch {
		case compatMatches(msg, keymap.Up):
			m.list.SelectPrev()
		case compatMatches(msg, keymap.Down):
			m.list.SelectNext()
		case compatMatches(msg, keymap.Quit):
			return bubbletea.Quit
		case compatMatches(msg, keymap.Switch):
			switch m.logTypeShow {
			case logTypeShowAll:
				m.logTypeShow = logTypeShowErr
			case logTypeShowErr:
				m.logTypeShow = logTypeShowOut
			case logTypeShowOut:
				m.logTypeShow = logTypeShowAll
			}
		}
	case msgRefresh:
		procs, err := m.db.List(core.WithIDs(m.ids...))
		if err != nil {
			log.Error().Err(err).Msg("get procs")
			return bubbletea.Quit
		}

		m.list.ItemsSet(fun.FilterMap[core.Proc](func(id core.PMID) (core.Proc, bool) {
			proc, ok := procs[id]
			return proc, ok
		}, m.ids...))
		if _, ok := m.list.Selected(); !ok {
			m.list.Select(0)
		}

		return bubbletea.Tick(_refreshInterval, func(t time.Time) bubbletea.Msg {
			return msgRefresh{}
		})
	case msgLog:
		procID := msg.line.ProcID
		m.logsList[procID] = append(m.logsList[procID], msg.line)
	case bubbletea.WindowSizeMsg:
		m.size = [2]int{msg.Height, msg.Width}
	}

	return nil
}

func (m *model) Update(msg bubbletea.Msg) (bubbletea.Model, bubbletea.Cmd) {
	return m, m.update(msg)
}

func (m *model) view(vb tea.Viewbox) {
	vbPane, vbHelp := vb.SplitY2(tea.Auto(), tea.Fixed(1))
	for yx, vb := range tablebox.Box(
		vbPane,
		[]tea.Layout{tea.Flex(1)},
		[]tea.Layout{tea.Flex(1), tea.Flex(4)},
		tablebox.NormalBorder,
		styles.Style{}.Foreground(styles.ANSIColor(238)),
	) {
		switch x := yx[1]; x {
		case 0:
			for i := 0; i < min(vb.Height, m.list.Total()); i++ {
				vb := vb.Row(i)

				style := fun.IF(i == m.list.SelectedIndex(), listItemStyleSelected, listItemStyle)
				vb.Styled(style).WriteLine(m.list.ItemsAll()[i].Name)
			}
		case 1:
			proc, ok := m.list.Selected()
			if !ok {
				continue
			}

			logs := m.logsList[proc.ID]
			if m.logTypeShow != logTypeShowAll {
				logs = fun.Filter(func(line core.LogLine) bool {
					return m.logTypeShow == logTypeShowErr && line.Type == core.LogTypeStderr ||
						m.logTypeShow == logTypeShowOut && line.Type == core.LogTypeStdout
				}, logs...)
			}

			// TODO: show last N log lines, update channels on refresh, update log lines on new log lines
			for i, line := range logs[max(0, len(logs)-vb.Height):] {
				vb := vb.Row(i)

				// style := fun.IF(index == m.Index(), listItemStyleSelected, listItemStyle)
				cleanLine := strings.ReplaceAll(stripansi.Strip(line.Line), "\r", "")
				x0 := vb.
					Styled(styles.Style{}.Foreground(styles.FgColor(fun.IF(
						line.Type == core.LogTypeStdout,
						"#00FF00",
						"#FF0000",
					)))).
					WriteLine(" │ ")
				// TODO: dim stderr
				vb.
					PaddingLeft(x0).
					WriteLineX(cleanLine)
			}
		}
	}
	m.help.View(vbHelp, m.keysMap)
}

func (m *model) View() string {
	vb := tea.NewViewbox(m.size[0], m.size[1])
	m.view(vb)
	return string(vb.Render())
}

func tui(
	ctx context.Context,
	db db.Handle,
	cfg core.Config,
	logsCh <-chan core.LogLine,
	ids ...core.PMID,
) error {
	m := &model{
		ids:    ids,
		db:     db,
		cfg:    cfg,
		logsCh: logsCh,
	}
	p := bubbletea.NewProgram(m, bubbletea.WithContext(ctx))
	m.dispatch = p.Send
	_, err := p.Run()
	return err
}
