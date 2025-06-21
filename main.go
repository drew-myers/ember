package main

import (
	"fmt"
	"log"
	"os"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screenState int

const (
	inputScreen screenState = iota
	resultsScreen
	embeddingsScreen
	loadingScreen
	quitConfirmationScreen
)

var (
	staticTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9567E3")).
			Bold(true).
			Width(80).
			Align(lipgloss.Left)

	userInputStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#C967E3")).
			Bold(true).
			Width(80).
			Align(lipgloss.Left)
)

type CustomEmbedding struct {
	Text      string
	Embedding []float64
}

// Messages for async operations
type embeddingCompleteMsg struct {
	embedding []float64
	text      string
	err       error
}

type customEmbeddingsCompleteMsg struct {
	embeddings []CustomEmbedding
	err        error
}

type model struct {
	textarea          textarea.Model
	embeddingsService *EmbeddingsService
	similarities      []SimilarityResult
	lastInput         string
	currentScreen     screenState
	progressBars      []progress.Model

	// Embeddings selection screen
	embeddingTexts   []textarea.Model
	selectedTextArea int
	customEmbeddings []CustomEmbedding

	// Loading screen
	spinner        spinner.Model
	loadingMessage string
}

func initialModel() model {
	ta := textarea.New()
	ta.Placeholder = "Enter text to embed..."
	ta.Focus()
	ta.SetWidth(80)
	ta.SetHeight(10)
	ta.ShowLineNumbers = false

	// Initialize embedding text areas with 2 default areas
	embeddingTexts := make([]textarea.Model, 2)
	for i := 0; i < 2; i++ {
		ta := textarea.New()
		ta.Placeholder = fmt.Sprintf("Enter comparison text %d...", i+1)
		ta.SetWidth(75)
		ta.SetHeight(3)
		ta.ShowLineNumbers = false
		embeddingTexts[i] = ta
	}

	// Initialize spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#C967E3"))

	// Initialize with static examples as default
	customEmbeddings := []CustomEmbedding{
		{Text: "I hate the state of california.", Embedding: staticExamples[0].Embedding},
		{Text: "Washington is a really great place.", Embedding: staticExamples[1].Embedding},
	}

	return model{
		textarea:          ta,
		embeddingsService: NewEmbeddingsService(),
		currentScreen:     inputScreen,
		embeddingTexts:    embeddingTexts,
		selectedTextArea:  0,
		customEmbeddings:  customEmbeddings,
		spinner:           s,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case embeddingCompleteMsg:
		if msg.err != nil {
			// Handle error - return to input screen
			m.currentScreen = inputScreen
			return m, nil
		}

		// Success - show results
		m.similarities = m.compareWithCustomEmbeddings(msg.embedding)
		m.lastInput = msg.text
		m.setupProgressBars()
		m.currentScreen = resultsScreen
		return m, nil

	case customEmbeddingsCompleteMsg:
		if msg.err != nil {
			// Handle error - return to input screen
			m.currentScreen = inputScreen
			return m, nil
		}

		// Success - update embeddings and return to input
		m.customEmbeddings = msg.embeddings
		m.currentScreen = inputScreen
		return m, nil

	case spinner.TickMsg:
		if m.currentScreen == loadingScreen {
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			if m.currentScreen == embeddingsScreen {
				m.currentScreen = inputScreen
				return m, nil
			}
			if m.currentScreen == quitConfirmationScreen {
				// Cancel quit confirmation - return to previous screen
				m.currentScreen = inputScreen
				return m, nil
			}
			// Show quit confirmation
			m.currentScreen = quitConfirmationScreen
			return m, nil
		case "enter":
			if m.currentScreen == resultsScreen {
				m.currentScreen = inputScreen
				return m, nil
			}
		case "y", "Y":
			if m.currentScreen == quitConfirmationScreen {
				return m, tea.Quit
			}
		case "n", "N":
			if m.currentScreen == quitConfirmationScreen {
				m.currentScreen = inputScreen
				return m, nil
			}
		case "tab":
			if m.currentScreen == inputScreen {
				m.currentScreen = embeddingsScreen
				if len(m.embeddingTexts) > 0 {
					m.embeddingTexts[0].Focus()
					for i := 1; i < len(m.embeddingTexts); i++ {
						m.embeddingTexts[i].Blur()
					}
				}
				return m, nil
			} else if m.currentScreen == embeddingsScreen {
				if len(m.embeddingTexts) > 0 {
					m.embeddingTexts[m.selectedTextArea].Blur()
					m.selectedTextArea = (m.selectedTextArea + 1) % len(m.embeddingTexts)
					m.embeddingTexts[m.selectedTextArea].Focus()
				}
				return m, nil
			}
		case "ctrl+n":
			if m.currentScreen == embeddingsScreen && len(m.embeddingTexts) < 10 {
				// Add new text area
				ta := textarea.New()
				ta.Placeholder = fmt.Sprintf("Enter comparison text %d...", len(m.embeddingTexts)+1)
				ta.SetWidth(75)
				ta.SetHeight(3)
				ta.ShowLineNumbers = false
				m.embeddingTexts = append(m.embeddingTexts, ta)
				return m, nil
			}
		case "ctrl+m":
			if m.currentScreen == embeddingsScreen && len(m.embeddingTexts) > 1 {
				// Remove current text area
				if m.selectedTextArea >= len(m.embeddingTexts) {
					m.selectedTextArea = len(m.embeddingTexts) - 1
				}
				// Remove the currently selected text area
				m.embeddingTexts = append(m.embeddingTexts[:m.selectedTextArea], m.embeddingTexts[m.selectedTextArea+1:]...)
				// Adjust selected index if needed
				if m.selectedTextArea >= len(m.embeddingTexts) {
					m.selectedTextArea = len(m.embeddingTexts) - 1
				}
				// Focus the new current text area
				if len(m.embeddingTexts) > 0 {
					for i := range m.embeddingTexts {
						m.embeddingTexts[i].Blur()
					}
					m.embeddingTexts[m.selectedTextArea].Focus()
				}
				return m, nil
			}
		case "alt+enter":
			if m.currentScreen == inputScreen {
				text := m.textarea.Value()
				if text != "" {
					m.loadingMessage = "Generating embeddings for comparison..."
					m.currentScreen = loadingScreen
					m.textarea.SetValue("")
					return m, tea.Batch(m.spinner.Tick, m.generateSingleEmbedding(text))
				}
				return m, nil
			} else if m.currentScreen == embeddingsScreen {
				// Check if all text areas have content
				texts := make([]string, 0, len(m.embeddingTexts))
				for _, ta := range m.embeddingTexts {
					text := ta.Value()
					if text != "" {
						texts = append(texts, text)
					}
				}
				if len(texts) > 0 {
					m.loadingMessage = "Generating custom embeddings..."
					m.currentScreen = loadingScreen
					return m, tea.Batch(m.spinner.Tick, m.generateAllEmbeddings(texts))
				}
				return m, nil
			}
		}
	}

	if m.currentScreen == inputScreen {
		m.textarea, cmd = m.textarea.Update(msg)
	} else if m.currentScreen == embeddingsScreen && len(m.embeddingTexts) > 0 {
		m.embeddingTexts[m.selectedTextArea], cmd = m.embeddingTexts[m.selectedTextArea].Update(msg)
	}
	return m, cmd
}

func (m *model) setupProgressBars() {
	m.progressBars = make([]progress.Model, len(m.similarities))
	for i := range m.similarities {
		prog := progress.New(progress.WithDefaultGradient())
		prog.Width = 60
		m.progressBars[i] = prog
	}
}

func (m model) View() string {
	switch m.currentScreen {
	case resultsScreen:
		return m.renderResultsScreen()
	case embeddingsScreen:
		return m.renderEmbeddingsScreen()
	case loadingScreen:
		return m.renderLoadingScreen()
	case quitConfirmationScreen:
		return m.renderQuitConfirmationScreen()
	default:
		return m.renderInputScreen()
	}
}

func (m model) renderInputScreen() string {
	s := "\033[2J\033[H" // Clear screen and move cursor to top

	// Add a fun header
	s += "╭─────────────────────────────────────────────────────────────────────────────╮\n"
	s += "│                               🟣 EMBER 🟣                                   │\n"
	s += "╰─────────────────────────────────────────────────────────────────────────────╯\n\n"

	// Style the label
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9567E3")).
		Bold(true)

	s += labelStyle.Render("✨ Enter your text below:") + "\n\n"
	s += m.textarea.View() + "\n\n"

	// Add styled instructions
	instructStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true)

	s += instructStyle.Render("💡 Alt+Enter to compare • Tab to configure comparisons • Ctrl+C to quit") + "\n"

	// Add padding to ensure clean display
	for i := 0; i < 10; i++ {
		s += "\n"
	}

	return s
}

