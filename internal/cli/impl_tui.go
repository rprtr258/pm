package cli

import (
	"cmp"
	"context"
	"fmt"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/acarl005/stripansi"
	key2 "github.com/charmbracelet/bubbles/key"
	tea2 "github.com/charmbracelet/bubbletea"
	"github.com/rprtr258/fun"
	"github.com/rprtr258/fun/set"
	"github.com/rprtr258/scuf"
	"github.com/rprtr258/tea"
	"github.com/rprtr258/tea/components/headless/list"
	"github.com/rprtr258/tea/components/help"
	"github.com/rprtr258/tea/components/key"
	"github.com/rprtr258/tea/components/tablebox"
	"github.com/rprtr258/tea/styles"

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
	// app controls
	Quit key.Binding
	// proc list
	Up, Down, Stop, Start, Delete key.Binding
	// logs list
	Switch, LogUp, LogDown key.Binding
}{
	Up:     key.Binding{[]string{tea2.KeyUp.String(), "k"}, key.Help{"↑/k", "move up"}, false},
	Down:   key.Binding{[]string{tea2.KeyDown.String(), "j"}, key.Help{"↓/j", "move down"}, false},
	Stop:   key.Binding{[]string{"x"}, key.Help{"x", "stop"}, false},
	Start:  key.Binding{[]string{tea2.KeyEnter.String()}, key.Help{"Enter", "start"}, false},
	Delete: key.Binding{[]string{"d"}, key.Help{"d", "delete"}, false},
	Quit:   key.Binding{[]string{"q", "esc", "ctrl+c"}, key.Help{"q", "quit"}, false},
	Switch: key.Binding{
		[]string{"h"},
		key.Help{"h", "show all/stderr/stdout logs"}, // TODO: help based on current value
		false,
	},
	LogUp:   key.Binding{[]string{tea2.KeyPgUp.String()}, key.Help{"PageUp", "scroll logs up"}, false},
	LogDown: key.Binding{[]string{tea2.KeyPgDown.String()}, key.Help{"PageDown", "scroll logs down"}, false},
}

func compatMatches(msg tea2.KeyMsg, keyy key.Binding) bool {
	return key2.Matches(msg, key2.NewBinding(
		key2.WithKeys(keyy.Keys...),
		key2.WithHelp(keyy.Help[0], keyy.Help[1]),
	))
}

type logTypeShow int8

const (
	logTypeShowAll logTypeShow = 0
	logTypeShowErr logTypeShow = 1
	logTypeShowOut logTypeShow = 2
)

func normalizePathRelativeToHomeDir(path string, homeDir fun.Option[string]) string {
	if homePath, ok := homeDir.Unpack(); ok {
		if after, ok := strings.CutPrefix(path, homePath); ok {
			return "~" + after
		}
	}
	return path
}

type model struct {
	ids    []core.PMID
	db     db.Handle
	cfg    core.Config
	logsCh <-chan core.LogLine

	dispatch func(msg tea2.Msg)
	size     [2]int
	homeDir  fun.Option[string]

	list    *list.List[core.ProcStat]
	keysMap help.KeyMap
	help    help.Model

	logsList    map[core.PMID][]core.LogLine
	logTypeShow logTypeShow
	logScroll   int
}

type msgRefresh struct{}

type msgLog struct{ line core.LogLine }

func (m *model) Init() tea2.Cmd {
	m.help.ShortSeparator = "  " // TODO: crumbs like in zellij
	m.help.ShowAll = true

	go func() {
		for line := range m.logsCh {
			m.dispatch(msgLog{line})
		}
	}()
	go m.dispatch(tea.EnterAltScreen())
	go m.dispatch(msgRefresh{})
	return nil
}

func (m *model) resetLogScroll() {
	if m.list.Total() > 0 {
		m.logScroll = 0
	}
}

