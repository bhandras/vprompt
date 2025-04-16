// Package vprompt provides a reusable, configurable multi-line prompt
// component with history and autocompletion for bubbletea applications.
package vprompt

import (
	"fmt"
	"strings"
	"unicode"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

// Suggestion represents a single autocomplete suggestion. It holds the word to
// be inserted and an optional description.
type Suggestion struct {
	// Text is the suggestion text itself (e.g., "SELECT").
	Text string
	// Description provides optional context for the suggestion (e.g.,
	// "Select data from a table").
	Description string
}

// defaultPromptStyle defines the style for the prompt symbols (e.g., "sql> ").
// Muted purple.
var defaultPromptStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))

// defaultCursorStyle defines the style for the cursor block. Dim background.
var defaultCursorStyle = lipgloss.NewStyle().Background(lipgloss.Color("240"))

// defaultPopupBoxStyle defines the style for the suggestion popup container.
// Internal padding, dark grey background, light foreground text.
var defaultPopupBoxStyle = lipgloss.NewStyle().
	Padding(0, 1).
	Background(lipgloss.Color("237")).
	Foreground(lipgloss.Color("252"))

// defaultSelectedItemStyle defines the style for the highlighted suggestion.
// Greenish background, white foreground text.
var defaultSelectedItemStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("60")).
	Foreground(lipgloss.Color("255"))

// defaultUnselectedItemStyle defines the style for non-highlighted suggestions.
// Inherits from PopupBox or terminal default.
var defaultUnselectedItemStyle = lipgloss.NewStyle()

// defaultDescriptionStyle defines the style for the description part of a
// suggestion. Dimmer grey text.
var defaultDescriptionStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("242"))

// PromptStyles holds the lipgloss styles used for rendering the prompt UI
// components.
type PromptStyles struct {
	// Prompt is the style for the prompt string (e.g., "sql> ").
	Prompt lipgloss.Style
	// Cursor is the style for the text cursor.
	Cursor lipgloss.Style
	// PopupBox is the style for the suggestion popup container box.
	PopupBox lipgloss.Style
	// SelectedItem is the style for the currently selected suggestion line.
	SelectedItem lipgloss.Style
	// UnselectedItem is the style for non-selected suggestion lines.
	UnselectedItem lipgloss.Style
	// Description is the style for the description part of suggestions.
	Description lipgloss.Style
}

// DefaultPromptStyles returns a default set of PromptStyles, initializing all
// fields.
func DefaultPromptStyles() PromptStyles {
	return PromptStyles{
		Prompt:         defaultPromptStyle,
		Cursor:         defaultCursorStyle,
		PopupBox:       defaultPopupBoxStyle,
		SelectedItem:   defaultSelectedItemStyle,
		UnselectedItem: defaultUnselectedItemStyle,
		Description:    defaultDescriptionStyle,
	}
}

// AutoCompleteFunc defines the signature for a user-provided function that
// returns suggestions. It receives the full text before the cursor and the
// current word fragment being typed, allowing for context-aware autocompletion
// logic.
type AutoCompleteFunc func(textBeforeCursor string,
	wordFragment string) []Suggestion

// ExecuteFunc defines the signature for a user-provided function that executes
// the final input. It receives the complete, joined input string and should
// return a string representing the output or result of the execution to be
// displayed to the user.
type ExecuteFunc func(input string) string

// IsCompleteFunc defines the signature for a user-provided function that
// determines if the current multi-line input is complete and ready for
// execution.
type IsCompleteFunc func(input string) bool

// IsWordCharFunc defines the signature for a user-provided function that
// determines if a given rune should be considered part of a "word" for
// autocompletion purposes.
type IsWordCharFunc func(r rune) bool

// PromptConfig holds all the customizable settings for the PromptModel.
type PromptConfig struct {
	// PromptPrimary is the prompt string for the first line.
	PromptPrimary string
	// PromptSecondary is the prompt string for subsequent lines.
	PromptSecondary string
	// AutoCompleteFn is the user function to get autocomplete suggestions.
	AutoCompleteFn AutoCompleteFunc
	// ExecuteFn is the user function to execute the completed input.
	ExecuteFn ExecuteFunc
	// IsCompleteFn is the user function to check if input is complete.
	IsCompleteFn IsCompleteFunc
	// IsWordCharFn is the user function to define word boundaries for
	// autocompletion.
	IsWordCharFn IsWordCharFunc
	// Styles contains the lipgloss styles for rendering various UI parts.
	Styles PromptStyles
	// ShowDescription controls description visibility in suggestions.
	ShowDescription bool
	// PopupMaxHeight limits the number of suggestions shown before
	// scrolling.
	PopupMaxHeight int
}

// DefaultIsComplete provides a default implementation for IsCompleteFunc. It
// considers input complete if it ends with a semicolon after trimming
// whitespace.
func DefaultIsComplete(input string) bool {
	trimmedInput := strings.TrimSpace(input)
	return strings.HasSuffix(trimmedInput, ";")
}