func (m model) renderResultsScreen() string {
	// Clear screen by adding enough content to fill the terminal
	s := "\033[2J\033[H" // ANSI escape codes to clear screen and move cursor to top

	s += fmt.Sprintf("Similarity Results for:\n%s\n\n", userInputStyle.Render(m.lastInput))
	s += "╭─────────────────────────────────────────────────────────────────────────────╮\n"
	s += "│                            ✨ COMPARISON RESULTS ✨                         │\n"
	s += "╰─────────────────────────────────────────────────────────────────────────────╯\n\n"

	for i, result := range m.similarities {
		s += staticTextStyle.Render(result.Text) + "\n"
		s += fmt.Sprintf("Similarity: %.3f\n", result.Similarity)
		if i < len(m.progressBars) {
			s += m.progressBars[i].ViewAs(result.Similarity) + "\n\n"
		}
	}

	s += "Press Enter to return to input screen, Ctrl+C or Esc to quit."

	// Add padding to ensure we cover the entire screen
	for i := 0; i < 20; i++ {
		s += "\n"
	}

	return s
}

func (m model) renderEmbeddingsScreen() string {
	s := "\033[2J\033[H" // Clear screen and move cursor to top

	// Add header
	s += "╭─────────────────────────────────────────────────────────────────────────────╮\n"
	s += "│                        🎯 CONFIGURE COMPARISONS 🎯                          │\n"
	s += fmt.Sprintf("│                     Define your comparison texts (%d/10)                    │\n", len(m.embeddingTexts))
	s += "╰─────────────────────────────────────────────────────────────────────────────╯\n\n"

	// Style for labels
	labelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9567E3")).
		Bold(true)

	activeStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#C967E3"))

	inactiveStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#666666"))

	// Render all text areas
	for i, ta := range m.embeddingTexts {
		s += labelStyle.Render(fmt.Sprintf("📝 Comparison text %d:", i+1)) + "\n"
		if m.selectedTextArea == i {
			s += activeStyle.Render(ta.View()) + "\n\n"
		} else {
			s += inactiveStyle.Render(ta.View()) + "\n\n"
		}
	}

	// Instructions
	instructStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#666666")).
		Italic(true)

	s += instructStyle.Render("💡 Tab to switch • Ctrl+N to add • Ctrl+M to remove • Alt+Enter to generate • Esc to return") + "\n"

	// Add padding
	for i := 0; i < 2; i++ {
		s += "\n"
	}

	return s
}