func (m *model) updateKey(msg tea2.KeyMsg) tea2.Cmd {
	switch {
	case compatMatches(msg, keymap.Up):
		m.resetLogScroll()
		m.list.SelectPrev()
	case compatMatches(msg, keymap.Down):
		m.resetLogScroll()
		m.list.SelectNext()
	case compatMatches(msg, keymap.Stop):
		proc, ok := m.list.Selected()
		if !ok {
			return nil
		}
		// TODO: remove logs here
		_ = implStop(m.db, proc.ID) // TODO: show error
	case compatMatches(msg, keymap.Start):
		proc, ok := m.list.Selected()
		if !ok {
			return nil
		}
		// TODO: remove logs here
		_ = implStart(m.db, proc.ID) // TODO: show error
	case compatMatches(msg, keymap.Delete):
		proc, ok := m.list.Selected()
		if !ok {
			return nil
		}
		// TODO: remove logs here
		_ = implDelete(m.db, cfg.DirLogs, proc.ID) // TODO: show error
	case compatMatches(msg, keymap.Quit):
		return tea2.Quit
	case compatMatches(msg, keymap.Switch):
		m.resetLogScroll()
		switch m.logTypeShow {
		case logTypeShowAll:
			m.logTypeShow = logTypeShowErr
		case logTypeShowErr:
			m.logTypeShow = logTypeShowOut
		case logTypeShowOut:
			m.logTypeShow = logTypeShowAll
		}
	case compatMatches(msg, keymap.LogUp):
		// TODO: limit until first line
		m.logScroll++
	case compatMatches(msg, keymap.LogDown):
		// TODO: limit until last line
		m.logScroll--
	}
	return nil
}

func (m *model) update(msg tea2.Msg) tea2.Cmd {
	switch msg := msg.(type) {
	case tea2.KeyMsg:
		return m.updateKey(msg)
	case msgRefresh:
		procs := listProcs(m.db).Filter(func(ps core.ProcStat) bool {
			return fun.Contains(ps.ID, m.ids...)
		}).Slice()
		slices.SortFunc(procs, func(a, b core.ProcStat) int {
			return cmp.Compare(a.ID, b.ID)
		})

		m.list.ItemsSet(procs)
		if _, ok := m.list.Selected(); !ok {
			m.list.Select(0)
		}

		return tea2.Tick(_refreshInterval, func(time.Time) tea2.Msg {
			return msgRefresh{}
		})
	case msgLog:
		procID := msg.line.ProcID
		m.logsList[procID] = append(m.logsList[procID], msg.line)
	case tea2.WindowSizeMsg:
		m.size = [2]int{msg.Height, msg.Width}
	}
	return nil
}

func (m *model) Update(msg tea2.Msg) (tea2.Model, tea2.Cmd) {
	cmd := m.update(msg)
	return m, cmd
}

func (m *model) viewInfo(vb tea.Viewbox) {
	proc, ok := m.list.Selected()
	if !ok {
		return
	}

	cwd := normalizePathRelativeToHomeDir(proc.Cwd, m.homeDir)
	cmd := normalizePathRelativeToHomeDir(proc.Command, m.homeDir)

	y := 0
	vb.Row(y).WriteLineX(fmt.Sprintf("ID: %s", proc.ID))
	y++
	vb.Row(y).WriteLineX(fmt.Sprintf("NAME: %s", proc.Name))
	y++
	vb.Row(y).WriteLineX(fmt.Sprintf("CWD: %s", cwd))
	y++
	vb.Row(y).WriteLineX(fmt.Sprintf("CMD: %s", cmd))
	y++
	vb.Row(y).
		WriteLineX("STARTUP: ").
		Styled(styles.Style{}.Foreground(fun.IF(proc.Startup, scuf.FgHiGreen, scuf.FgHiBlack))).
		WriteLine(fun.IF(proc.Startup, "on", "off"))
	y++
	if watch, ok := proc.Watch.Unpack(); ok {
		vb.Row(y).WriteLineX(fmt.Sprintf("WATCH: %s", watch))
		y++
	}
	if len(proc.DependsOn) > 0 {
		vb.Row(y).WriteLineX(fmt.Sprintf("DEPENDS: %s", strings.Join(proc.DependsOn, ",")))
		y++
	}
	_ = y
}

