package main

import (
	"fmt"
	"time"

	"github.com/conformal/gotk3/glib"
	"github.com/conformal/gotk3/gtk"
	"github.com/jrick/go-webkit2/wk2"
)

// Page is any widget that may be added to the page manager notebook.
type Page interface {
	gtk.IWidget
	fmt.Stringer
	TitleLabel() *gtk.Label

	// Show calls the Show method of the Page widget.  This is required
	// to be explicitly implemented since the gtk.IWidget interface does
	// not provide any exported way to access or call *gtk.Widget methods.
	Show()
}

// PageDescription describes the kind and parameters to create a new page.
type PageDescription interface {
	NewPage() Page
}

// HTMLPageDescription holds the parameters required for the page mangager
// to open and manage a new HTML page
type HTMLPageDescription struct {
	uri string
}

// NewPage creates a new HTML page from the description.  The returned page
// will be a *HTMLPage.
func (d *HTMLPageDescription) NewPage() Page {
	return NewHTMLPage(d.uri)
}

// DownloadsPageDescription describes a downloads page.
type DownloadsPageDescription struct{}

// NewPage creates a new downloads page from the description.  The returned
// page will be a *DownloadsPage.
func (DownloadsPageDescription) NewPage() Page {
	return NewDownloadsPage()
}

// SettingsPageDescription describes a settings page.
type SettingsPageDescription struct{}

// NewPage creates a new settings page from the description.  The returned
// page will be a *DownloadsPage.
func (SettingsPageDescription) NewPage() Page {
	return NewSettingsPage()
}

// PageManager maintains all open pages and displays them in tabs.
type PageManager struct {
	*gtk.Notebook
	htmls     map[uintptr]*HTMLPage
	downloads *DownloadsPage
	settings  *SettingsPage
	actions   *ActionMenu
}

// NewPageManager creates and initializes a page manager.
func NewPageManager(session []PageDescription) *PageManager {
	nb, _ := gtk.NotebookNew()
	nb.SetShowBorder(false)
	p := &PageManager{
		Notebook: nb,
		htmls:    map[uintptr]*HTMLPage{},
	}

	if session == nil {
		blank := &HTMLPageDescription{"about:blank"}
		session = []PageDescription{blank}
	}
	for _, page := range session {
		p.OpenPage(page)
	}

	actions := NewActionMenu()
	actions.newTab.Connect("activate", func() {
		blank := &HTMLPageDescription{"about:blank"}
		n := p.OpenPage(blank)
		p.FocusPageN(n)
	})
	actions.quit.Connect("activate", func() {
		gtk.MainQuit()
	})

	nb.SetActionWidget(actions, gtk.PACK_END)
	actions.Show()

	return p
}

// OpenPage opens the page described by desc.  If the description is for a
// downloads or settings page and one is already shown by the manager,
// it is switched to instead of opening a new page.  The notebook index of
// the shown page is returned.
func (p *PageManager) OpenPage(desc PageDescription) int {
	switch d := desc.(type) {
	case *HTMLPageDescription:
		page := d.NewPage().(*HTMLPage)
		p.htmls[page.Native()] = page
		return p.openNewPage(page)

	case DownloadsPageDescription:
		if p.downloads == nil {
			p.downloads = d.NewPage().(*DownloadsPage)
			return p.openNewPage(p.downloads)
		}
		index := p.PageNum(p.downloads)
		p.SetCurrentPage(index)
		return index

	case SettingsPageDescription:
		if p.settings == nil {
			p.settings = d.NewPage().(*SettingsPage)
			return p.openNewPage(p.settings)
		}
		index := p.PageNum(p.settings)
		p.SetCurrentPage(index)
		return index

	default:
		// Unknown page. Logically should never happen, but if it
		// ever does, handle it by showing it, but don't add it to
		// the page manager's internal structures.
		page := d.NewPage()
		return p.openNewPage(page)
	}
}

const notebookIconSize = 1

// openNewPage adds the page content and title label to the notebook.  A close
// tab button is added to the title label to create the notebook tab widget.
// When the close button is pressed, the page will be removed from the manager.
func (p *PageManager) openNewPage(page Page) int {
	// Create tab content using title label and connect necessary signals.
	tabContent, _ := gtk.GridNew()
	image, _ := gtk.ImageNewFromIconName("window-close", notebookIconSize)
	closeButton, _ := gtk.ButtonNew()
	closeButton.SetImage(image)
	tabContent.Add(closeButton)
	tabContent.Add(page.TitleLabel())

	closeButton.Connect("clicked", func() {
		pageNum := p.PageNum(page)
		p.RemovePage(pageNum)
		switch page := page.(type) {
		case *HTMLPage:
			delete(p.htmls, page.Native())
		case *DownloadsPage:
			p.downloads = nil
		case *SettingsPage:
			p.settings = nil
		}

		if p.GetNPages() == 0 {
			p.OpenPage(&HTMLPageDescription{"about:blank"})
		}
	})

	tabContent.ShowAll()
	page.Show()
	n := p.AppendPage(page, tabContent)
	p.SetTabReorderable(page, true)
	return n
}