func (m model) renderLoadingScreen() string {
	s := "\033[2J\033[H" // Clear screen and move cursor to top

	// Add header
	s += "╭─────────────────────────────────────────────────────────────────────────────╮\n"
	s += "│                           🤖 PROCESSING 🤖                                  │\n"
	s += "│                      Generating embeddings...                               │\n"
	s += "╰─────────────────────────────────────────────────────────────────────────────╯\n\n"

	// Center the spinner and message
	s += "\n\n\n\n\n\n"
	s += fmt.Sprintf("                              %s %s\n", m.spinner.View(), m.loadingMessage)

	// Add padding
	for i := 0; i < 15; i++ {
		s += "\n"
	}

	return s
}

func (m model) renderQuitConfirmationScreen() string {
	s := "\033[2J\033[H" // Clear screen and move cursor to top

	// Add header
	s += "╭─────────────────────────────────────────────────────────────────────────────╮\n"
	s += "│                               ⚠️  WARNING ⚠️                                │\n"
	s += "│                            Are you sure you want to quit?                   │\n"
	s += "╰─────────────────────────────────────────────────────────────────────────────╯\n\n"

	// Center the content
	s += "\n\n\n\n\n\n"

	// Warning message
	warningStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#ff6b6b")).
		Bold(true).
		Align(lipgloss.Center).
		Width(80)

	s += warningStyle.Render("You may lose any unsaved text input!") + "\n\n\n"

	// Instructions
	instructStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9567E3")).
		Bold(true).
		Align(lipgloss.Center).
		Width(80)

	s += instructStyle.Render("Press Y to quit • Press N to cancel • Press Esc to cancel") + "\n"

	// Add padding
	for i := 0; i < 10; i++ {
		s += "\n"
	}

	return s
}

func (m model) compareWithCustomEmbeddings(inputEmbedding []float64) []SimilarityResult {
	results := make([]SimilarityResult, len(m.customEmbeddings))

	for i, example := range m.customEmbeddings {
		similarity := cosineSimilarity(inputEmbedding, example.Embedding)
		results[i] = SimilarityResult{
			Text:       example.Text,
			Similarity: similarity,
		}
	}

	return results
}

func (m model) generateSingleEmbedding(text string) tea.Cmd {
	return func() tea.Msg {
		embedding, err := m.embeddingsService.GenerateEmbedding(text)
		return embeddingCompleteMsg{
			embedding: embedding,
			text:      text,
			err:       err,
		}
	}
}

func (m model) generateAllEmbeddings(texts []string) tea.Cmd {
	return func() tea.Msg {
		embeddings := make([]CustomEmbedding, 0, len(texts))

		for _, text := range texts {
			embedding, err := m.embeddingsService.GenerateEmbedding(text)
			if err != nil {
				return customEmbeddingsCompleteMsg{err: err}
			}
			embeddings = append(embeddings, CustomEmbedding{
				Text:      text,
				Embedding: embedding,
			})
		}

		return customEmbeddingsCompleteMsg{
			embeddings: embeddings,
			err:        nil,
		}
	}
}

func checkAPIKey() {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		displayAPIKeyError()
		os.Exit(1)
	}
}

func displayAPIKeyError() {
	fmt.Println("❌ Error: OPENAI_API_KEY environment variable not set.")
}

func main() {
	// Check for API key before starting the application
	checkAPIKey()
	
	p := tea.NewProgram(initialModel())
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
