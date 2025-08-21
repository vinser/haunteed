package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/vinser/haunteed/internal/style"
)

// Page renders page with title at the top, content block and footer at the botttom
// Style of content leave intact
func Page(title, renderedContent, footer string, width, height, termWidth, termHeight int) string {
	// Render top pattern of slashes
	renderedTopPattern := style.TopPattern.Render(strings.Repeat("/", width))

	// Render the title
	renderedTitle := style.Title.Render(title)

	// Render the content
	// renderedContent := content

	// Render the footer
	renderedFooter := style.Footer.Render(footer)

	// Calculate available height for content after accounting for title and footer
	availableHeight := height - lipgloss.Height(renderedTopPattern) - lipgloss.Height(renderedTitle) - lipgloss.Height(renderedFooter)

	// Place content vertically centered within the available height
	centeredContent := lipgloss.PlaceVertical(availableHeight, lipgloss.Center, renderedContent)

	// Assemble the final page
	view := lipgloss.JoinVertical(
		lipgloss.Left,
		renderedTopPattern,
		renderedTitle,
		centeredContent,
		renderedFooter,
	)
	if termWidth > 0 && termHeight > 0 {
		return lipgloss.Place(termWidth, termHeight, lipgloss.Center, lipgloss.Center, view)
	}
	return view
}
