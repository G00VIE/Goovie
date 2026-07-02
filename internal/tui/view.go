package tui

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"

	"bubble-stream/internal/assets"
	"bubble-stream/internal/config"
	"github.com/charmbracelet/lipgloss"
	"github.com/disintegration/imaging"
)

var (
	TitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("99")).
			Align(lipgloss.Left)

	cardStyle = lipgloss.NewStyle().
			Align(lipgloss.Center)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("9")).
			Align(lipgloss.Center)
)

func LoadPNG(filename string) (image.Image, error) {
	f, err := assets.EmbeddedFiles.Open(filename)
	if err == nil {
		defer f.Close()
		return png.Decode(f)
	}
	return nil, err
}

func renderDatabaseMatchCamera(title string, items [][2]string, cursor int, searchStr string) string {
	maxNameLen := 0
	for _, item := range items {
		l := len([]rune(item[0]))
		if l > maxNameLen {
			maxNameLen = l
		}
	}
	if maxNameLen > 45 {
		maxNameLen = 45
	}
	if maxNameLen < 20 {
		maxNameLen = 20
	}
	// Add more space between name and release year divider
	maxNameLen += 10

	rawTitle := strings.ToUpper(title) + ":"
	if searchStr != "" {
		rawTitle += " " + searchStr
	}
	spacedTitle := strings.Join(strings.Split(rawTitle, ""), " ")
	spacedTitle += " \033[5m_\033[0m"

	visibleTitleLen := len([]rune(strings.Join(strings.Split(rawTitle, ""), " "))) + 2

	innerWidth := 8 + maxNameLen
	if innerWidth < visibleTitleLen+2 {
		innerWidth = visibleTitleLen + 2
		maxNameLen = innerWidth - 8
	}

	leftTitlePad := innerWidth - visibleTitleLen
	paddedTitle := " " + spacedTitle + strings.Repeat(" ", leftTitlePad-1)

	rightStrPlain := fmt.Sprintf("  %d / %d  ", cursor+1, len(items))
	rightWidth := len(rightStrPlain)

	topBorder := "    ┌" + strings.Repeat("─", innerWidth) + "┬" + strings.Repeat("─", rightWidth) + "┐"
	headerText := fmt.Sprintf("    │%s│%s│", paddedTitle, rightStrPlain)
	bottomBorder := "    └" + strings.Repeat("─", innerWidth) + "┴" + strings.Repeat("─", rightWidth) + "┘"

	s := fmt.Sprintf("\n%s\n%s\n%s\n\n", topBorder, headerText, bottomBorder)

	const maxLocalItems = 10
	start := cursor - (maxLocalItems / 2)
	if start < 0 {
		start = 0
	}
	end := start + maxLocalItems
	if end > len(items) {
		end = len(items)
		start = end - maxLocalItems
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		isTarget := (i == cursor)
		name := items[i][0]
		details := items[i][1]

		if len([]rune(name)) > maxNameLen {
			name = string([]rune(name)[:maxNameLen-3]) + "..."
		} else {
			namePad := maxNameLen - len([]rune(name))
			name += strings.Repeat(" ", namePad)
		}
		
		displayRow := fmt.Sprintf("%2d  %s   │   %s", i+1, name, details)

		if isTarget {
			s += fmt.Sprintf("  \033[30;47m  \033[5m●\033[0;30;47m %s  \033[0m\n", displayRow)
		} else {
			s += fmt.Sprintf("      %s\n", displayRow)
		}
	}
	return s
}

var brailleMatrix = [4][2]int{
	{0x01, 0x08},
	{0x02, 0x10},
	{0x04, 0x20},
	{0x40, 0x80},
}

func luminance(c color.Color) float64 {
	r, g, b, _ := c.RGBA()
	return 0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(b>>8)
}