// DefaultIsWordChar provides a default implementation for IsWordCharFunc. It
// considers letters, digits, underscore (_), and period (.) as word characters.
func DefaultIsWordChar(r rune) bool {
	isLetter := unicode.IsLetter(r)
	isDigit := unicode.IsDigit(r)
	isUnderscore := r == '_'
	isPeriod := r == '.'
	return isLetter || isDigit || isUnderscore || isPeriod
}

// NewPromptConfig creates a new PromptConfig with sensible defaults. The user
// must provide the primary/secondary prompts, the autocomplete function, and
// the execution function. Other settings use default implementations.
func NewPromptConfig(primary, secondary string, acFn AutoCompleteFunc,
	execFn ExecuteFunc) PromptConfig {

	return PromptConfig{
		PromptPrimary:   primary,
		PromptSecondary: secondary,
		AutoCompleteFn:  acFn,
		ExecuteFn:       execFn,
		// Use default semicolon check
		IsCompleteFn: DefaultIsComplete,
		// Use default word character definition
		IsWordCharFn: DefaultIsWordChar,
		// Use default styling
		Styles: DefaultPromptStyles(),
		// Hide descriptions by default
		ShowDescription: false,
		// Show max 6 suggestions by default
		PopupMaxHeight: 6,
	}
}

// PromptModel encapsulates the state and logic for the reusable prompt
// component. It holds the configuration and the internal state related to
// input, history, autocompletion, and output display. It implements the
// bubbletea.Model interface.
type PromptModel struct {
	// config holds the user-provided configuration.
	config PromptConfig

	// lines stores each line of the potentially multi-line input.
	lines []string

	// cursorRow is the zero-based row index where the cursor is located.
	cursorRow int

	// cursorCol is the zero-based column index where the cursor is located
	// within the current row.
	cursorCol int

	// history holds previously executed commands as strings.
	history []string

	// historyIndex is the current index when navigating history (-1 means
	// not navigating).
	historyIndex int

	// suggestions holds the current list of generated suggestions.
	suggestions []Suggestion

	// showPopup indicates if the suggestion popup should be visible.
	showPopup bool

	// selectedSuggestionIndex is the index of the currently highlighted
	// suggestion in the list.
	selectedSuggestionIndex int

	// lastSuggestedWord is the word fragment used to generate the current
	// suggestions.
	lastSuggestedWord string

	// popupScrollOffset is the index of the first suggestion visible in a
	// scrollable popup.
	popupScrollOffset int

	// Output Display State: Holds the result from the last command
	// execution. lastOutput stores the string returned by the ExecuteFn to
	// display temporarily.
	lastOutput string
}

// NewPromptModel creates a new prompt model instance with the given
// configuration. It initializes the internal state and ensures configuration
// defaults are applied. Returns a pointer suitable for use with
// bubbletea.NewProgram.
func NewPromptModel(config PromptConfig) *PromptModel {
	// Ensure default functions are set if the user provided nil.
	if config.IsCompleteFn == nil {
		config.IsCompleteFn = DefaultIsComplete
	}

	if config.IsWordCharFn == nil {
		config.IsWordCharFn = DefaultIsWordChar
	}

	// Ensure styles are initialized (basic check: see if a core style is
	// uninitialized).
	if config.Styles.Prompt.GetForeground() == (lipgloss.NoColor{}) {
		config.Styles = DefaultPromptStyles()
	}

	// Ensure PopupMaxHeight has a positive value.
	if config.PopupMaxHeight <= 0 {
		// Default to 6 if invalid
		config.PopupMaxHeight = 6
	}

	return &PromptModel{
		config:       config,
		lines:        []string{""},
		cursorRow:    0,
		cursorCol:    0,
		history:      []string{},
		historyIndex: -1,
	}
}

// Init initializes the PromptModel. Currently, it performs no initial actions.
// It satisfies the bubbletea.Model interface. Can return an initial command.
func (m *PromptModel) Init() tea.Cmd {
	// No initial command needed for the prompt itself.
	return nil
}

// Update handles incoming Bubble Tea messages (like key presses, window size
// changes) and updates the PromptModel's state accordingly. It satisfies the
// bubbletea.Model interface.
func (m *PromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Process messages based on their type.
	switch msg := msg.(type) {
	// Handle key press messages.
	case tea.KeyMsg:
		// Delegate key press handling to a dedicated method. Pass the
		// pointer receiver (*m) because handlers modify the model.
		return m.handleKeyPress(msg)

		// Handle other message types (e.g., window resize) if needed in
		// the future.
		// case tea.WindowSizeMsg:
		//     // Example: m.handleResize(msg)
		//     return m, nil
	}

	// If the message type is not handled, return the model unchanged.
	return m, nil
}

