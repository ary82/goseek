package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ary82/goseek/internal/chunk"
	"github.com/ary82/goseek/internal/constants"
	"github.com/ary82/goseek/internal/llm"
	"github.com/ary82/goseek/internal/scrape"
	"github.com/ary82/goseek/internal/search"
	"github.com/ary82/goseek/internal/vectorstorage"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/google/uuid"
	_ "github.com/joho/godotenv/autoload"
	"github.com/pinecone-io/go-pinecone/v3/pinecone"
)

// Pipeline orchestrator
type GoSeekPipeline struct {
	search  search.SearchEngine
	scraper scrape.Scraper
	chunker chunk.Chunker
	vector  vectorstorage.VectorStore
	llm     llm.LLM
	mu      sync.RWMutex
	cache   map[string]string
}

func NewGoSeekPipeline() (*GoSeekPipeline, error) {
	se, err := search.NewGoogleSearchEngine(constants.SEARCH_API, os.Getenv("SEARCH_API_KEY"), os.Getenv("SEARCH_CX"))
	if err != nil {
		return nil, err
	}

	sc := scrape.NewWebScraper(&http.Client{}, constants.UA, 4)
	ch := chunk.NewTextChunker(512, 64, 0.1)

	db, err := vectorstorage.NewPineconeStorage(os.Getenv("PINECONE_API_KEY"), os.Getenv("PINECONE_HOST"))
	if err != nil {
		return nil, err
	}

	genllm, err := llm.NewGeminiLLM(context.Background(), os.Getenv("SEARCH_API_KEY"))
	if err != nil {
		return nil, err
	}

	return &GoSeekPipeline{
		search:  se,
		scraper: sc,
		chunker: ch,
		vector:  db,
		llm:     genllm,
		cache:   make(map[string]string),
	}, nil
}

func (p *GoSeekPipeline) ProcessQuery(ctx context.Context, query string) (string, error) {
	// Check cache first
	p.mu.RLock()
	if cached, exists := p.cache[query]; exists {
		p.mu.RUnlock()
		return cached, nil
	}
	p.mu.RUnlock()

	// Step 1: Search
	searchResults, err := p.search.Search(ctx, query, search.QueryParams{})
	if err != nil {
		return "", fmt.Errorf("search failed: %w", err)
	}

	if len(searchResults.Items) == 0 {
		return "No search results found for your query.", nil
	}

	// Step 2: Extract URLs and scrape
	var toBeScraped []string
	for _, v := range searchResults.Items {
		toBeScraped = append(toBeScraped, v.Link)
	}

	scrapedContent, err := p.scraper.Scrape(ctx, toBeScraped)
	if err != nil {
		return "", fmt.Errorf("scraping failed: %w", err)
	}

	if len(scrapedContent) == 0 {
		return "Could not scrape any content from the search results.", nil
	}

	// Step 3: Chunk the content

	var allChunks []chunk.Chunk
	for i, v := range scrapedContent {
		c, err := p.chunker.Chunk(ctx, i, v.Content)
		if err != nil {
			return "", err
		}
		allChunks = append(allChunks, c...)
	}

	// Step 4: Store in vector database

	var records []*pinecone.IntegratedRecord
	ns := uuid.NewString()
	num := 0
	for _, v := range allChunks {
		record := pinecone.IntegratedRecord{
			"id":   uuid.NewString(),
			"text": v.Content,
			"link": v.Link,
		}
		records = append(records, &record)
		num += 1
		if num == 50 {
			err = p.vector.UpsertRecords(ctx, records, ns)
			if err != nil {
				log.Printf("upsert failed")
				// return "", err
			}
			records = []*pinecone.IntegratedRecord{}
			num = 0
		}
	}
	// err = p.vector.UpsertRecords(ctx, records, ns)
	// if err != nil {
	// 	return "", err
	// }

	time.Sleep(3 * time.Second)

	// Step 5: Retrieve relevant chunks
	relevantRecords, err := p.vector.SearchTopK(ctx, query, 5, ns)
	if err != nil {
		log.Println(err)
		return "", fmt.Errorf("vector search failed: %w", err)
	}
	rr, ok := relevantRecords.(*pinecone.SearchRecordsResponse)
	if !ok {
		return "", fmt.Errorf("vector search result corrupted")
	}

	// Step 6: Generate response with LLM
	var ctxForLLM string
	for _, v := range rr.Result.Hits {
		str := fmt.Sprintf("[%s] %s\n\n", v.Fields["link"], v.Fields["text"])
		ctxForLLM += str
	}

	prompt := fmt.Sprintf(constants.PROMPT, query, ctxForLLM)
	response, err := p.llm.GenerateContent(ctx, prompt)
	if err != nil {
		return "", fmt.Errorf("LLM generation failed: %w", err)
	}

	// Cache the result
	p.mu.Lock()
	p.cache[query] = *response
	p.mu.Unlock()

	return *response, nil
}

