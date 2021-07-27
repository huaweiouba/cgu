package main

// An example demonstrating an application with multiple views.
//
// Note that this example was produced before the Bubbles progress component
// was available (github.com/charmbracelet/bubbles/progress) and thus, we're
// implementing a progress bar from scratch here.

import (
	"fmt"
	"github.com/gookit/color"
	"math"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/fogleman/ease"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/muesli/reflow/indent"
	"github.com/muesli/termenv"
)

const (
	progressBarWidth  = 71
	progressFullChar  = "█"
	progressEmptyChar = "░"
)

// General stuff for styling the view
var (
	term          = termenv.ColorProfile()
	subtle        = makeFgStyle("241")
	progressEmpty = subtle(progressEmptyChar)
	dot           = colorFg(" • ", "236")

	// Gradient colors we'll use for the progress bar
	ramp = makeRamp("#B14FFF", "#00FFA3", progressBarWidth)
)

func chooseUser(isGlobal bool) error {
	allUsers, err := getAllUser()
	if err != nil {
		return err
	}
	var oldChoice int
	var choose int

	if isGlobal {
		oldChoice = 0
		choose = 1
	} else {
		if !isGitDir() {
			return ErrNotGitDir
		}
		oldChoice, err = getNowSelectedUser(allUsers)
		if err != nil {
			return err
		}
		choose = 0
		if oldChoice == 0 {
			choose = 1
		}
	}
	initialModel := chooseModel{oldChoice, choose, allUsers, false, 30, 0, 0, false, false, isGlobal, false, false}
	p := tea.NewProgram(initialModel)
	if err := p.Start(); err != nil {
		return fmt.Errorf("加载bubbletea列表失败: %w", err)
	}
	return nil
}

func delUser() error {
	allUsers, err := getAllUser()
	if err != nil {
		return err
	}
	initialModel := chooseModel{0, 1, allUsers, false, 30, 0, 0, false, false, false, true, false}
	p := tea.NewProgram(initialModel)
	if err := p.Start(); err != nil {
		return fmt.Errorf("加载bubbletea列表失败: %w", err)
	}
	return nil
}

type tickMsg struct{}
type frameMsg struct{}

func tick() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg {
		return tickMsg{}
	})
}

func frame() tea.Cmd {
	return tea.Tick(time.Second/60, func(time.Time) tea.Msg {
		return frameMsg{}
	})
}

type chooseModel struct {
	oldChoice   int
	Choice      int
	ChoiceSlice [][]string
	Chosen      bool
	Ticks       int
	Frames      int
	Progress    float64
	Loaded      bool
	Quitting    bool
	IsGlobal    bool
	IsDel       bool
	IsQQuit     bool
}

func (m chooseModel) Init() tea.Cmd {
	return tick()
}

// Update Main update function.
func (m chooseModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Make sure these keys always quit
	if msg, ok := msg.(tea.KeyMsg); ok {
		k := msg.String()
		if k == "q" || k == "esc" || k == "ctrl+c" {
			m.Quitting = true
			m.IsQQuit = true
			return m, tea.Quit
		}
	}

	// Hand off the message and model to the appropriate update function for the
	// appropriate view based on the current state.
	if !m.Chosen {
		return updateChoices(msg, m)
	}
	return updateChosen(msg, m)
}

// View The main view, which just calls the appropriate sub-view
func (m chooseModel) View() string {
	var s string
	if m.Quitting {
		if m.IsQQuit {
			return "\n  👋再见👋\n\n" //👋see you later👋
		} else {
			if m.IsDel {
				return "\n  删除成功！\n\n"
			} else {
				gName, gEmail, err := getGlobalUser()
				if err != nil {
					return "\n  获取全局用户失败\n\n"
				}
				color256 := color.C256(211)
				if m.IsGlobal {

					return "\n  设置全局成功 name=" + color256.Sprint(gName) + " email=" + color256.Sprint(gEmail) + "\n\n"
				} else {
					pName, pEmail, err := getProjectUser()
					if err != nil {
						return "\n  获取当前目录用户失败\n\n"
					}
					nowGitPath, err := getNowGitPath()
					if err != nil {
						return "\n  获取当前目录git路径失败\n\n"
					}
					return "\n  当前目录使用 name=" + color256.Sprint(pName) + " email=" + color256.Sprint(pEmail) + " (作用于" + nowGitPath + ")\n\n"
				}
			}
		}

	}
	if !m.Chosen {
		s = choicesView(m)
	} else {
		s = chosenView(m)
	}
	return indent.String("\n"+s+"\n\n", 2)
}

// Sub-update functions

// Update loop for the first view where you're choosing a task.
func updateChoices(msg tea.Msg, m chooseModel) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			m.Choice += 1
			if m.oldChoice == m.Choice {
				m.Choice += 1
			}
			if m.Choice > len(m.ChoiceSlice)-1 {
				m.Choice = 0
				if m.oldChoice == m.Choice {
					m.Choice += 1
				}
			}

		case "k", "up":
			m.Choice -= 1
			if m.oldChoice == m.Choice {
				m.Choice -= 1
			}
			if m.Choice < 0 {
				m.Choice = len(m.ChoiceSlice) - 1
				if m.oldChoice == m.Choice {
					m.Choice -= 1
				}
			}

		case "enter":
			m.Chosen = true
			return m, frame()
		}

	case tickMsg:
		if m.Ticks == 0 {
			m.Quitting = true
			m.IsQQuit = true
			return m, tea.Quit
		}
		m.Ticks -= 1
		return m, tick()
	}

	return m, nil
}