// handleKeyPress acts as the central dispatcher for key press events. It routes
// the key press to more specific handler methods based on the key type.
func (m *PromptModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Clear the output from the previous command as soon as the user
	// interacts again (except when pressing Enter to potentially submit).
	if msg.Type != tea.KeyEnter {
		m.clearLastOutputOnEdit(msg.Type)
	}

	// Dispatch based on the specific key type for reliable handling.
	switch msg.Type {
	case tea.KeyCtrlC, tea.KeyEsc:
		// Exit the application.
		return m, tea.Quit

	case tea.KeyEnter:
		// Handle command submission or newline insertion.
		m.handleEnter()
		return m, nil

	case tea.KeyBackspace:
		// Handle character deletion or line merging.
		m.handleBackspace()
		// Update autocomplete suggestions based on the change.
		m.updateAutocomplete()
		return m, nil

	case tea.KeyTab:
		// Handle attempt to apply the selected autocomplete suggestion.
		m.handleAutocompleteTab()
		return m, nil

	case tea.KeyUp:
		// Handle moving cursor up, navigating history, or suggestion
		// list.
		m.handleUpArrow()
		return m, nil

	case tea.KeyDown:
		// Handle moving cursor down, navigating history, or suggestion
		// list.
		m.handleDownArrow()
		return m, nil

	case tea.KeyLeft:
		// Handle moving cursor left.
		m.moveCursorLeft()
		// Clear suggestions as horizontal movement usually cancels
		// completion intent.
		m.clearAutocomplete()
		return m, nil

	case tea.KeyRight:
		// Handle moving cursor right.
		m.moveCursorRight()
		// Clear suggestions as horizontal movement usually cancels
		// completion intent.
		m.clearAutocomplete()
		return m, nil

	case tea.KeySpace:
		// Handle spacebar press. Insert a space character.
		m.insertCharacter(' ')
		// Update/clear suggestions (often clears after space).
		m.updateAutocomplete()
		return m, nil

	case tea.KeyRunes:
		// Handle input of regular printable characters. Insert the
		// typed characters.
		m.insertRunes(msg.Runes)
		// Update suggestions based on the new input.
		m.updateAutocomplete()
		return m, nil

	default:
		// Ignore any other key types not explicitly handled.
		return m, nil
	}
}

// clearLastOutputOnEdit clears the display area for the previous command's
// output if the pressed key indicates editing or significant navigation is
// occurring.
func (m *PromptModel) clearLastOutputOnEdit(keyType tea.KeyType) {
	switch keyType {
	// List of key types that trigger clearing the output.
	case tea.KeyBackspace, tea.KeyRunes, tea.KeySpace,
		tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight:
		// Reset the lastOutput field.
		m.lastOutput = ""
	}
}

// insertCharacter is a convenience method to insert a single character rune.
func (m *PromptModel) insertCharacter(r rune) {
	m.insertRunes([]rune{r})
}

// insertRunes inserts a slice of printable characters (runes) at the cursor
// position. It filters out non-printable runes and resets history Browse mode.
func (m *PromptModel) insertRunes(runes []rune) {
	// Filter out potential control characters that might slip through as
	// runes.
	printableRunes := []rune{}
	for _, r := range runes {
		// Basic check for printable range (includes space).
		if r >= ' ' {
			printableRunes = append(printableRunes, r)
		}
	}

	// Only proceed if there are actual printable runes to insert.
	if len(printableRunes) == 0 {
		return
	}

	// Get the current line where the cursor is.
	line := m.lines[m.cursorRow]

	// Reconstruct the line with the new runes inserted at the cursor
	// column.
	m.lines[m.cursorRow] = line[:m.cursorCol] + string(printableRunes) +
		line[m.cursorCol:]

	// Move the cursor forward by the number of runes inserted.
	// Note: Using len() is okay here as we inserted printable runes.
	m.cursorCol += len(printableRunes)

	// If the user types anything, they are no longer Browse history.
	m.historyIndex = -1
}

// deleteBeforeCursor handles the Backspace key logic: deleting a character
// or merging the current line with the previous one if at the start of a line.
func (m *PromptModel) deleteBeforeCursor() {
	if m.cursorCol > 0 {
		// Case 1: Cursor is not at the beginning of the line.
		// Delete the character immediately before the cursor.
		line := m.lines[m.cursorRow]

		// Reconstruct the line without the character at cursorCol-1.
		m.lines[m.cursorRow] = line[:m.cursorCol-1] + line[m.cursorCol:]

		// Move the cursor back one position.
		m.cursorCol--
	} else if m.cursorRow > 0 {
		// Case 2: Cursor is at the beginning of a line (but not the
		// first line). Merge this line with the previous line.
		prevLine := m.lines[m.cursorRow-1]
		currentLine := m.lines[m.cursorRow]

		// Store the target cursor column (end of the previous line).
		targetCol := len(prevLine)

		// Append the current line's content to the previous line.
		m.lines[m.cursorRow-1] += currentLine

		// Remove the current line from the slice by slicing around it.
		m.lines = append(
			m.lines[:m.cursorRow], m.lines[m.cursorRow+1:]...,
		)

		// Move the cursor position up to the previous line.
		m.cursorRow--

		// Set the cursor column to where the merge happened.
		m.cursorCol = targetCol
	}
	// If cursorRow is 0 and cursorCol is 0, Backspace does nothing.
}