func renderImageToLines(img image.Image, wCells int, selected bool) []string {
	if img == nil {
		lines := make([]string, 10)
		for i := range lines {
			lines[i] = strings.Repeat(" ", wCells)
		}
		return lines
	}

	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()
	if imgWidth == 0 || imgHeight == 0 {
		lines := make([]string, 10)
		for i := range lines {
			lines[i] = strings.Repeat(" ", wCells)
		}
		return lines
	}

	aspect := float64(imgHeight) / float64(imgWidth)

	pixelWidth := wCells * 2
	pixelHeight := int(float64(wCells) * aspect * 2)
	pixelHeight = pixelHeight - (pixelHeight % 4)
	if pixelHeight < 4 {
		pixelHeight = 4
	}

	hCells := pixelHeight / 4

	dst := imaging.Resize(img, pixelWidth, pixelHeight, imaging.Lanczos)

	lines := make([]string, hCells)
	factor := 1.0
	if !selected {
		factor = 0.4
	}

	for y := 0; y < pixelHeight; y += 4 {
		var row strings.Builder
		for x := 0; x < pixelWidth; x += 2 {
			var brailleVal int
			var rSum, gSum, bSum, count float64

			for dy := 0; dy < 4; dy++ {
				for dx := 0; dx < 2; dx++ {
					c := dst.At(x+dx, y+dy)
					_, _, _, a := c.RGBA()

					// Ignore fully transparent pixels
					if a == 0 {
						continue
					}

					lum := luminance(c)

					if lum > 50.0 {
						brailleVal += brailleMatrix[dy][dx]

						r, g, b, _ := c.RGBA()
						rSum += float64(r >> 8)
						gSum += float64(g >> 8)
						bSum += float64(b >> 8)
						count++
					}
				}
			}

			brailleChar := string(rune(0x2800 + brailleVal))

			if count > 0 {
				rawR := (rSum / count) * factor * 1.2
				rawG := (gSum / count) * factor * 1.2
				rawB := (bSum / count) * factor * 1.2

				if rawR > 255 {
					rawR = 255
				}
				if rawG > 255 {
					rawG = 255
				}
				if rawB > 255 {
					rawB = 255
				}

				avgR := uint8(rawR)
				avgG := uint8(rawG)
				avgB := uint8(rawB)

				fmt.Fprintf(&row, "\x1b[38;2;%d;%d;%dm%s", avgR, avgG, avgB, brailleChar)
			} else if brailleVal > 0 {
				row.WriteString(brailleChar)
			} else {
				row.WriteString(" ")
			}
		}
		row.WriteString("\x1b[0m")
		lines[y/4] = row.String()
	}

	return lines
}

func getStyledLabel(label string, wCells int, selected bool, r, g, b int) string {
	colorHex := fmt.Sprintf("#%02x%02x%02x", r, g, b)

	style := lipgloss.NewStyle().
		Width(wCells).
		Align(lipgloss.Center)

	if selected {
		style = style.Background(lipgloss.Color(colorHex)).Foreground(lipgloss.Color("#000000"))
	} else {
		style = style.Foreground(lipgloss.Color(colorHex))
	}

	return style.Render(label)
}

func centerStr(s string, w int) string {
	if w <= len(s) {
		return s
	}
	pad := (w - len(s)) / 2
	return strings.Repeat(" ", pad) + s
}

func renderMenuCamera(items []string, cursor int, termHeight int, title string) string {
	s := fmt.Sprintf("\n  %s\n\n", title)
	maxGlobalRows := termHeight - 8
	if maxGlobalRows < 5 {
		maxGlobalRows = 5
	}

	start := cursor - (maxGlobalRows / 2)
	if start < 0 {
		start = 0
	}
	end := start + maxGlobalRows
	if end > len(items) {
		end = len(items)
		start = end - maxGlobalRows
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		cursorStr := "  "
		if i == cursor {
			cursorStr = "> "
		}
		s += fmt.Sprintf("  %s%s\n", cursorStr, items[i])
	}
	s += "\n  [Up/Down] Select • [Enter] Continue • [Esc] Quit\n"
	return s
}

func renderImageToBlock(img image.Image, wCells int, selected bool) string {
	lines := renderImageToLines(img, wCells, selected)
	return strings.Join(lines, "\n")
}

func PreRenderCache(img image.Image, widths []int) map[int]map[bool]string {
	cache := make(map[int]map[bool]string)
	for _, w := range widths {
		cache[w] = map[bool]string{
			true:  renderImageToBlock(img, w, true),
			false: renderImageToBlock(img, w, false),
		}
	}
	return cache
}

