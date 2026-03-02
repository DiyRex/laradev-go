package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/DiyRex/laradev-go/config"
	"github.com/DiyRex/laradev-go/process"
	"github.com/DiyRex/laradev-go/tui/components"
	"github.com/DiyRex/laradev-go/tui/pages"
	"github.com/DiyRex/laradev-go/tui/shared"
)

type App struct {
	Config  *config.Config
	Manager *process.Manager

	activePage shared.PageID
	pageStack  []shared.PageID

	mainMenu *pages.MainMenuPage
	services *pages.ServicesPage
	database *pages.DatabasePage
	develop  *pages.DevelopPage
	makePage *pages.MakePage
	logs     *pages.LogsPage
	cache    *pages.CachePage
	cfgPage  *pages.ConfigPage

	width    int
	height   int
	quitting bool
}

func NewApp(cfg *config.Config, mgr *process.Manager) App {
	return App{
		Config:     cfg,
		Manager:    mgr,
		activePage: shared.PageMainMenu,
		mainMenu:   pages.NewMainMenuPage(cfg, mgr),
		services:   pages.NewServicesPage(cfg, mgr),
		database:   pages.NewDatabasePage(cfg, mgr),
		develop:    pages.NewDevelopPage(cfg, mgr),
		makePage:   pages.NewMakePage(cfg, mgr),
		logs:       pages.NewLogsPage(cfg, mgr),
		cache:      pages.NewCachePage(cfg, mgr),
		cfgPage:    pages.NewConfigPage(cfg, mgr),
	}
}

func (a App) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return shared.TickMsg(t)
	})
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.propagateSize()
		return a, nil

	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			a.quitting = true
			return a, tea.Quit
		}

	case shared.NavigateMsg:
		return a.navigate(msg.Page)

	case shared.NavigateBackMsg:
		return a.goBack()

	case shared.TickMsg:
		return a, tickCmd()

	case shared.ExecDoneMsg:
		return a, nil
	}

	var cmd tea.Cmd
	switch a.activePage {
	case shared.PageMainMenu:
		cmd = a.mainMenu.Update(msg)
	case shared.PageServices:
		cmd = a.services.Update(msg)
	case shared.PageDatabase:
		cmd = a.database.Update(msg)
	case shared.PageDevelop:
		cmd = a.develop.Update(msg)
	case shared.PageMake:
		cmd = a.makePage.Update(msg)
	case shared.PageLogs:
		cmd = a.logs.Update(msg)
	case shared.PageCache:
		cmd = a.cache.Update(msg)
	case shared.PageConfig:
		cmd = a.cfgPage.Update(msg)
	}

	return a, cmd
}

func (a *App) navigate(page shared.PageID) (App, tea.Cmd) {
	a.pageStack = append(a.pageStack, a.activePage)
	a.activePage = page

	switch page {
	case shared.PageServices:
		a.services.Enter()
	case shared.PageDatabase:
		a.database.Enter()
	case shared.PageDevelop:
		a.develop.Enter()
	case shared.PageMake:
		a.makePage.Enter()
	case shared.PageLogs:
		a.logs.Enter()
	case shared.PageCache:
		a.cache.Enter()
	case shared.PageConfig:
		a.cfgPage.Enter()
	}

	a.propagateSize()
	return *a, nil
}

func (a *App) goBack() (App, tea.Cmd) {
	if len(a.pageStack) > 0 {
		a.activePage = a.pageStack[len(a.pageStack)-1]
		a.pageStack = a.pageStack[:len(a.pageStack)-1]
	} else {
		a.activePage = shared.PageMainMenu
	}
	return *a, nil
}

func (a *App) propagateSize() {
	pageH := a.height - 9
	if pageH < 5 {
		pageH = 5
	}
	a.mainMenu.SetSize(a.width, pageH)
	a.services.SetSize(a.width, pageH)
	a.database.SetSize(a.width, pageH)
	a.develop.SetSize(a.width, pageH)
	a.makePage.SetSize(a.width, pageH)
	a.logs.SetSize(a.width, pageH)
	a.cache.SetSize(a.width, pageH)
	a.cfgPage.SetSize(a.width, pageH)
}

func (a App) View() string {
	if a.quitting {
		return ""
	}

	pageName := "Main Menu"
	switch a.activePage {
	case shared.PageServices:
		pageName = "Services"
	case shared.PageDatabase:
		pageName = "Database"
	case shared.PageDevelop:
		pageName = "Development"
	case shared.PageMake:
		pageName = "Make (Generate)"
	case shared.PageLogs:
		pageName = "Logs"
	case shared.PageCache:
		pageName = "Cache & Optimize"
	case shared.PageConfig:
		pageName = "Config"
	}

	w := a.width
	if w < 40 {
		w = 40
	}

	// Title bar: " LaraDev  >  Page                 DiyRex "
	// Build the inner content, then let TitleBarStyle.Width pad it
	left := " LaraDev " + shared.BrandStyle.Render(" > ") + " " + pageName
	right := shared.BrandStyle.Render("By DiyRex") + " "
	innerW := w - 2 // account for Padding(0,1) = 1 char each side
	gap := innerW - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 2 {
		gap = 2
	}
	titleContent := left + strings.Repeat(" ", gap) + right
	titleBar := shared.TitleBarStyle.Width(w).Render(titleContent)

	statusBar := components.RenderStatusBar(a.Manager, a.Config.PHPPort, a.Config.VitePort)
	infoBox := components.RenderInfoBox(a.Config, a.width)

	var pageView string
	switch a.activePage {
	case shared.PageMainMenu:
		pageView = a.mainMenu.View()
	case shared.PageServices:
		pageView = a.services.View()
	case shared.PageDatabase:
		pageView = a.database.View()
	case shared.PageDevelop:
		pageView = a.develop.View()
	case shared.PageMake:
		pageView = a.makePage.View()
	case shared.PageLogs:
		pageView = a.logs.View()
	case shared.PageCache:
		pageView = a.cache.View()
	case shared.PageConfig:
		pageView = a.cfgPage.View()
	}

	// Separator line
	sepW := w - 4
	if sepW < 20 {
		sepW = 20
	}
	separator := lipgloss.NewStyle().Foreground(shared.CeriseDk).PaddingLeft(2).
		Render(strings.Repeat("─", sepW))

	// Footer: keybinds on left, credit on right
	footerKeys := shared.FooterKeyStyle.Render("↑↓") + shared.FooterStyle.Render(" nav  ") +
		shared.FooterKeyStyle.Render("enter") + shared.FooterStyle.Render(" select  ") +
		shared.FooterKeyStyle.Render("esc") + shared.FooterStyle.Render(" back  ") +
		shared.FooterKeyStyle.Render("q") + shared.FooterStyle.Render(" quit")

	footerLine := fmt.Sprintf("  %s", footerKeys)

	return lipgloss.JoinVertical(lipgloss.Left,
		titleBar,
		statusBar,
		infoBox,
		pageView,
		separator,
		footerLine,
	)
}
