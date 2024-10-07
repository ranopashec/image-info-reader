package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/barasher/go-exiftool"
	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type model struct {
	fp          filepicker.Model
	res         table.Model
	isResHidden bool
	dir         string
	file        string
	quitting    bool
	err         error
}

func (m model) Init() tea.Cmd {
	return m.fp.Init()
}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

type clearErrorMsg struct{}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "d":
			if m.dir != "" {
				err := m.updateResFrom(m.dir)
				if err != nil {
					m.err = err
					return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
				}
			} else {
				m.isResHidden = false
				m.err = errors.New("dir not selected")
				return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
			}
		case "f":
			if m.dir != "" && m.file != "" {
				err := m.updateResFrom(m.dir + "/" + m.file)
				if err != nil {
					m.err = err
					return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
				}
			} else {
				m.isResHidden = false
				m.err = errors.New("file not selected")
				return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
			}
		}
	case clearErrorMsg:
		m.err = nil
	}

	m.fp, cmd = m.fp.Update(msg)

	if didSelect, path := m.fp.DidSelectFile(msg); didSelect {
		fmt.Println(path)
		m.dir = filepath.Dir(path)
		m.file = filepath.Base(path)
	}

	if didSelect, path := m.fp.DidSelectDisabledFile(msg); didSelect {
		m.err = errors.New(path + " is not valid.")
		m.file = ""
		m.dir = ""
		return m, tea.Batch(cmd, clearErrorAfter(2*time.Second))
	}

	return m, cmd
}

func (m model) View() string {
	if m.quitting {
		// TODO ANIMATION
		return ""
	}
	var s strings.Builder
	s.WriteString("\n  ")
	if m.err != nil {
		s.WriteString(m.fp.Styles.DisabledFile.Render(m.err.Error()))
		m.isResHidden = true
	} else {
		if !m.isResHidden {
			s.WriteString("\n" + m.res.View() + "\n\n")
		}
		if m.file == "" || m.dir == "" {
			s.WriteString("Pick file (arrows to navigate) (press enter to confirm)")
		} else {
			s.WriteString("Press \"d\" to get info about all files in " + m.fp.Styles.Selected.Render(m.dir) + "\n Press \"f\" t get info only about " + m.fp.Styles.Selected.Render(m.file) + "\n")
		}
	}
	s.WriteString("\n\n" + m.fp.View() + "\n")
	return s.String()
}

func main() {
	fp := filepicker.New()
	fp.AllowedTypes = []string{".png", ".jpg", ".gif", ".tif", ".bmp", ".pcx"}
	fp.CurrentDirectory, _ = os.UserHomeDir()

	/* TABLE STYLE */
	columns := []table.Column{
		{Title: "Name", Width: 20},
		{Title: "Res", Width: 10},
		{Title: "DPI", Width: 10},
		{Title: "Depth", Width: 10},
		{Title: "Compression", Width: 10},
	}
	res := table.New(table.WithColumns(columns), table.WithRows([]table.Row{}))
	s := table.DefaultStyles()
	s.Header = s.Header.
		BorderStyle(lipgloss.NormalBorder()).
		BorderForeground(lipgloss.Color("240")).
		BorderBottom(true).
		Bold(false)
	s.Selected = s.Selected.
		Foreground(lipgloss.Color("229")).
		Background(lipgloss.Color("57")).
		Bold(false)
	res.SetStyles(s)
	/**************/

	m := model{
		fp:          fp,
		res:         res,
		isResHidden: true,
	}

	tea.NewProgram(&m).Run()
}

func convertToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func (m model) updateResFrom(path string) error {
	et, err := exiftool.NewExiftool()
	if err != nil {
		fmt.Printf("Error when intializing: %v\n", err)
		return err
	}
	defer et.Close()
	fileInfos := et.ExtractMetadata(path)

	rows := []table.Row{}

	for _, fileInfo := range fileInfos {
		if fileInfo.Err != nil {
			continue
		}
		// TODO COMPRESSION
		name := convertToString(fileInfo.Fields["FileName"])
		size := convertToString(fileInfo.Fields["ImageSize"])
		dpi := convertToString(fileInfo.Fields["XResolution"]) + "x" + convertToString(fileInfo.Fields["YResolution"])
		depth := convertToString(fileInfo.Fields["BitDepth"])
		rows = append(rows, table.Row{name, size, dpi, depth, "0"})
	}
	fmt.Println(rows)
	m.res.SetRows(rows)
	return nil
}