// TUI Model
type model struct {
	pipeline   *GoSeekPipeline
	textarea   textarea.Model
	viewport   viewport.Model
	help       help.Model
	spinner    spinner.Model
	ready      bool
	processing bool
	// response   string
	query     string
	sessionID string
	width     int
	height    int
}

type processMsg struct {
	response string
	err      error
}

// Key bindings
type keyMap struct {
	Submit key.Binding
	Quit   key.Binding
	Help   key.Binding
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Submit, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Submit, k.Help},
		{k.Quit},
	}
}

var keys = keyMap{
	Submit: key.NewBinding(
		key.WithKeys("ctrl+s"),
		key.WithHelp("ctrl+s", "submit query"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+c", "q"),
		key.WithHelp("ctrl+c/q", "quit"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle help"),
	),
}

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	responseStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2).
			Margin(1, 0)

	inputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04B575")).
			Padding(0, 1)

	processingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFA500")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF6B6B")).
			Bold(true)
)

func initialModel(pipeline *GoSeekPipeline, sessionID string) model {
	ta := textarea.New()
	ta.Placeholder = "Ask me anything..."
	ta.Focus()
	ta.Prompt = "┃ "
	ta.CharLimit = 500
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)
	vp.SetContent("Welcome to SSH GoSeek! 🔍\n\nType your question and press Ctrl+S to search.")

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))

	return model{
		pipeline:  pipeline,
		textarea:  ta,
		viewport:  vp,
		help:      help.New(),
		spinner:   sp,
		sessionID: sessionID,
		ready:     true,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textarea.Blink, m.spinner.Tick)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		tiCmd tea.Cmd
		vpCmd tea.Cmd
		spCmd tea.Cmd
	)

	m.textarea, tiCmd = m.textarea.Update(msg)
	m.viewport, vpCmd = m.viewport.Update(msg)
	m.spinner, spCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := lipgloss.Height(m.headerView())
		footerHeight := lipgloss.Height(m.footerView())
		verticalMarginHeight := headerHeight + footerHeight

		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-verticalMarginHeight)
			m.viewport.YPosition = headerHeight
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - verticalMarginHeight
		}

		m.textarea.SetWidth(msg.Width - 4)

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Submit):
			if m.processing {
				return m, nil
			}
			query := strings.TrimSpace(m.textarea.Value())
			if query == "" {
				return m, nil
			}
			m.query = query
			m.processing = true
			m.textarea.Reset()
			return m, tea.Batch(
				m.processQuery(query),
				m.spinner.Tick,
			)
		}

	case processMsg:
		m.processing = false
		if msg.err != nil {
			content := errorStyle.Render("Error: "+msg.err.Error()) + "\n\n" + m.viewport.View()
			m.viewport.SetContent(content)
		} else {
			styledResponse := responseStyle.Width(m.viewport.Width - 4).Render(msg.response)
			content := fmt.Sprintf("🔍 Query: %s\n\n%s\n\n%s",
				m.query,
				styledResponse,
				m.viewport.View(),
			)
			m.viewport.SetContent(content)
			m.viewport.GotoTop()
		}
	}

	return m, tea.Batch(tiCmd, vpCmd, spCmd)
}

func (m model) processQuery(query string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
		defer cancel()

		response, err := m.pipeline.ProcessQuery(ctx, query)
		return processMsg{response: response, err: err}
	}
}

func (m model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	return fmt.Sprintf("%s\n%s\n%s",
		m.headerView(),
		m.viewport.View(),
		m.footerView(),
	)
}

func (m model) headerView() string {
	title := titleStyle.Render("SSH GoSeek")
	status := ""
	if m.processing {
		status = processingStyle.Render(fmt.Sprintf(" %s Processing...", m.spinner.View()))
	}
	line := strings.Repeat("─", max(0, m.width-lipgloss.Width(title+status)))
	return lipgloss.JoinHorizontal(lipgloss.Center, title, line, status)
}

func (m model) footerView() string {
	info := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render(
		fmt.Sprintf("Session: %s", m.sessionID),
	)

	inputArea := inputStyle.Render(m.textarea.View())

	help := m.help.ShortHelpView(keys.ShortHelp())

	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(info)-lipgloss.Width(help)))
	topLine := lipgloss.JoinHorizontal(lipgloss.Center, info, gap, help)

	return lipgloss.JoinVertical(lipgloss.Left, topLine, inputArea)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// SSH Server setup
func main() {
	// Initialize pipeline with mock implementations
	// Replace these with your actual implementations
	pipeline, err := NewGoSeekPipeline()
	if err != nil {
		log.Fatal(err)
	}

	host := "localhost"
	port := "23234"

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/term_info_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
				sessionID := fmt.Sprintf("%s-%d", s.RemoteAddr().String(), time.Now().Unix())
				m := initialModel(pipeline, sessionID)
				return m, []tea.ProgramOption{tea.WithAltScreen()}
			}),
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Fatalln(err)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Printf("Starting SSH server on %s", net.JoinHostPort(host, port))
	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Fatalln(err)
		}
	}()

	<-done
	log.Println("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		log.Fatalln(err)
	}
}