// FocusPage switches notebook tabs to make page visible.
func (p *PageManager) FocusPage(page Page) {
	n := p.PageNum(page)
	p.SetCurrentPage(n)
}

// FocusPageN switches tabs to make the page at index n visible.
func (p *PageManager) FocusPageN(n int) {
	p.SetCurrentPage(n)
}

// tabContent is the widget added to the page manager's notebook tabs.
type tabContent struct {
	*gtk.Grid
	closeButton *gtk.Button
	title       *gtk.Label
}

// HTMLPage is a page for displaying and navigating web content.  A toolbar
// at the top acts as a navigation bar and contains an entry for displaying
// and modifying the page URI.  A WebKit WebView is placed below to render
// and display HTML content.
type HTMLPage struct {
	*gtk.Stack
	title      string
	uri        string
	titleLabel *gtk.Label
	navbar     *NavigationBar
	wv         *wk2.WebView
	crash      *gtk.Label
}

// NewHTMLPage creates a new HTML page and begins loading the URI specified
// by uri.  The URI `about:blank` may be used to load a blank page.
func NewHTMLPage(uri string) *HTMLPage {
	grid, _ := gtk.GridNew()
	grid.SetOrientation(gtk.ORIENTATION_VERTICAL)
	navbar := NewNavigationBar()
	navbar.SetHExpand(true)
	wv := wk2.NewWebView()
	wv.SetHExpand(true)
	wv.SetVExpand(true)
	title, _ := gtk.LabelNew("New Tab")
	crash, _ := gtk.LabelNew("WebKit crashed :'(")

	grid.SetCanFocus(false)
	navbar.SetCanFocus(false)
	title.SetCanFocus(false)

	grid.Add(navbar)
	navbar.Show()
	grid.Add(wv)
	grid.Show()

	// The focus chain is explicitly set to remove the navbar from the
	// tabbing order.  This is a workaround for a quirk with the page
	// manager's notebook always moving focus to the first focus chain
	// element when switching notebook pages, even if the desired widget
	// is explicitly focused from the "switch-page" signal.
	grid.SetFocusChain([]gtk.IWidget{wv})

	stack, _ := gtk.StackNew()
	stack.SetCanFocus(false)
	stack.AddNamed(grid, "webview")
	stack.AddNamed(crash, "crash")
	stack.SetVisibleChild(crash)

	page := &HTMLPage{stack, "New Tab", uri, title, navbar, wv, crash}

	page.connectNavbarSignals()
	page.connectWebViewSignals()

	// XXX: Hacks! work around for webkit race
	go func() {
		time.Sleep(time.Second)
		glib.IdleAdd(func() {
			stack.SetVisibleChild(grid)
			wv.Show()
			page.LoadURI(uri)
		})
	}()

	return page
}

func (p *HTMLPage) connectNavbarSignals() {
	p.navbar.uriEntry.Connect("activate", func(e *gtk.Entry) {
		fmt.Println("ok...")
		uri, _ := e.GetText()
		p.LoadURI(uri)
		p.wv.GrabFocus()
	})

	p.navbar.uriEntry.Connect("notify::is-focus", func(e *gtk.Entry) {
		if !e.IsFocus() {
			e.SelectRegion(0, 0)
		}
	})

	p.navbar.uriEntry.Connect("button-release-event", func(e *gtk.Entry) {
		e.SelectRegion(0, -1)
		//e.GrabFocus()
	})
}

func (p *HTMLPage) connectWebViewSignals() {
	p.wv.Connect("load-changed", func(wv *wk2.WebView, e wk2.LoadEvent) {
		switch e {
		case wk2.LoadStarted:
		case wk2.LoadRedirected:
		case wk2.LoadCommitted:
		case wk2.LoadFinished:
			p.navbar.uriEntry.SetProgressFraction(0)
		}
	})

	p.wv.Connect("web-process-crashed", func() {
		p.crash.Show()
		p.SetVisibleChild(p.crash)
	})

	p.wv.Connect("notify::estimated-load-progress", func() {
		progress := p.wv.EstimatedLoadProgress()
		p.navbar.uriEntry.SetProgressFraction(progress)
	})

	p.wv.Connect("notify::uri", func() {
		p.setURI(p.wv.URI())
	})

	p.wv.Connect("notify::title", func() {
		p.setTitle(p.wv.Title())
	})
}