func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  ❌ Error: %v\n\n  Press [Esc] to exit.", m.err)
	}
	termHeight := m.terminalHeight
	if termHeight == 0 {
		termHeight = 24
	}

	switch m.state {
	case StateCheckingAPIKey:
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}
		spinnerView := m.loadingSpinner.View()
		msg := lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("99")).Render(m.loadingPhrase)
		finalUI := lipgloss.JoinVertical(lipgloss.Center, spinnerView, msg)
		return lipgloss.Place(width, termHeight, lipgloss.Center, lipgloss.Center, finalUI)

	case StateSetupAPIKey:
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}
		title := lipgloss.NewStyle().Align(lipgloss.Center).Foreground(lipgloss.Color("99")).Bold(true).Render("Prowlarr API Key Not Found")
		subtitle := lipgloss.NewStyle().Align(lipgloss.Center).Render("Please enter your Prowlarr API Key to continue:")
		inputView := lipgloss.NewStyle().Align(lipgloss.Center).Render(m.setupInput.View())
		
		finalUI := lipgloss.JoinVertical(lipgloss.Center, title, "\n", subtitle, "\n", inputView, "\n", "[Enter] to continue • [Esc] to quit")
		return lipgloss.Place(width, termHeight, lipgloss.Center, lipgloss.Center, finalUI)

	case StateLoadingTorrent:
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}
		spinnerView := m.loadingSpinner.View()
		loadingText := lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Render(m.loadingPhrase)
		
		finalUI := lipgloss.JoinVertical(lipgloss.Center, spinnerView, loadingText)
		return lipgloss.Place(width, termHeight, lipgloss.Center, lipgloss.Center, finalUI)

	case StateFrontPage:
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}

		if width < 30 || termHeight < 15 {
			return lipgloss.Place(width, termHeight, lipgloss.Center, lipgloss.Center, errorStyle.Render("Terminal too small. Please enlarge."))
		}
		
		subtitleStyle := lipgloss.NewStyle().Align(lipgloss.Center)
		enterInst := subtitleStyle.Render("[ ENTER ] to continue")
		backInst := subtitleStyle.Render("[ BACKSPACE ] to go back")
		escInst := subtitleStyle.Render("[ ESC ] to exit")
		
		finalUI := lipgloss.JoinVertical(lipgloss.Center,
			m.cachedFrontTitle,
			"\n\n",
			enterInst,
			"\n",
			backInst,
			"\n",
			escInst,
		)
		return lipgloss.Place(width, termHeight, lipgloss.Center, lipgloss.Center, finalUI)

	case StateModeSelect:
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}

		if width < 30 || termHeight < 15 {
			return lipgloss.Place(width, termHeight, lipgloss.Center, lipgloss.Center, errorStyle.Render("Terminal too small. Please enlarge."))
		}

		labels := []struct {
			label   string
			r, g, b int
		}{
			{"MOVIE", 188, 38, 37},
			{"TV SHOW", 47, 81, 61},
			{"ANIME", 226, 147, 140},
		}

		movieCard := lipgloss.JoinVertical(lipgloss.Center,
			m.cacheMovies[m.activeWCell][m.cursor == 0],
			"\n"+getStyledLabel(labels[0].label, m.activeWCell, m.cursor == 0, labels[0].r, labels[0].g, labels[0].b),
		)
		tvCard := lipgloss.JoinVertical(lipgloss.Center,
			m.cacheTV[m.activeWCell][m.cursor == 1],
			"\n"+getStyledLabel(labels[1].label, m.activeWCell, m.cursor == 1, labels[1].r, labels[1].g, labels[1].b),
		)
		animeCard := lipgloss.JoinVertical(lipgloss.Center,
			m.cacheAnime[m.activeWCell][m.cursor == 2],
			"\n"+getStyledLabel(labels[2].label, m.activeWCell, m.cursor == 2, labels[2].r, labels[2].g, labels[2].b),
		)

		var cardsBlock string
		if width < 100 {
			cardsBlock = lipgloss.JoinVertical(lipgloss.Center, movieCard, "", tvCard, "", animeCard)
		} else {
			spacingPad := strings.Repeat(" ", 6)
			cardsBlock = lipgloss.JoinHorizontal(lipgloss.Top, movieCard, spacingPad, tvCard, spacingPad, animeCard)
		}

		finalUI := lipgloss.JoinVertical(lipgloss.Center,
			m.cachedTitle,
			"\n\n",
			cardsBlock,
		)

		return lipgloss.Place(width, termHeight, lipgloss.Center, lipgloss.Center, finalUI)
	case StateOriginSelect:
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}

		rawName := "SELECT CONTENT TYPE"
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		
		contentLen := len([]rune(spacedName))
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedName := strings.Repeat(" ", leftPad) + spacedName + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│   %d/2", paddedName, m.cursor+1)
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		boxBlock := fmt.Sprintf("\n%s\n%s\n%s\n\n\n", topBorder, headerText, bottomBorder)
		
		opts := []string{"Western", "Asian"}
		var optsUI string
		for i, opt := range opts {
			if i == m.cursor {
				optsUI += fmt.Sprintf("      \033[47;30m • %s \033[0m\n", opt)
			} else {
				optsUI += fmt.Sprintf("         %s \n", opt)
			}
		}
		
		return boxBlock + optsUI
	case StateAnimeTypeSelect:
		rawName := "SELECT MEDIA TYPE"
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		
		contentLen := len([]rune(spacedName))
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedName := strings.Repeat(" ", leftPad) + spacedName + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│   %d/%d", paddedName, m.cursor+1, len(config.AnimeTypeOptions))
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		boxBlock := fmt.Sprintf("\n%s\n%s\n%s\n\n\n", topBorder, headerText, bottomBorder)
		
		var optsUI string
		for i, opt := range config.AnimeTypeOptions {
			if i == m.cursor {
				optsUI += fmt.Sprintf("      \033[47;30m • %s \033[0m\n", opt)
			} else {
				optsUI += fmt.Sprintf("         %s \n", opt)
			}
		}
		return boxBlock + optsUI
	case StateSearch:
		modeStr := "MOVIE"
		if m.isAnime {
			modeStr = "ANIME " + strings.ToUpper(m.animeTypeFilter)
		} else if m.isTVShow {
			modeStr = "TV SHOW"
		}

		rawName := fmt.Sprintf("ENTER %s TITLE", modeStr)
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		
		contentLen := len([]rune(spacedName))
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedName := strings.Repeat(" ", leftPad) + spacedName + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│", paddedName)
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		boxBlock := fmt.Sprintf("%s\n%s\n%s", topBorder, headerText, bottomBorder)
		
		inputStr := fmt.Sprintf("      %s\033[5m_\033[0m", m.textInput.Value())

		return fmt.Sprintf("\n%s\n\n%s\n", boxBlock, inputStr)

	case StateMovieSelect:
		var items [][2]string
		for _, r := range m.cinemetaMovies {
			year := r.Year
			if year == "" {
				year = r.ReleaseInfo
			}
			if len(year) > 4 {
				year = year[:4]
			}
			items = append(items, [2]string{r.Name, year})
		}
		if len(items) == 0 {
			return "\n  ❌ No database matches found on Cinemeta. Press [Esc] to quit."
		}
		return renderDatabaseMatchCamera("SELECT DATABASE MATCH", items, m.cursor, m.dbMatchSearch)

	// Anime Path Views
	case StateAnimeSelect:
		var items [][2]string
		for _, a := range m.animeList {
			yearStr := "N/A"
			if a.Year > 0 {
				yearStr = fmt.Sprintf("%d", a.Year)
			}
			details := fmt.Sprintf("%s  •  %d Eps", yearStr, a.Episodes)
			items = append(items, [2]string{fmt.Sprintf("[%s] %s", a.Type, a.Title), details})
		}
		if len(items) == 0 {
			return "\n  ❌ No database matches found on Jikan. Press [Esc] to quit."
		}
		return renderDatabaseMatchCamera("SELECT DATABASE MATCH", items, m.cursor, m.dbMatchSearch)
	case StateAnikotoShowSelect:
		rawName := "SELECT PROVIDER RESULT :"
		if len(m.dbMatchSearch) > 0 {
			rawName += " " + m.dbMatchSearch
		}
		
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		
		contentLen := len([]rune(spacedName)) + 2 // +2 for the blinking " _"
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedName := strings.Repeat(" ", leftPad) + spacedName + " \033[5m_\033[0m" + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│   %d/%d", paddedName, m.cursor+1, len(m.anikotoShows))
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		boxBlock := fmt.Sprintf("\n%s\n%s\n%s\n\n\n", topBorder, headerText, bottomBorder)
		
		maxGlobalRows := termHeight - 8
		if maxGlobalRows < 5 {
			maxGlobalRows = 5
		}

		start := m.cursor - (maxGlobalRows / 2)
		if start < 0 {
			start = 0
		}
		end := start + maxGlobalRows
		if end > len(m.anikotoShows) {
			end = len(m.anikotoShows)
			start = end - maxGlobalRows
			if start < 0 {
				start = 0
			}
		}

		var optsUI string
		for i := start; i < end; i++ {
			s := m.anikotoShows[i]
			title := fmt.Sprintf("%d. %s", i+1, s.Title)
			if i == m.cursor {
				optsUI += fmt.Sprintf("      \033[47;30m • %s \033[0m\n", title)
			} else {
				optsUI += fmt.Sprintf("         %s \n", title)
			}
		}
		return boxBlock + optsUI
	case StateAnikotoEpSelect:
		rawName := "SELECT EPISODE NUMBER :"
		if len(m.dbMatchSearch) > 0 {
			rawName += " " + m.dbMatchSearch
		}
		
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		
		contentLen := len([]rune(spacedName)) + 2 // +2 for the blinking " _"
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedName := strings.Repeat(" ", leftPad) + spacedName + " \033[5m_\033[0m" + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│   %d/%d", paddedName, m.cursor+1, len(m.anikotoEpisodes))
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		boxBlock := fmt.Sprintf("\n%s\n%s\n%s\n\n\n", topBorder, headerText, bottomBorder)
		
		maxGlobalRows := termHeight - 8
		if maxGlobalRows < 5 {
			maxGlobalRows = 5
		}

		start := m.cursor - (maxGlobalRows / 2)
		if start < 0 {
			start = 0
		}
		end := start + maxGlobalRows
		if end > len(m.anikotoEpisodes) {
			end = len(m.anikotoEpisodes)
			start = end - maxGlobalRows
			if start < 0 {
				start = 0
			}
		}

		var optsUI string
		for i := start; i < end; i++ {
			ep := m.anikotoEpisodes[i]
			title := fmt.Sprintf("%d. Episode %s", i+1, ep.Num)
			if i == m.cursor {
				optsUI += fmt.Sprintf("      \033[47;30m • %s \033[0m\n", title)
			} else {
				optsUI += fmt.Sprintf("         %s \n", title)
			}
		}
		return boxBlock + optsUI
	case StateAnikotoModeSelect:
		rawName := "SELECT AUDIO TRACK 🗣️"
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		
		contentLen := len([]rune(spacedName))
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedName := strings.Repeat(" ", leftPad) + spacedName + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│", paddedName)
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		boxBlock := fmt.Sprintf("\n%s\n%s\n%s\n\n\n", topBorder, headerText, bottomBorder)
		
		opts := []string{"Japanese (with subtitles)", "English (dubbed)"}
		var optsUI string
		for i, opt := range opts {
			if i == m.cursor {
				optsUI += fmt.Sprintf("      \033[47;30m • %s \033[0m\n", opt)
			} else {
				optsUI += fmt.Sprintf("         %s \n", opt)
			}
		}
		return boxBlock + optsUI

	case StateAsianShowSelect:
		rawName := "SELECT DRAMA RESULT :"
		if len(m.dbMatchSearch) > 0 {
			rawName += " " + m.dbMatchSearch
		}
		
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		
		contentLen := len([]rune(spacedName)) + 2
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedName := strings.Repeat(" ", leftPad) + spacedName + " \033[5m_\033[0m" + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│   %d/%d", paddedName, m.cursor+1, len(m.asianShows))
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		boxBlock := fmt.Sprintf("\n%s\n%s\n%s\n\n\n", topBorder, headerText, bottomBorder)
		
		maxGlobalRows := termHeight - 8
		if maxGlobalRows < 5 {
			maxGlobalRows = 5
		}

		start := m.cursor - (maxGlobalRows / 2)
		if start < 0 {
			start = 0
		}
		end := start + maxGlobalRows
		if end > len(m.asianShows) {
			end = len(m.asianShows)
			start = end - maxGlobalRows
			if start < 0 {
				start = 0
			}
		}

		var optsUI string
		for i := start; i < end; i++ {
			s := m.asianShows[i]
			title := fmt.Sprintf("%d. %s (%s)", i+1, s.Title, strings.ToUpper(s.Type))
			if i == m.cursor {
				optsUI += fmt.Sprintf("      \033[47;30m • %s \033[0m\n", title)
			} else {
				optsUI += fmt.Sprintf("         %s \n", title)
			}
		}
		return boxBlock + optsUI

	case StateAsianEpSelect:
		rawName := "SELECT EPISODE NUMBER :"
		if len(m.dbMatchSearch) > 0 {
			rawName += " " + m.dbMatchSearch
		}
		
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		
		contentLen := len([]rune(spacedName)) + 2
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedName := strings.Repeat(" ", leftPad) + spacedName + " \033[5m_\033[0m" + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│   %d/%d", paddedName, m.cursor+1, len(m.asianEpisodes))
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		boxBlock := fmt.Sprintf("\n%s\n%s\n%s\n\n\n", topBorder, headerText, bottomBorder)
		
		maxGlobalRows := termHeight - 8
		if maxGlobalRows < 5 {
			maxGlobalRows = 5
		}

		start := m.cursor - (maxGlobalRows / 2)
		if start < 0 {
			start = 0
		}
		end := start + maxGlobalRows
		if end > len(m.asianEpisodes) {
			end = len(m.asianEpisodes)
			start = end - maxGlobalRows
			if start < 0 {
				start = 0
			}
		}

		var optsUI string
		for i := start; i < end; i++ {
			ep := m.asianEpisodes[i]
			title := fmt.Sprintf("%d. Episode %s", i+1, ep.Title)
			if i == m.cursor {
				optsUI += fmt.Sprintf("      \033[47;30m • %s \033[0m\n", title)
			} else {
				optsUI += fmt.Sprintf("         %s \n", title)
			}
		}
		return boxBlock + optsUI

	// Western Path Views
	case StateTVShowSelect:
		var items [][2]string
		for _, show := range m.tvShows {
			year := show.Premiered
			if len(year) >= 4 {
				year = year[:4]
			}
			items = append(items, [2]string{show.Name, year})
		}
		return renderDatabaseMatchCamera("SELECT DATABASE MATCH", items, m.cursor, m.dbMatchSearch)
	case StateTVSeasonSelect:
		var opts []string
		for _, season := range m.tvSeasons {
			opts = append(opts, fmt.Sprintf("Season %d", season.Number))
		}
		return renderMenuCamera(opts, m.cursor, termHeight, fmt.Sprintf("📺 %s - Choose Target Season Pack:", m.selectedShow))
	case StateQuality:
		modeStr := "MOVIE"
		if m.isTVShow {
			modeStr = "TV SHOW"
		}
		title := fmt.Sprintf("SELECT %s QUALITY", modeStr)
		
		rawTitle := strings.ToUpper(title)
		spacedTitle := strings.Join(strings.Split(rawTitle, ""), " ")
		
		contentLen := len([]rune(spacedTitle))
		boxWidth := contentLen + 4
		if boxWidth < 36 {
			boxWidth = 36
		}
		padAmt := boxWidth - contentLen
		leftPad := padAmt / 2
		rightPad := padAmt - leftPad
		
		paddedTitle := strings.Repeat(" ", leftPad) + spacedTitle + strings.Repeat(" ", rightPad)
		
		topBorder := "    ┌" + strings.Repeat("─", boxWidth) + "┐"
		headerText := fmt.Sprintf("    │%s│", paddedTitle)
		bottomBorder := "    └" + strings.Repeat("─", boxWidth) + "┘"
		
		s := fmt.Sprintf("\n%s\n%s\n%s\n\n", topBorder, headerText, bottomBorder)

		for i, opt := range config.QualityOptions {
			isTarget := (i == m.cursor)
			displayRow := fmt.Sprintf(" %s ", opt)

			if isTarget {
				s += fmt.Sprintf("  \033[30;47m  \033[5m●\033[0;30;47m %s  \033[0m\n", displayRow)
			} else {
				s += fmt.Sprintf("      %s\n", displayRow)
			}
		}
		return s
	case StateTVFileSelect:
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}

		// Cinematic Header
		rawTitle := strings.ToUpper(m.currentQuery)
		spaced := strings.Join(strings.Split(rawTitle, ""), " ")
		bigTitle := strings.ReplaceAll(spaced, "   ", "     ")

		dotted := centerStr(strings.Repeat("-", len(bigTitle)+8), width)
		paddedTitle := centerStr(bigTitle, width)
		coloredTitle := strings.Replace(paddedTitle, bigTitle, fmt.Sprintf("\033[1;37m%s\033[0m", bigTitle), 1)

		rawQual := m.qualityFilter
		if rawQual == "All" {
			rawQual = "ANY QUALITY"
		} else {
			rawQual = strings.ToUpper(rawQual)
		}
		paddedQual := centerStr(rawQual, width)

		var s string
		s += fmt.Sprintf("\n%s\n%s\n%s\n%s\n\n", dotted, coloredTitle, dotted, paddedQual)
		
		rawName := "EPISODE NO: " + m.tvFileSearch
		spacedName := strings.Join(strings.Split(rawName, ""), " ")
		visibleLen := len([]rune(spacedName)) + 2 // +2 for " _"
		padAmtHeader := 30 - visibleLen
		if padAmtHeader < 0 {
			padAmtHeader = 0
		}
		paddedName := spacedName + " \033[5m_\033[0m" + strings.Repeat(" ", padAmtHeader)

		topBorder := "    ┌" + strings.Repeat("─", 32) + "┐"
		headerText := fmt.Sprintf("    │ %s │", paddedName)
		bottomBorder := "    └" + strings.Repeat("─", 32) + "┘"

		currentLocal := 0
		if len(m.tvFiles) > 0 {
			currentLocal = m.cursor + 1
		}
		counterStr := ""
		if len(m.tvFiles) > 0 {
			counterStr = fmt.Sprintf("  |  %d/%d", currentLocal, len(m.tvFiles))
		}

		middleRow := headerText + strings.Repeat(" ", 32) + counterStr

		s += "\n" + topBorder + "\n" + middleRow + "\n" + bottomBorder + "\n\n"

		const maxLocalItems = 10
		start := m.cursor - (maxLocalItems / 2)
		if start < 0 {
			start = 0
		}
		end := start + maxLocalItems
		if end > len(m.tvFiles) {
			end = len(m.tvFiles)
			start = end - maxLocalItems
			if start < 0 {
				start = 0
			}
		}

		for i := start; i < end; i++ {
			isTarget := (i == m.cursor)
			fileText := m.tvFiles[i]

			fields := strings.Fields(fileText)
			if len(fields) > 1 && len(fields[0]) > 0 && fields[0][0] >= '0' && fields[0][0] <= '9' {
				fileText = strings.TrimPrefix(fileText, fields[0])
				fileText = strings.TrimSpace(fileText)
			}

			if len([]rune(fileText)) > 60 {
				fileText = string([]rune(fileText)[:57]) + "..."
			} else {
				padAmtTitle := 60 - len([]rune(fileText))
				fileText = fileText + strings.Repeat(" ", padAmtTitle)
			}
			
			displayRow := fileText

			if isTarget {
				s += fmt.Sprintf("\033[30;47m  \033[5m●\033[0;30;47m %s  \033[0m\n", displayRow)
			} else {
				s += fmt.Sprintf("    %s\n", displayRow)
			}
		}

		return s

	case StateLoading:
		spinner := m.loadingSpinner.View()
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}
		height := m.terminalHeight
		if height == 0 {
			height = 24
		}
		purpleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

		content := lipgloss.JoinVertical(lipgloss.Center, spinner, purpleStyle.Render(m.loadingPhrase))
		return lipgloss.Place(width, height, lipgloss.Center, lipgloss.Center, content)

	case StateList:
		width := m.terminalWidth
		if width == 0 {
			width = 80 // fallback
		}

		// 1. Cinematic Header
		rawTitle := strings.ToUpper(m.currentQuery)
		spaced := strings.Join(strings.Split(rawTitle, ""), " ")
		bigTitle := strings.ReplaceAll(spaced, "   ", "     ")

		dotted := centerStr(strings.Repeat("-", len(bigTitle)+8), width)
		paddedTitle := centerStr(bigTitle, width)
		coloredTitle := strings.Replace(paddedTitle, bigTitle, fmt.Sprintf("\033[1;37m%s\033[0m", bigTitle), 1)

		rawQual := m.qualityFilter
		if rawQual == "All" {
			rawQual = "ANY QUALITY"
		} else {
			rawQual = strings.ToUpper(rawQual)
		}
		paddedQual := centerStr(rawQual, width)

		s := fmt.Sprintf("\n%s\n%s\n%s\n%s\n\n", dotted, coloredTitle, dotted, paddedQual)

		var rows []renderRow
		globalMovieCounter := 0
		for _, g := range m.groups {
			// Gatekeeper: Hide dead providers and zero-result pulls
			if g.Status == "No Matches" || (g.Status == "Complete" && len(g.Results) == 0) {
				continue
			}

			// Clean Header Format: ┌───┐ Provider box
			rawName := strings.ToUpper(g.Name)
			spacedName := strings.Join(strings.Split(rawName, ""), " ")
			if len([]rune(spacedName)) > 30 {
				spacedName = string([]rune(spacedName)[:27]) + "..."
			}
			padAmtHeader := 30 - len([]rune(spacedName))
			if padAmtHeader < 0 {
				padAmtHeader = 0
			}
			paddedName := spacedName + strings.Repeat(" ", padAmtHeader)

			topBorder := "    ┌" + strings.Repeat("─", 32) + "┐"
			headerText := fmt.Sprintf("    │ %s │", paddedName)
			bottomBorder := "    └" + strings.Repeat("─", 32) + "┘"

			const maxLocalItems = 6
			localCursor := -1
			if m.cursor >= globalMovieCounter && m.cursor < globalMovieCounter+len(g.Results) {
				localCursor = m.cursor - globalMovieCounter
			}

			currentLocal := 0
			if localCursor != -1 {
				currentLocal = localCursor + 1
			}

			var counterStr string
			if g.Status == "Loading..." {
				counterStr = "  |  ⏳"
			} else if len(g.Results) > 0 {
				counterStr = fmt.Sprintf("  |  %d/%d", currentLocal, len(g.Results))
			}

			// Header string is 38 visual columns. Target `|` is at 73.
			// 38 + 32 spaces = 70. 70 + `  |` = 73.
			middleRow := headerText + strings.Repeat(" ", 32) + counterStr

			rows = append(rows, renderRow{text: topBorder})
			rows = append(rows, renderRow{text: middleRow})
			rows = append(rows, renderRow{text: bottomBorder})

			if len(g.Results) == 0 {
				rows = append(rows, renderRow{text: "     ⏳ Scanning peer tables..."})
				rows = append(rows, renderRow{text: ""})
				continue
			}

			start := 0
			end := len(g.Results)
			if end > maxLocalItems {
				if localCursor != -1 {
					start = localCursor - (maxLocalItems / 2)
					if start < 0 {
						start = 0
					}
					end = start + maxLocalItems
					if end > len(g.Results) {
						end = len(g.Results)
						start = end - maxLocalItems
					}
				} else {
					end = maxLocalItems
				}
			}

			for i := start; i < end; i++ {
				res := g.Results[i]
				isTarget := (globalMovieCounter+i == m.cursor)
				sizeGB := float64(res.Size) / (1024 * 1024 * 1024)

				// Strict Title Padding (28 chars)
				titleStr := m.currentQuery
				if m.isTVShow {
					titleStr = fmt.Sprintf("%s S%02d", m.selectedShow, m.selectedSeason)
				}
				if len([]rune(titleStr)) > 28 {
					titleStr = string([]rune(titleStr)[:25]) + "..."
				}
				padAmtTitle := 28 - len([]rune(titleStr))
				if padAmtTitle < 0 {
					padAmtTitle = 0
				}
				titleCol := titleStr + strings.Repeat(" ", padAmtTitle)

				// Strict Seeder Padding (16 visual columns)
				seederText := fmt.Sprintf("🟢 %d Seeders", res.Seeders)
				vLenSeeder := len([]rune(seederText)) + 1
				padAmtSeeder := 16 - vLenSeeder
				if padAmtSeeder < 0 {
					padAmtSeeder = 0
				}
				seederCol := seederText + strings.Repeat(" ", padAmtSeeder)

				// Strict Size Padding (12 visual columns)
				sizeText := fmt.Sprintf("💾 %.2f GB", sizeGB)
				vLenSize := len([]rune(sizeText)) + 1
				padAmtSize := 12 - vLenSize
				if padAmtSize < 0 {
					padAmtSize = 0
				}
				sizeCol := sizeText + strings.Repeat(" ", padAmtSize)

				// Format extraction (Strict 8 visual columns)
				format := "Unknown"
				icon := "💿"
				t := strings.ToLower(res.Title)
				if strings.Contains(t, "bluray") || strings.Contains(t, "blu-ray") || strings.Contains(t, "bdrip") || strings.Contains(t, "brrip") {
					format = "BluRay"
					icon = "📀"
				} else if strings.Contains(t, "web-dl") || strings.Contains(t, "webdl") {
					format = "WEB-DL"
					icon = "📀"
				} else if strings.Contains(t, "webrip") || strings.Contains(t, "web") {
					format = "WEBRip"
					icon = "📀"
				} else if strings.Contains(t, "hdrip") {
					format = "HDRip"
					icon = "📀"
				} else if strings.Contains(t, "hdtv") {
					format = "HDTV"
					icon = "📀"
				} else if strings.Contains(t, "dvd") {
					format = "DVD"
					icon = "📀"
				} else if strings.Contains(t, "cam") || strings.Contains(t, "ts") || strings.Contains(t, "telesync") {
					format = "CAM/TS"
					icon = "📀"
				}

				padAmtFormat := 8 - len([]rune(format))
				if padAmtFormat < 0 {
					padAmtFormat = 0
				}
				formatStr := format + strings.Repeat(" ", padAmtFormat)
				formatCol := fmt.Sprintf("%s %s", icon, formatStr)

				// Build perfectly aligned row
				leftPart := fmt.Sprintf("%s  |  %s  |  %s", titleCol, seederCol, sizeCol)

				var displayRow string
				if m.isTVShow {
					displayRow = fmt.Sprintf("%s  |  %s  |  ✅ COMPLETE", leftPart, formatCol)
				} else {
					displayRow = fmt.Sprintf("%s  |  %s", leftPart, formatCol)
				}

				// Apply highlight and blinking dot
				if isTarget {
					finalText := fmt.Sprintf("\033[30;47m  \033[5m●\033[0;30;47m %s  \033[0m", displayRow)
					rows = append(rows, renderRow{isTarget: true, text: finalText})
				} else {
					rows = append(rows, renderRow{isTarget: false, text: fmt.Sprintf("    %s", displayRow)})
				}
			}

			globalMovieCounter += len(g.Results)
			rows = append(rows, renderRow{text: ""})
		}

		if m.totalMovies == 0 && m.pendingSearch == 0 {
			s += "     ❌ No unmasked swarms found.\n"
			return s
		}

		maxGlobalRows := termHeight - 8
		if maxGlobalRows < 5 {
			maxGlobalRows = 5
		}

		targetRowIdx := 0
		for i, r := range rows {
			if r.isTarget {
				targetRowIdx = i
				break
			}
		}

		startGlobal := targetRowIdx - (maxGlobalRows / 2)
		if startGlobal < 0 {
			startGlobal = 0
		}
		endGlobal := startGlobal + maxGlobalRows
		if endGlobal > len(rows) {
			endGlobal = len(rows)
			startGlobal = endGlobal - maxGlobalRows
			if startGlobal < 0 {
				startGlobal = 0
			}
		}

		for i := startGlobal; i < endGlobal; i++ {
			s += rows[i].text + "\n"
		}

		return s
	}
	return ""
}