// insertNewline handles inserting a newline character. It splits the current
// line at the cursor position into two lines.
func (m *PromptModel) insertNewline() {
	// Get the content of the current line.
	currentLine := m.lines[m.cursorRow]

	// Get the part of the line before the cursor.
	left := currentLine[:m.cursorCol]

	// Get the part of the line at and after the cursor.
	right := currentLine[m.cursorCol:]

	// Construct the new slice of lines.
	// 1. Copy all lines before the current row.
	newLines := append([]string{}, m.lines[:m.cursorRow]...)

	// 2. Add the 'left' part as the new content of the current row.
	newLines = append(newLines, left)

	// 3. Add the 'right' part as a new line immediately following.
	newLines = append(newLines, right)

	// 4. Append any lines that existed after the original cursor row.
	if m.cursorRow+1 < len(m.lines) {
		newLines = append(newLines, m.lines[m.cursorRow+1:]...)
	}

	// Replace the lines slice.
	m.lines = newLines
	// Move the cursor down to the newly created line ('right' part).
	m.cursorRow++
	// Position the cursor at the beginning of the new line.
	m.cursorCol = 0

	// Clean up any potential extra blank lines created at the end.
	m.cleanupExtraBlankLines()

	// Ensure the cursor row index is still valid after potential cleanup.
	// (Cleanup might remove the line the cursor just moved to).
	if m.cursorRow >= len(m.lines) && len(m.lines) > 0 {
		// Move cursor to the new last line.
		m.cursorRow = len(m.lines) - 1
	} else if len(m.lines) == 0 {
		// Safety check: If all lines were somehow removed, reset to a
		// single empty line.
		m.lines = []string{""}
		m.cursorRow = 0
	}
}

// moveCursorUp moves the cursor up one line. If the target line is shorter than
// the current column, it snaps the cursor to the end of that line.
func (m *PromptModel) moveCursorUp() {
	// Only move up if not already at the first row.
	if m.cursorRow > 0 {
		m.cursorRow--

		// Check if the target column position exists on the new line.
		if m.cursorCol > len(m.lines[m.cursorRow]) {
			// If not, move cursor to the end of the shorter line.
			m.cursorCol = len(m.lines[m.cursorRow])
		}
	}
}

// moveCursorDown moves the cursor down one line. If the target line is shorter
// than the current column, it snaps the cursor to the end of that line.
func (m *PromptModel) moveCursorDown() {
	// Only move down if not already at the last row.
	if m.cursorRow < len(m.lines)-1 {
		m.cursorRow++

		// Check if the target column position exists on the new line.
		if m.cursorCol > len(m.lines[m.cursorRow]) {
			// If not, move cursor to the end of the shorter line.
			m.cursorCol = len(m.lines[m.cursorRow])
		}
	}
}

// moveCursorLeft moves the cursor one position left. If at the beginning of a
// line (and not the first line), it wraps to the end of the previous line.
func (m *PromptModel) moveCursorLeft() {
	// If not at the beginning of the line, simply move left.
	if m.cursorCol > 0 {
		m.cursorCol--
	} else if m.cursorRow > 0 {
		// If at the beginning of a line (but not the first), wrap to
		// the previous line.
		m.cursorRow--
		// Position cursor at the end of the previous line.
		m.cursorCol = len(m.lines[m.cursorRow])
	}
}

// moveCursorRight moves the cursor one position right. If at the end of a line
// (and not the last line), it wraps to the beginning of the next line.
func (m *PromptModel) moveCursorRight() {
	// If not at the end of the current line, simply move right.
	if m.cursorCol < len(m.lines[m.cursorRow]) {
		m.cursorCol++
	} else if m.cursorRow < len(m.lines)-1 {
		// If at the end of a line (but not the last), wrap to the next
		// line.
		m.cursorRow++
		// Position cursor at the beginning of the next line.
		m.cursorCol = 0
	}
}

// getTextBeforeCursor returns all text from the beginning of the input up to
// the current cursor position, joining lines with newlines. This is used to
// provide context to the AutoCompleteFunc.
func (m *PromptModel) getTextBeforeCursor() string {
	// Safety check for cursor row bounds.
	if m.cursorRow < 0 || m.cursorRow >= len(m.lines) {
		return ""
	}

	var sb strings.Builder

	// Append all lines *before* the current cursor row.
	for i := 0; i < m.cursorRow; i++ {
		sb.WriteString(m.lines[i])

		// Add newline separator between lines.
		sb.WriteRune('\n')
	}

	// Append the part of the current line *before* the cursor column.
	currentLine := m.lines[m.cursorRow]
	if m.cursorCol > 0 {
		// Use runes for slicing to handle multi-byte characters
		// correctly.
		runes := []rune(currentLine)

		// Ensure the column index is within the bounds of the rune
		// slice.
		col := min(m.cursorCol, len(runes))
		sb.WriteString(string(runes[:col]))
	}

	return sb.String()
}