// Update loop for the second view after a choice has been made
func updateChosen(msg tea.Msg, m chooseModel) (tea.Model, tea.Cmd) {
	switch msg.(type) {

	case frameMsg:
		if !m.Loaded {
			m.Frames += 2
			m.Progress = ease.OutBounce(float64(m.Frames) / float64(100))
			if m.Progress >= 1 {
				m.Progress = 1
				m.Loaded = true
				m.Ticks = 0
				return m, tick()
				//m.Quitting = true
				//return m, tea.Quit
			}
			return m, frame()
		}

	case tickMsg:
		if m.Loaded {
			if m.Ticks == 0 {
				m.Quitting = true
				return m, tea.Quit
			}
			m.Ticks -= 1
			return m, tick()
		}
	}

	return m, nil
}

// Sub-views

// The first view, where you're choosing a task
func choicesView(m chooseModel) string {
	c := m.Choice

	var tpl string
	if m.IsDel {
		tpl = "请选择您要删除的账号\n\n"
	} else {
		if m.IsGlobal {
			tpl = "请选择您要设置的全局账号\n\n"
		} else {
			tpl = "请选择您要使用的账号\n\n"
		}
	}

	tpl += "%s\n\n"
	tpl += "若无选择，程序将在 %s 秒后自动退出\n\n"
	tpl += subtle("↑/↓选择") + dot + subtle("回车确认") + dot + subtle("q退出")

	max := 0
	for _, v := range m.ChoiceSlice {
		max = int(math.Max(float64(len(v[0])), float64(max)))
	}
	var choices string
	for k, v := range m.ChoiceSlice {
		temp := "name:" + getBlank(v[0], max) + "  email:" + v[1]
		choices += fmt.Sprintf("%s\n", checkbox(temp, k == c, k == m.oldChoice, m.IsDel))
	}
	choices = strings.TrimRight(choices, "\n")

	return fmt.Sprintf(tpl, choices, colorFg(strconv.Itoa(m.Ticks), "79"))
}
func getBlank(str string, max int) string {
	if len(str) < max {
		len := max - len(str)
		for i := 0; i < len; i++ {
			str += " "
		}
	}
	return str
}

// The second view, after a task has been chosen
func chosenView(m chooseModel) string {
	var msg string
	var label string

	if m.IsDel {
		msg = "您将删除name=" + m.ChoiceSlice[m.Choice][0] + " email=" + m.ChoiceSlice[m.Choice][1]
		err := doDel(m.ChoiceSlice[m.Choice][0], m.ChoiceSlice[m.Choice][1])
		label = "删除中..."
		if m.Loaded {
			if err != nil {
				label = fmt.Sprintf("删除失败，将在 %s 秒后自动退出...", colorFg(strconv.Itoa(m.Ticks), "79"))
			} else {
				label = fmt.Sprintf("删除成功，将在 %s 秒后自动退出...", colorFg(strconv.Itoa(m.Ticks), "79"))
			}
		}
	} else {
		msg = "您将设置name=" + m.ChoiceSlice[m.Choice][0] + " email=" + m.ChoiceSlice[m.Choice][1]
		err := doUse(m.ChoiceSlice[m.Choice][0], m.ChoiceSlice[m.Choice][1], m.IsGlobal)
		label = "设置中..."
		if m.Loaded {
			if err != nil {
				label = fmt.Sprintf("设置失败，将在 %s 秒后自动退出...", colorFg(strconv.Itoa(m.Ticks), "79"))
			} else {
				label = fmt.Sprintf("设置成功，将在 %s 秒后自动退出...", colorFg(strconv.Itoa(m.Ticks), "79"))
			}
		}
	}
	return msg + "\n\n" + label + "\n" + progressbar(m.Progress) + "%"
}

func checkbox(label string, checked bool, oldSelected bool, isDel bool) string {
	if checked {
		return colorFg("[x] "+label, "212")
	}
	if oldSelected {
		if isDel {
			return colorFg(fmt.Sprintf("[ ] %s", label+"(全局用户不样删除🙅)"), "243")
		} else {
			return colorFg(fmt.Sprintf("[ ] %s", label+"(当前使用)"), "243")
		}
	}
	return fmt.Sprintf("[ ] %s", label)
}

func progressbar(percent float64) string {
	w := float64(progressBarWidth)

	fullSize := int(math.Round(w * percent))
	var fullCells string
	for i := 0; i < fullSize; i++ {
		fullCells += termenv.String(progressFullChar).Foreground(term.Color(ramp[i])).String()
	}

	emptySize := int(w) - fullSize
	emptyCells := strings.Repeat(progressEmpty, emptySize)

	return fmt.Sprintf("%s%s %3.0f", fullCells, emptyCells, math.Round(percent*100))
}

// Utils

// Color a string's foreground with the given value.
func colorFg(val, color string) string {
	return termenv.String(val).Foreground(term.Color(color)).String()
}

// Return a function that will colorize the foreground of a given string.
func makeFgStyle(color string) func(string) string {
	return termenv.Style{}.Foreground(term.Color(color)).Styled
}

// Generate a blend of colors.
func makeRamp(colorA, colorB string, steps float64) (s []string) {
	cA, _ := colorful.Hex(colorA)
	cB, _ := colorful.Hex(colorB)

	for i := 0.0; i < steps; i++ {
		c := cA.BlendLuv(cB, i/steps)
		s = append(s, colorToHex(c))
	}
	return
}

// Convert a colorful.Color to a hexadecimal format compatible with termenv.
func colorToHex(c colorful.Color) string {
	return fmt.Sprintf("#%s%s%s", colorFloatToHex(c.R), colorFloatToHex(c.G), colorFloatToHex(c.B))
}

// Helper function for converting colors to hex. Assumes a value between 0 and
// 1.
func colorFloatToHex(f float64) (s string) {
	s = strconv.FormatInt(int64(f*255), 16)
	if len(s) == 1 {
		s = "0" + s
	}
	return
}
