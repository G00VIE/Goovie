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
	case StateFrontPage:
		width := m.terminalWidth
		if width == 0 {
			width = 80
		}

		if width < 30 || termHeight < 15 {
			return lipgloss.Place(width, termHeight, lipgloss.Center, lipgloss.Center, errorStyle.Render("Terminal too small. Please enlarge."))
		}
		
		subtitleStyle := lipgloss.NewStyle().Align(lipgloss.Center)
		subtitle := subtitleStyle.Render("[ ENTER ] to continue")
		
		finalUI := lipgloss.JoinVertical(lipgloss.Center,
			m.cachedFrontTitle,
			"\n\n",
			subtitle,
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
	case StateAnimeTypeSelect:
		return renderMenuCamera(config.AnimeTypeOptions, m.cursor, termHeight, "🗡️ Anime Pipeline: Select Sub-Type Filter:")
	case StateSearch:
		modeStr := "Movie"
		if m.isTVShow {
			modeStr = "TV Show"
		} else if m.isAnime {
			modeStr = "Anime"
		}
		return fmt.Sprintf("\n  🍿 GOOVIE CLI v25.0\n\n  Enter %s Title: \n  %s\n", modeStr, m.textInput.View())

	case StateMovieSelect:
		var opts []string
		for _, r := range m.cinemetaMovies {
			year := r.Year
			if len(year) > 4 {
				year = year[:4]
			}
			opts = append(opts, fmt.Sprintf("%s (%s)", r.Name, year))
		}
		if len(opts) == 0 {
			return "\n  ❌ No database matches found on Cinemeta. Press [Esc] to quit."
		}
		return renderMenuCamera(opts, m.cursor, termHeight, "🎬 Select Unified Database Match (Cinemeta):")

	// Anime Path Views
	case StateAnimeSelect:
		var opts []string
		for _, a := range m.animeList {
			yearStr := "N/A"
			if a.Year > 0 {
				yearStr = fmt.Sprintf("%d", a.Year)
			}
			opts = append(opts, fmt.Sprintf("[%s] %s (%s) • %d Eps", a.Type, a.Title, yearStr, a.Episodes))
		}
		if len(opts) == 0 {
			return "\n  ❌ No database matches found on Jikan. Press [Esc] to quit."
		}
		return renderMenuCamera(opts, m.cursor, termHeight, "🗡️ Select Unified Database Match (Jikan):")
	case StateAnikotoShowSelect:
		var opts []string
		for _, s := range m.anikotoShows {
			opts = append(opts, s.Title)
		}
		return renderMenuCamera(opts, m.cursor, termHeight, "🗡️ Select Provider Match (Anikoto):")
	case StateAnikotoEpSelect:
		var opts []string
		for _, ep := range m.anikotoEpisodes {
			opts = append(opts, fmt.Sprintf("Episode %s", ep.Num))
		}
		return renderMenuCamera(opts, m.cursor, termHeight, "🗡️ Select Target Episode:")
	case StateAnikotoModeSelect:
		return renderMenuCamera([]string{"1. Japanese (Subtitles)", "2. English (Dubbed)"}, m.cursor, termHeight, "🗡️ Select Audio Track Mode:")
	case StateAnikotoServerSelect:
		var opts []string
		for _, s := range m.anikotoServers {
			opts = append(opts, fmt.Sprintf("Server: %s", s.Name))
		}
		return renderMenuCamera(opts, m.cursor, termHeight, "🗡️ Select Streaming Relay Server:")

	// Western Path Views
	case StateTVShowSelect:
		var opts []string
		for _, show := range m.tvShows {
			year := show.Premiered
			if len(year) >= 4 {
				year = year[:4]
			}
			opts = append(opts, fmt.Sprintf("%s (%s)", show.Name, year))
		}
		return renderMenuCamera(opts, m.cursor, termHeight, "📺 Select Unified Database Match:")
	case StateTVSeasonSelect:
		var opts []string
		for _, season := range m.tvSeasons {
			opts = append(opts, fmt.Sprintf("Season %d", season.Number))
		}
		return renderMenuCamera(opts, m.cursor, termHeight, fmt.Sprintf("📺 %s - Choose Target Season Pack:", m.selectedShow))
	case StateQuality:
		return renderMenuCamera(config.QualityOptions, m.cursor, termHeight, fmt.Sprintf("🍿 Processing Matrix: \"%s\"\n\n  Select Sizing Profile Filter:", m.currentQuery))
	case StateTVFileSelect:
		return renderMenuCamera(m.tvFiles, m.cursor, termHeight, "📺 EXTRACTION COMPLETE - Select Target Episode:")

	case StateLoading:
		if m.isAnime {
			return "\n  📡 Penetrating Anikoto API Relays & Scraping Media IDs...\n"
		} else if m.isTVShow && m.selectedMagnet != "" {
			return "\n  📡 Connecting to Swarm & Extracting Episode Metadata...\n"
		}
		return "\n  📡 Querying Index Repositories & Populating Channels...\n"

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