// updateAutocomplete checks the context around the cursor and calls the
// configured AutoCompleteFunc if appropriate, updating the suggestion state.
func (m *PromptModel) updateAutocomplete() {
	// Get the function that defines word characters from the config.
	isWordCharFn := m.config.IsWordCharFn

	// Get the potential word fragment ending at the cursor.
	word := m.currentWordFragment(isWordCharFn)

	// Determine if the suggestion popup should be cleared/hidden.
	clear := false
	if word == "" {
		// No word fragment means no suggestions.
		clear = true
	} else if m.cursorCol > 0 && m.cursorRow < len(m.lines) {
		// Check if the character immediately before the cursor is a
		// word character. If not (e.g., space, punctuation), clear
		// suggestions.
		lineRunes := []rune(m.lines[m.cursorRow])

		// Check bounds before accessing rune slice.
		if m.cursorCol <= len(lineRunes) &&
			!isWordCharFn(lineRunes[m.cursorCol-1]) {
			clear = true
		} else if m.cursorCol > len(lineRunes) {
			// Cursor is out of bounds, clear.
			clear = true
		}
	} else if m.cursorCol == 0 {
		// Cursor is at the start of the line, clear suggestions.
		clear = true
	}

	// If clearing, reset autocomplete state and return.
	if clear {
		m.clearAutocomplete()
		return
	}

	// If the word fragment has changed since last time, generate new
	// suggestions.
	if word != m.lastSuggestedWord {
		// Reset selection to the top.
		m.selectedSuggestionIndex = 0

		// Reset scroll position.
		m.popupScrollOffset = 0

		// Check if an autocomplete function is configured.
		if m.config.AutoCompleteFn != nil {
			// Get the text context before the cursor.
			textBefore := m.getTextBeforeCursor()
			// Call the configured function to get suggestions.
			m.suggestions = m.config.AutoCompleteFn(
				textBefore, word,
			)
		} else {
			// No function configured, ensure suggestions are empty.
			m.suggestions = nil
		}
		// Show the popup only if suggestions were returned.
		m.showPopup = len(m.suggestions) > 0

		// Store the word fragment that generated these suggestions.
		m.lastSuggestedWord = word
	} else if len(m.suggestions) == 0 {
		// If the word fragment hasn't changed, but there are no
		// suggestions (e.g., function returned empty list), ensure the
		// popup is hidden.
		m.showPopup = false
	}
}

// clearAutocomplete hides the suggestion popup and resets related state
// variables.
func (m *PromptModel) clearAutocomplete() {
	// Clear the suggestion slice.
	m.suggestions = nil

	// Hide the popup.
	m.showPopup = false

	// Reset selection index.
	m.selectedSuggestionIndex = 0

	// Clear the last word fragment.
	m.lastSuggestedWord = ""

	// Reset the scroll offset.
	m.popupScrollOffset = 0
}

// navigateAutocompleteUp moves the selection index up within the suggestion
// list, handling wrapping and adjusting the scroll offset if necessary.
func (m *PromptModel) navigateAutocompleteUp() {
	// Only navigate if the popup is shown and suggestions exist.
	if m.showPopup && len(m.suggestions) > 0 {
		// Decrement the selected index.
		m.selectedSuggestionIndex--

		// Check if we've moved past the top suggestion.
		if m.selectedSuggestionIndex < 0 {
			// Wrap around to the last suggestion.
			m.selectedSuggestionIndex = len(m.suggestions) - 1

			// Scroll the view to show the bottom part of the list.
			m.popupScrollOffset = max(
				0, len(m.suggestions)-m.config.PopupMaxHeight,
			)
		} else if m.selectedSuggestionIndex < m.popupScrollOffset {
			// If the new selection is above the current visible
			// area, scroll up. Set the scroll offset so the
			// selection is the first visible item.
			m.popupScrollOffset = m.selectedSuggestionIndex
		}
	}
}

// navigateAutocompleteDown moves the selection index down within the suggestion
// list, handling wrapping and adjusting the scroll offset if necessary.
func (m *PromptModel) navigateAutocompleteDown() {
	// Only navigate if the popup is shown and suggestions exist.
	if m.showPopup && len(m.suggestions) > 0 {
		// Increment the selected index.
		m.selectedSuggestionIndex++

		// Check if we've moved past the last suggestion.
		if m.selectedSuggestionIndex >= len(m.suggestions) {
			// Wrap around to the first suggestion.
			m.selectedSuggestionIndex = 0

			// Scroll the view to the top.
			m.popupScrollOffset = 0
		} else if m.selectedSuggestionIndex >=
			m.popupScrollOffset+m.config.PopupMaxHeight {
			// If the new selection is below the current visible
			// area, scroll down. Adjust the scroll offset so the
			// selection is the last visible item.
			m.popupScrollOffset = m.selectedSuggestionIndex -
				m.config.PopupMaxHeight + 1
		}
	}
}