// LoadURI begins loading the page's WebView with the URI described by uri.
func (p *HTMLPage) LoadURI(uri string) {
	p.wv.LoadURI(uri)
}

// Show calls the Show method of the page's Stack.
func (p *HTMLPage) Show() {
	p.Stack.Show()
}

// TitleLabel returns the current page title in a label.  This is intended
// to be used when creating the page manager's notebook tab content.
func (p *HTMLPage) TitleLabel() *gtk.Label {
	return p.titleLabel
}

// String returns the title of the WebView.
func (p *HTMLPage) String() string {
	return p.title
}

// setTitle sets the internal title and title label for a new WebView title.
func (p *HTMLPage) setTitle(title string) {
	p.title = title
	p.titleLabel.SetLabel(title)
}

// setURI sets the internal URI of the html page and modifies the URI entry
// in the navigation bar if the entry is not currently being modified.
func (p *HTMLPage) setURI(uri string) {
	p.uri = uri
	if e := p.navbar.uriEntry; !e.HasFocus() {
		e.SetText(uri)
	}
}

// NavigationBar is a toolbar for HTML page navigation.  It contains a URI
// entry to show and modify the currently-shown page in a WebView.
type NavigationBar struct {
	*gtk.Toolbar
	uriEntry *gtk.Entry
}

// NewNavigationBar creates a new navigation bar for a HTML page.
func NewNavigationBar() *NavigationBar {
	tb, _ := gtk.ToolbarNew()
	uriEntry, _ := gtk.EntryNew()
	uriEntry.SetInputPurpose(gtk.INPUT_PURPOSE_URL)

	tool, _ := gtk.ToolItemNew()
	tool.Add(uriEntry)
	tool.SetExpand(true)
	uriEntry.Show()

	tb.Add(tool)
	tool.Show()

	return &NavigationBar{tb, uriEntry}
}

type DownloadsPage struct {
	*gtk.Widget // TODO
}

func NewDownloadsPage() *DownloadsPage {
	return nil // TODO
}

func (p *DownloadsPage) Show() {
	p.Widget.Show()
}

func (*DownloadsPage) String() string {
	return "Downloads"
}

func (*DownloadsPage) TitleLabel() *gtk.Label {
	return nil // TODO
}

type SettingsPage struct {
	*gtk.Widget // TODO
}

func NewSettingsPage() *SettingsPage {
	return nil // TODO
}

func (p *SettingsPage) Show() {
	p.Widget.Show()
}

func (*SettingsPage) String() string {
	return "Settings"
}

func (*SettingsPage) TitleLabel() *gtk.Label {
	return nil // TODO
}

// ActionMenu is a button with a dropdown menu for page manager and application
// actions.
type ActionMenu struct {
	*gtk.MenuButton

	newTab *gtk.MenuItem

	downloads *gtk.MenuItem
	favorites *gtk.MenuItem
	settings  *gtk.MenuItem

	restart *gtk.MenuItem
	quit    *gtk.MenuItem
}

// NewActionMenu creates a new action menu button.
func NewActionMenu() *ActionMenu {
	mb, _ := gtk.MenuButtonNew()
	menu, _ := gtk.MenuNew()
	menu.SetHAlign(gtk.ALIGN_END)

	newTab, _ := gtk.MenuItemNewWithLabel("New tab")
	menu.Append(newTab)
	newTab.Show()

	sep, _ := gtk.SeparatorMenuItemNew()
	menu.Append(sep)
	sep.Show()

	downloads, _ := gtk.MenuItemNewWithLabel("Downloads (TODO)")
	menu.Append(downloads)
	downloads.Show()

	favorites, _ := gtk.MenuItemNewWithLabel("Favorites (TODO)")
	menu.Append(favorites)
	favorites.Show()

	settings, _ := gtk.MenuItemNewWithLabel("Settings (TODO)")
	menu.Append(settings)
	settings.Show()

	sep, _ = gtk.SeparatorMenuItemNew()
	menu.Append(sep)
	sep.Show()

	restart, _ := gtk.MenuItemNewWithLabel("Restart (TODO)")
	menu.Append(restart)
	restart.Show()

	quit, _ := gtk.MenuItemNewWithLabel("Quit")
	menu.Append(quit)
	quit.Show()

	mb.SetPopup(menu)
	menu.Show()

	return &ActionMenu{
		MenuButton: mb,
		newTab:     newTab,
		downloads:  downloads,
		favorites:  favorites,
		settings:   settings,
		restart:    restart,
		quit:       quit,
	}
}
