package core

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/lipgloss"

	tea "github.com/charmbracelet/bubbletea"
)

var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Padding(0, 1, 0, 0)

	selectedTitleStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				Padding(0, 0, 0, 1)

	selectedDescStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder(), false, false, false, true).
				Padding(0, 0, 0, 1)
)

type version struct {
	version *Version
	chosen  bool
}

func (v *version) Title() string {
	s := strings.Builder{}
	if v.chosen {
		s.WriteString("[x] ")
	} else {
		s.WriteString("[ ] ")
	}
	s.WriteString(v.version.Message)
	return s.String()
}

func (v *version) Description() string {
	return v.version.Hash
}

func (v *version) FilterValue() string {
	return v.version.Message
}

func (v *version) toggle() {
	v.chosen = !v.chosen
}

type viewModel struct {
	list     list.Model
	keys     *listKeyMap
	choices  []list.Item
	required bool
}

type delegateKeyMap struct {
	choose key.Binding
}

func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.choose,
	}
}

func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.choose,
		},
	}
}

type listKeyMap struct {
	togglePagination key.Binding
	toggleHelpMenu   key.Binding
	finish           key.Binding
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		togglePagination: key.NewBinding(
			key.WithKeys("P"),
			key.WithHelp("P", "toggle pagination"),
		),
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
		finish: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "finish"),
		),
	}
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		choose: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "choose"),
		),
	}
}

func newItemDelegate(keys *delegateKeyMap) list.DefaultDelegate {
	d := list.NewDefaultDelegate()

	d.Styles.SelectedTitle = selectedTitleStyle
	d.Styles.SelectedDesc = selectedDescStyle

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		switch msg := msg.(type) {
		case tea.KeyMsg:
			switch {
			case key.Matches(msg, keys.choose):
				m.SelectedItem().(*version).toggle()
			}
		}

		return nil
	}

	help := []key.Binding{keys.choose}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d
}

func newModel(title string, manifest *Manifest, required bool, selected []*Version) viewModel {
	var (
		listKeys     = newListKeyMap()
		delegateKeys = newDelegateKeyMap()
	)

	var versions []list.Item
	for i := len(manifest.Versions) - 1; i >= 0; i-- {
		v := manifest.Versions[i]

		chosen := false
		for _, s := range selected {
			if v.Equal(s) {
				chosen = true
				break
			}
		}

		versions = append(versions, &version{
			version: v,
			chosen:  chosen,
		})
	}

	delegate := newItemDelegate(delegateKeys)
	versionList := list.New(versions, delegate, 0, 0)
	versionList.DisableQuitKeybindings()
	versionList.Title = title
	versionList.Styles.Title = titleStyle
	versionList.AdditionalShortHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.finish,
		}
	}
	versionList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.togglePagination,
			listKeys.toggleHelpMenu,
		}
	}
	versionList.Filter = func(term string, targets []string) []list.Rank {
		ranks := list.DefaultFilter(term, targets)
		for _, rank := range ranks {
			for i := range rank.MatchedIndexes {
				rank.MatchedIndexes[i] += 4
			}
		}
		return ranks
	}
	return viewModel{versionList, listKeys, versions, required}
}

func (m viewModel) Init() tea.Cmd {
	return tea.EnterAltScreen
}

func (m viewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.togglePagination):
			m.list.SetShowPagination(!m.list.ShowPagination())
			return m, nil

		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.list.SetShowHelp(!m.list.ShowHelp())
			return m, nil

		case key.Matches(msg, m.keys.finish):
			if m.required && !m.HasChosen() {
				return m, m.list.NewStatusMessage("You must select at least one version!")
			}
			return m, tea.Quit
		}
	}

	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m viewModel) View() string {
	return appStyle.Render(m.list.View())
}

func (m viewModel) HasChosen() bool {
	for _, item := range m.choices {
		version := item.(*version)
		if version.chosen {
			return true
		}
	}
	return false
}

func (d *Dorothy) ChooseVersions(title string, required bool) ([]string, error) {
	return d.ChooseVersionsWithSelected(title, required, nil)
}

func (d *Dorothy) ChooseVersionsWithSelected(title string, required bool, selected []*Version) ([]string, error) {
	model := newModel(title, d.Manifest, required, selected)

	p := tea.NewProgram(model)
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	var parents []string
	for _, item := range m.(viewModel).choices {
		version := item.(*version)
		if version.chosen {
			parents = append(parents, version.version.Hash)
		}
	}

	return parents, nil
}