// applyAutocomplete replaces the current word fragment with the selected
// suggestion's Word.
func (m *PromptModel) applyAutocomplete() {
	// Only apply if the popup is shown and suggestions exist.
	if m.showPopup && len(m.suggestions) > 0 {
		// Get the currently selected suggestion struct.
		selectedSuggestion := m.suggestions[m.selectedSuggestionIndex]

		// Extract the text to be inserted.
		selectedText := selectedSuggestion.Text

		// Get current line context.
		line := m.lines[m.cursorRow]
		col := m.cursorCol

		// Use configured function
		isWordCharFn := m.config.IsWordCharFn

		// Find the starting position of the word fragment being
		// completed. Scan backwards from the cursor position.
		start := col
		runes := []rune(line)

		for start > 0 {
			// Stop scanning if a non-word character is found.
			if isWordCharFn(runes[start-1]) {
				start--
			} else {
				break
			}
		}
		// Ensure start index is valid.
		if start < 0 {
			start = 0
		}

		// Reconstruct the line:
		// Part before fragment + selected suggestion word + part after
		// original cursor position. Part before the word fragment.
		prefix := string(runes[:start])

		// Part after the original cursor position.
		suffix := ""

		// Safely get the suffix, handling potential out-of-bounds cursor.
		if col < len(runes) {
			suffix = string(runes[col:])
		} else if col > len(runes) {
			// Correct cursor position if it somehow went out of
			// bounds.
			col = len(runes)
		} // If col == len(runes), suffix remains "" (correct).

		// Update the current line in the model.
		m.lines[m.cursorRow] = prefix + selectedText + suffix

		// Move the cursor to the end of the inserted suggestion word.
		m.cursorCol = start + len(selectedText)

		// Hide the popup and reset autocomplete state.
		m.clearAutocomplete()
	}
}

// navigateHistoryUp loads the previous command from history into the input
// area.
func (m *PromptModel) navigateHistoryUp() {
	// Do nothing if history is empty.
	if len(m.history) == 0 {
		return
	}

	// If not currently Browse history, start from the most recent entry.
	if m.historyIndex == -1 {
		m.historyIndex = len(m.history) - 1
	} else if m.historyIndex > 0 {
		// If already Browse, move to the previous (older) entry.
		m.historyIndex--
	} else {
		// Already at the oldest entry (index 0), do nothing more.
		return
	}
	// Load the content of the selected history entry.
	m.loadHistoryEntry()
}

// navigateHistoryDown loads the next (more recent) command from history, or
// clears the input if moving past the most recent entry.
func (m *PromptModel) navigateHistoryDown() {
	// Do nothing if not currently Browse history.
	if m.historyIndex == -1 {
		return
	}

	// Check if there are more recent entries to navigate to.
	if m.historyIndex < len(m.history)-1 {
		// Move to the next (more recent) history entry.
		m.historyIndex++

		// Load its content.
		m.loadHistoryEntry()
	} else {
		// Currently at the most recent entry (last item in history).
		// Exiting history mode downwards clears the input line.
		m.historyIndex = -1

		// Reset to a single empty line.
		m.lines = []string{""}
		m.cursorRow = 0
		m.cursorCol = 0

		// Ensure suggestions are cleared.
		m.clearAutocomplete()
	}
}

// loadHistoryEntry replaces the current input lines with the history entry
// specified by the current m.historyIndex.
func (m *PromptModel) loadHistoryEntry() {
	// Check if the history index is valid.
	if m.historyIndex >= 0 && m.historyIndex < len(m.history) {
		// Split the stored command (which might be multi-line) into
		// lines.
		m.lines = strings.Split(m.history[m.historyIndex], "\n")
		// Position the cursor at the end of the loaded command.
		m.cursorRow = len(m.lines) - 1
		// Handle case where history entry might be empty or invalid.
		if m.cursorRow < 0 {
			m.cursorRow = 0
		}
		if len(m.lines) > 0 {
			m.cursorCol = len(m.lines[m.cursorRow])
		} else {
			// If history entry resulted in no lines, reset to a
			// safe state.
			m.lines = []string{""}
			m.cursorRow = 0
			m.cursorCol = 0
		}
		// Clear any autocomplete suggestions shown before history
		// navigation.
		m.clearAutocomplete()
	}
}