func (m *model) viewLogs(vb tea.Viewbox) {
	proc, ok := m.list.Selected()
	if !ok {
		return
	}

	logs := m.logsList[proc.ID]
	if m.logTypeShow != logTypeShowAll {
		logs = fun.Filter(func(line core.LogLine) bool {
			return m.logTypeShow == logTypeShowErr && line.Type == core.LogTypeStderr ||
				m.logTypeShow == logTypeShowOut && line.Type == core.LogTypeStdout
		}, logs...)
	}

	// TODO: show last N log lines, update channels on refresh, update log lines on new log lines
	for i, line := range fun.Subslice(max(0, len(logs)-max(vb.Height+m.logScroll, 1)), vb.Height, logs...) {
		vbRow := vb.Row(i)

		cleanLine := strings.ReplaceAll(stripansi.Strip(line.Line), "\r", "")
		x0 := vbRow.
			Styled(styles.Style{}.Foreground(styles.FgColor(fun.IF(
				line.Type == core.LogTypeStdout,
				"#00FF00",
				"#FF0000",
			)))).
			WriteLine(" │ ")
		// TODO: dim stderr
		vbRow.
			PaddingLeft(x0).
			WriteLineX(cleanLine)
	}
}

func (m *model) view(vb tea.Viewbox) {
	// TODO: pane headers
	vbPane, vbHelp := vb.SplitY2(tea.Auto(), tea.Fixed(3))
	for i, vb := range tablebox.Box(
		vbPane,
		tablebox.NormalBorder,
		styles.Style{}.Foreground(styles.ANSIColor(238)),
		tablebox.Span(
			tea.Flex(1),
			tablebox.SpanTitle("Info", tea.Fixed(7)),
			tablebox.SpanTitle("Procs", tea.Auto()),
			tablebox.SpanTitle("Tags", tea.Fixed(3)),
		),
		tablebox.SpanTitle(
			"Logs",
			tea.Flex(3),
		),
	) {
		switch i {
		case 0: // info
			m.viewInfo(vb)
		case 1: // procs list
			for i := 0; i < min(vb.Height, m.list.Total()); i++ {
				var (
					statusColor scuf.Modifier
					statusChar  string
				)
				switch m.list.ItemsAll()[i].Status {
				case core.StatusCreated:
					statusColor = scuf.FgHiYellow
					statusChar = "❍ "
				case core.StatusRunning:
					statusColor = scuf.FgHiGreen
					statusChar = "✔ "
				case core.StatusStopped:
					statusColor = scuf.Combine(scuf.FgRed, scuf.ModBold)
					statusChar = "❌"
				}

				style := fun.IF(i == m.list.SelectedIndex(), listItemStyleSelected, listItemStyle)
				vbRow := vb.Row(i).Styled(style)
				x0 := vbRow.Styled(style.Foreground(statusColor)).WriteLine(statusChar)
				vbRow.PaddingLeft(x0).WriteLine(m.list.ItemsAll()[i].Name)
			}
		case 2: // tags
			tagsSet := set.New[string](1)
			for _, proc := range m.list.ItemsAll() {
				for _, tag := range proc.Tags {
					tagsSet.Add(tag)
				}
			}

			tags := tagsSet.List()
			sort.Strings(tags)

			for i, tag := range tags[:min(vb.Height, len(tags))] {
				vb.Row(i).WriteLine(tag)
			}
		case 3: // logs
			m.viewLogs(vb)
		}
	}
	m.help.View(vbHelp, m.keysMap) // TODO: extended help on ?, short by default
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
	homeDirOpt := fun.Invalid[string]()
	if homeDir, err := os.UserHomeDir(); err == nil {
		homeDirOpt = fun.Valid(homeDir)
	}

	m := &model{
		ids:     ids,
		db:      db,
		cfg:     cfg,
		logsCh:  logsCh,
		homeDir: homeDirOpt,
		keysMap: help.KeyMap{
			ShortHelp: nil,
			FullHelp: [][]key.Binding{
				{keymap.Up, keymap.Down, keymap.Quit},
				{keymap.Stop, keymap.Start, keymap.Delete},
				{keymap.Switch, keymap.LogUp, keymap.LogDown},
			},
		},
		help:        help.New(),
		list:        list.New([]core.ProcStat{}),
		logsList:    map[core.PMID][]core.LogLine{},
		dispatch:    nil, // NOTE: set below
		size:        [2]int{},
		logTypeShow: logTypeShowAll,
		logScroll:   0,
	}
	p := tea2.NewProgram(m, tea2.WithContext(ctx))
	m.dispatch = p.Send
	_, err := p.Run()
	return err
}
