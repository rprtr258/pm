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

	dispatch func(msg bubbletea.Msg)
	size     [2]int

	list     *list.List[core.Proc]
	keys     keyMap
	keysMap  help.KeyMap
	help     help.Model
	logsList *list.List[core.LogLine]
}

type msgRefresh struct{}

type msgLog struct{ line core.LogLine }

func (m *model) Init() bubbletea.Cmd {
	m.keys = keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
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
		case key.Matches(msg, m.keys.Up):
			m.list.SelectPrev()
		case key.Matches(msg, m.keys.Down):
			m.list.SelectNext()
		case key.Matches(msg, m.keys.Quit):
			return bubbletea.Quit
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
		m.logsList.ItemsSet(append(m.logsList.ItemsAll(), msg.line))
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