// handleEnter determines whether to submit the command or insert a newline,
// based on the configured IsCompleteFn.
func (m *PromptModel) handleEnter() {
	// Get the current input, potentially spanning multiple lines.
	fullInput := m.getCurrentInput()
	// Use the configured function to check if the input is complete.
	isComplete := m.config.IsCompleteFn(fullInput)

	// Check if complete and avoid submitting just an empty semicolon.
	if isComplete && strings.TrimSpace(fullInput) != ";" {
		// Check if an execution function is configured.
		if m.config.ExecuteFn != nil {
			// Call the configured function and store its output.
			output := m.config.ExecuteFn(fullInput)
			// Format the output for display in the View.
			m.lastOutput = fmt.Sprintf("\n--- Executing ---\n%s\n-----------------\n", output)
		} else {
			// Provide feedback if no execution function is set.
			m.lastOutput = "\n--- No ExecuteFn Configured ---\n"
		}

		// Add the submitted command to history if it's not just
		// whitespace.
		if strings.TrimSpace(fullInput) != "" {
			m.history = append(m.history, fullInput)
		}

		// Reset the input state for the next command.
		m.lines = []string{""}
		m.cursorRow = 0
		m.cursorCol = 0

		// Exit history Browse mode.
		m.historyIndex = -1

		// Clear suggestions.
		m.clearAutocomplete()

	} else {
		// Input is not complete, so insert a newline.
		m.insertNewline()

		// Clear suggestions when inserting a newline.
		m.clearAutocomplete()
	}
}

// handleBackspace calls the core deletion/merge logic.
func (m *PromptModel) handleBackspace() {
	m.deleteBeforeCursor()
	// Note: updateAutocomplete is called after this in handleKeyPress
}

// handleAutocompleteTab calls the logic to apply the selected suggestion.
func (m *PromptModel) handleAutocompleteTab() {
	m.applyAutocomplete()
}

// handleUpArrow dispatches to the appropriate action based on context:
// navigate suggestions, navigate history, or move cursor up.
func (m *PromptModel) handleUpArrow() {
	if m.showPopup {
		// If popup is visible, navigate suggestions.
		m.navigateAutocompleteUp()
	} else if m.cursorRow == 0 {
		// If at the top line and no popup, navigate history.
		m.navigateHistoryUp()
	} else {
		// Otherwise, just move the cursor up within the input area.
		m.moveCursorUp()
	}
}

// handleDownArrow dispatches to the appropriate action based on context:
// navigate suggestions, navigate history, or move cursor down.
func (m *PromptModel) handleDownArrow() {
	if m.showPopup {
		// If popup is visible, navigate suggestions.
		m.navigateAutocompleteDown()
	} else if m.historyIndex != -1 {
		// If currently Browse history, navigate history down.
		m.navigateHistoryDown()
	} else {
		// Otherwise, just move the cursor down within the input area.
		m.moveCursorDown()
	}
}

// View generates the string representation of the UI based on the current model
// state. It uses the configured styles and prompts.
func (m PromptModel) View() string {
	var sb strings.Builder

	// Get the configured styles.
	styles := m.config.Styles

	// 1. Display output from the last executed command, if any.
	if m.lastOutput != "" {
		// Trim trailing newlines from the stored output to prevent
		// double spacing.
		sb.WriteString(strings.TrimRight(m.lastOutput, "\n"))

		// Add exactly one newline after the output block.
		sb.WriteRune('\n')
	}

	// 2. Render the input lines.
	for i, line := range m.lines {
		// Determine the correct prompt string based on the line number.
		prefix := m.config.PromptPrimary
		if i > 0 {
			prefix = m.config.PromptSecondary
		}
		// Render the prompt string with its configured style.
		sb.WriteString(styles.Prompt.Render(prefix))

		// Check if this is the line the cursor is currently on.
		if i == m.cursorRow {
			// Render the line character by character to insert the
			// cursor. Use runes for correct indexing.
			runes := []rune(line)
			for j := 0; j <= len(runes); j++ {
				// Check if this is the cursor's column
				// position.
				if j == m.cursorCol {
					// Determine the character under the
					// cursor (or space if at end).
					cursorChar := " "
					if j < len(runes) {
						cursorChar = string(runes[j])
					}

					// Render the character/space with the
					// cursor style.
					sb.WriteString(
						styles.Cursor.Render(cursorChar),
					)
				}

				// Write the original character if it's not the
				// one under the cursor. Ensure index j is
				// within the bounds of the runes slice.
				if j < len(runes) && j != m.cursorCol {
					sb.WriteRune(runes[j])
				}
			}
		} else {
			// If it's not the cursor line, render the line content
			// directly.
			sb.WriteString(line)
		}

		// Add a newline after rendering the line content, unless it's
		// the very last line AND that line is empty (prevents an extra
		// blank line below the prompt).
		if i < len(m.lines)-1 || line != "" {
			sb.WriteRune('\n')
		}
	}

	// 3. Render the autocomplete popup if it should be visible.
	if m.showPopup && len(m.suggestions) > 0 {
		// Add spacing before the popup if the last line written wasn't
		// a newline.
		if sb.Len() > 0 && sb.String()[sb.Len()-1] != '\n' {
			sb.WriteRune('\n')
		}

		// To hold the rendered suggestion strings.
		suggestionLines := []string{}

		// Determine the range of suggestions to display based on
		// scrolling.
		maxH := m.config.PopupMaxHeight
		numSuggestions := len(m.suggestions)

		// Ensure scroll offset is valid (can become invalid if
		// suggestions change).
		if m.popupScrollOffset >= numSuggestions {
			m.popupScrollOffset = max(0, numSuggestions-1)
		}

		// First visible index.
		startIdx := m.popupScrollOffset

		// Last visible index (exclusive).
		endIdx := min(startIdx+maxH, numSuggestions)

		// Calculate the maximum display width of the suggestion words
		// in the visible range to allow for aligning the descriptions.
		maxWordWidth := 0
		for i := startIdx; i < endIdx; i++ {
			// Use runewidth.StringWidth for accurate width of
			// potentially wide characters.
			width := runewidth.StringWidth(m.suggestions[i].Text)

			if width > maxWordWidth {
				maxWordWidth = width
			}
		}

		// Iterate through the *visible* suggestions only.
		for i := startIdx; i < endIdx; i++ {
			// Get the current suggestion struct.
			sugg := m.suggestions[i]
			textPart := sugg.Text
			descPart := ""

			// Format the description part if enabled and available.
			if m.config.ShowDescription && sugg.Description != "" {
				// Apply the configured description style.
				descPart = styles.Description.Render(
					sugg.Description,
				)
			}

			// Pad the word part with spaces to align the
			// descriptions. Calculate padding needed based on rune
			// width.
			padding := maxWordWidth - runewidth.StringWidth(
				textPart,
			)

			// Avoid negative padding.
			if padding < 0 {
				padding = 0
			}
			paddedWord := textPart + strings.Repeat(" ", padding)

			// Combine the padded word and the description using
			// lipgloss.JoinHorizontal. This helps manage spacing
			// and potential future styling. Add separator spaces.
			line := lipgloss.JoinHorizontal(
				lipgloss.Left, paddedWord, "  ", descPart,
			)

			// Determine the style for the current line (selected or
			// unselected).
			style := styles.UnselectedItem
			if i == m.selectedSuggestionIndex {
				style = styles.SelectedItem
			}

			// Render the complete line with the appropriate style.
			suggestionLines = append(
				suggestionLines, style.Render(line),
			)
		}

		// Join the rendered lines and apply the overall popup box style.
		sb.WriteString(styles.PopupBox.Render(
			strings.Join(suggestionLines, "\n")),
		)
	}

	return sb.String()
}

// joinNonEmptyLines combines lines from a slice, removing any trailing lines
// that consist only of whitespace. Used before executing a command.
func joinNonEmptyLines(lines []string) string {
	end := len(lines)

	// Iterate backwards through the lines.
	for end > 0 {
		// Stop at the first line that contains non-whitespace
		// characters.
		if strings.TrimSpace(lines[end-1]) != "" {
			break
		}

		// Decrement end pointer if the line is empty/whitespace.
		end--
	}

	// Join the lines up to the last non-empty one found.
	return strings.Join(lines[:end], "\n")
}

// getCurrentInput is a convenience method on the model to get the processed
// input string.
func (m *PromptModel) getCurrentInput() string {
	return joinNonEmptyLines(m.lines)
}

// cleanupExtraBlankLines removes consecutive blank lines specifically from the
// *end* of the input lines slice. This prevents excessive blank lines during
// input.
func (m *PromptModel) cleanupExtraBlankLines() {
	// Loop while there are at least two lines and the last two are blank.
	for len(m.lines) > 1 &&
		strings.TrimSpace(m.lines[len(m.lines)-1]) == "" &&
		strings.TrimSpace(m.lines[len(m.lines)-2]) == "" {
		// Slice off the last line.
		m.lines = m.lines[:len(m.lines)-1]
	}
}

// currentWordFragment identifies the sequence of "word" characters (as defined
// by the configured IsWordCharFunc) immediately preceding the cursor. Returns
// an empty string if no word fragment is found there.
func (m *PromptModel) currentWordFragment(isWordCharFn IsWordCharFunc) string {
	// Basic bounds checks for cursor position.
	if m.cursorRow >= len(m.lines) {
		return ""
	}
	line := m.lines[m.cursorRow]
	if m.cursorCol == 0 {
		return ""
	}

	// Work with runes for multi-byte character safety.
	lineRunes := []rune(line)
	// Check column bounds against rune count.
	if m.cursorCol > len(lineRunes) {
		return ""
	}

	// Scan backwards from the cursor column to find the start of the word
	// fragment.
	start := m.cursorCol
	for start > 0 {
		// Use the configured function to check if the character is part
		// of a word.
		if isWordCharFn(lineRunes[start-1]) {
			// Continue scanning left.
			start--
		} else {
			// Stop at the first non-word character.
			break
		}
	}

	// Verify that a fragment was actually found (start < cursorCol) and
	// that the character *immediately* before the cursor is indeed a word
	// character. This prevents matching if the cursor is right after a
	// space, e.g., "SELECT |".
	if start < m.cursorCol && m.cursorCol > 0 &&
		isWordCharFn(lineRunes[m.cursorCol-1]) {

		// Return the identified word fragment as a string.
		return string(lineRunes[start:m.cursorCol])
	}

	// No valid word fragment found ending at the cursor.
	return ""
}
