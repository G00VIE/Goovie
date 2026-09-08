package player

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"bubble-stream/internal/bittorrent"
	"bubble-stream/internal/config"
	tea "github.com/charmbracelet/bubbletea"
)

var proxySeq uint64
var GlobalJar, _ = cookiejar.New(nil)
var GlobalProxy *VibeProxy

var pngIENDMarker = []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82}

type VibeSegment struct{ URL string }
type VibeSession struct {
	MasterURL   string
	Referer     string
	Variants    map[string][]VibeSegment
	VariantURLs map[string]string
	Keys        map[string]string
	Mu          sync.Mutex
}
type VibeProxy struct {
	BaseURL  string
	Sessions map[string]*VibeSession
	Mu       sync.Mutex
	server   *http.Server
	listener net.Listener
}

type PlayerFinishedMsg struct {
	Err error
}

func InitProxy() {
	GlobalProxy = &VibeProxy{Sessions: make(map[string]*VibeSession)}
	mux := http.NewServeMux()
	mux.HandleFunc("/stream/", GlobalProxy.handle)
	server := &http.Server{Handler: mux}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		GlobalProxy.server = server
		GlobalProxy.listener = listener
		GlobalProxy.BaseURL = "http://" + listener.Addr().String()
		go server.Serve(listener)
	}
}

// CloseProxy terminates the anime proxy server and closes all listening ports.
func CloseProxy() {
	if GlobalProxy != nil {
		GlobalProxy.Mu.Lock()
		defer GlobalProxy.Mu.Unlock()
		if GlobalProxy.server != nil {
			GlobalProxy.server.SetKeepAlivesEnabled(false)
			_ = GlobalProxy.server.Close()
			GlobalProxy.server = nil
		}
		if GlobalProxy.listener != nil {
			_ = GlobalProxy.listener.Close()
			GlobalProxy.listener = nil
		}
	}
}

func fetchHTTPWithReferer(client *http.Client, reqURL string, referer string) ([]byte, error) {
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", config.UserAgent)
	if referer != "" {
		req.Header.Set("Referer", referer)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func (p *VibeProxy) Register(masterURL, referer string) string {
	seq := atomic.AddUint64(&proxySeq, 1)
	id := fmt.Sprintf("%s-%x", strconv.FormatInt(time.Now().UnixNano(), 36), seq)
	p.Mu.Lock()
	p.Sessions[id] = &VibeSession{
		MasterURL:   masterURL,
		Referer:     referer,
		Variants:    make(map[string][]VibeSegment),
		VariantURLs: make(map[string]string),
		Keys:        make(map[string]string),
	}
	p.Mu.Unlock()
	return p.BaseURL + "/stream/" + id + "/master.m3u8"
}

func (p *VibeProxy) handle(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/stream/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.NotFound(w, r)
		return
	}
	sessionID := parts[0]

	p.Mu.Lock()
	session, ok := p.Sessions[sessionID]
	p.Mu.Unlock()
	if !ok {
		http.NotFound(w, r)
		return
	}

	switch {
	case len(parts) == 2 && parts[1] == "master.m3u8":
		p.serveMaster(w, sessionID, session)
	case len(parts) == 2 && strings.HasSuffix(parts[1], ".m3u8"):
		p.serveVariant(w, sessionID, session, parts[1])
	case len(parts) == 4 && parts[2] == "seg":
		p.serveSegment(w, r, session, parts[1], parts[3])
	case len(parts) == 4 && parts[2] == "key":
		p.serveKey(w, r, session, parts[1], parts[3])
	default:
		http.NotFound(w, r)
	}
}

func resolvePlaylistURL(baseURL, entry string) string {
	base, err := url.Parse(baseURL)
	if err != nil {
		return entry
	}
	if entry == "" {
		if idx := strings.LastIndex(base.Path, "/"); idx >= 0 {
			base.Path = base.Path[:idx+1]
		}
		base.RawQuery = ""
		base.Fragment = ""
		return base.String()
	}
	ref, err := url.Parse(entry)
	if err != nil {
		return entry
	}
	return base.ResolveReference(ref).String()
}

func rewritePlaylistLines(playlist string, mapLine func(string) string) string {
	lines := strings.Split(strings.ReplaceAll(playlist, "\r\n", "\n"), "\n")
	for i, line := range lines {
		lines[i] = mapLine(line)
	}
	return strings.Join(lines, "\n")
}

func stripPNGWrapper(data []byte) []byte {
	idx := bytes.Index(data, pngIENDMarker)
	if idx < 0 {
		return data
	}
	return data[idx+len(pngIENDMarker):]
}

func writePlaylist(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = io.WriteString(w, body)
}

func (p *VibeProxy) serveMaster(w http.ResponseWriter, sessionID string, session *VibeSession) {
	client := &http.Client{Timeout: 10 * time.Second}
	bodyBytes, err := fetchHTTPWithReferer(client, session.MasterURL, session.Referer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	body := string(bodyBytes)

	if strings.Contains(body, "#EXTINF:") || strings.Contains(body, "#EXT-X-TARGETDURATION:") {
		p.serveVariantContent(w, sessionID, session, "main", session.MasterURL, body)
		return
	}

	session.Mu.Lock()
	defer session.Mu.Unlock()

	varIdx := 0
	uriRE := regexp.MustCompile(`URI="([^"]+)"`)

	rewritten := rewritePlaylistLines(body, func(line string) string {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return line
		}
		if strings.HasPrefix(trimmed, "#") {
			if uriRE.MatchString(trimmed) {
				return uriRE.ReplaceAllStringFunc(trimmed, func(match string) string {
					sub := uriRE.FindStringSubmatch(match)
					if len(sub) < 2 {
						return match
					}
					resolved := resolvePlaylistURL(session.MasterURL, sub[1])
					vKey := fmt.Sprintf("v%d", varIdx)
					varIdx++
					session.VariantURLs[vKey] = resolved
					return fmt.Sprintf(`URI="%s/stream/%s/%s.m3u8"`, p.BaseURL, sessionID, vKey)
				})
			}
			return line
		}

		resolved := resolvePlaylistURL(session.MasterURL, trimmed)
		vKey := fmt.Sprintf("v%d", varIdx)
		varIdx++
		session.VariantURLs[vKey] = resolved
		return fmt.Sprintf("%s/stream/%s/%s.m3u8", p.BaseURL, sessionID, vKey)
	})
	writePlaylist(w, rewritten)
}

func (p *VibeProxy) serveVariant(w http.ResponseWriter, sessionID string, session *VibeSession, variantName string) {
	variantKey := strings.TrimSuffix(variantName, ".m3u8")

	session.Mu.Lock()
	variantURL, ok := session.VariantURLs[variantKey]
	session.Mu.Unlock()

	if !ok {
		variantURL = resolvePlaylistURL(session.MasterURL, variantName)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	bodyBytes, err := fetchHTTPWithReferer(client, variantURL, session.Referer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	body := string(bodyBytes)
	p.serveVariantContent(w, sessionID, session, variantKey, variantURL, body)
}

func (p *VibeProxy) serveVariantContent(w http.ResponseWriter, sessionID string, session *VibeSession, variantKey, variantURL, body string) {
	session.Mu.Lock()
	defer session.Mu.Unlock()

	var segments []VibeSegment
	keyIdx := 0
	uriRE := regexp.MustCompile(`URI="([^"]+)"`)

	rewritten := rewritePlaylistLines(body, func(line string) string {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			return line
		}
		if strings.HasPrefix(trimmed, "#") {
			if (strings.HasPrefix(trimmed, "#EXT-X-KEY") || strings.HasPrefix(trimmed, "#EXT-X-MAP")) && uriRE.MatchString(trimmed) {
				return uriRE.ReplaceAllStringFunc(trimmed, func(match string) string {
					sub := uriRE.FindStringSubmatch(match)
					if len(sub) < 2 {
						return match
					}
					resolved := resolvePlaylistURL(variantURL, sub[1])
					kID := fmt.Sprintf("%s-k%d", variantKey, keyIdx)
					session.Keys[kID] = resolved
					newURL := fmt.Sprintf(`%s/stream/%s/%s/key/%d`, p.BaseURL, sessionID, variantKey, keyIdx)
					keyIdx++
					return fmt.Sprintf(`URI="%s"`, newURL)
				})
			}
			return line
		}

		segmentURL := resolvePlaylistURL(variantURL, trimmed)
		index := len(segments)
		segments = append(segments, VibeSegment{URL: segmentURL})
		return fmt.Sprintf("%s/stream/%s/%s/seg/%d", p.BaseURL, sessionID, variantKey, index)
	})

	session.Variants[variantKey] = segments
	writePlaylist(w, rewritten)
}

func (p *VibeProxy) serveSegment(w http.ResponseWriter, r *http.Request, session *VibeSession, variantKey, indexText string) {
	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 {
		http.NotFound(w, r)
		return
	}

	session.Mu.Lock()
	segments, ok := session.Variants[variantKey]
	session.Mu.Unlock()

	if !ok || index >= len(segments) {
		http.NotFound(w, r)
		return
	}

	client := &http.Client{Timeout: 15 * time.Second}
	data, err := fetchHTTPWithReferer(client, segments[index].URL, session.Referer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	body := stripPNGWrapper(data)
	w.Header().Set("Content-Type", "video/mp2t")
	w.Header().Set("Content-Length", strconv.Itoa(len(body)))
	_, _ = w.Write(body)
}

func (p *VibeProxy) serveKey(w http.ResponseWriter, r *http.Request, session *VibeSession, variantKey, indexText string) {
	keyID := fmt.Sprintf("%s-k%s", variantKey, indexText)

	session.Mu.Lock()
	keyURL, ok := session.Keys[keyID]
	session.Mu.Unlock()

	if !ok {
		http.NotFound(w, r)
		return
	}

	client := &http.Client{Timeout: 10 * time.Second}
	data, err := fetchHTTPWithReferer(client, keyURL, session.Referer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	_, _ = w.Write(data)
}

func LaunchPlayer(target string, fileIndex string, referer string, subtitleURL string) tea.Cmd {
	if strings.HasPrefix(target, "http") {
		// Anime path uses pure mpv with optional referer and subtitle
		var args []string
		if referer != "" {
			args = append(args, "--referrer="+referer)
		}
		if subtitleURL != "" {
			args = append(args, "--sub-file="+subtitleURL)
		}
		args = append(args, target)
		c := exec.Command("mpv", args...)
		c.Stdout = io.Discard
		c.Stderr = io.Discard
		return func() tea.Msg {
			err := c.Run()
			return PlayerFinishedMsg{Err: err}
		}
	}

	// Torrent path: stream via pure Go bittorrent engine directly into MPV
	return func() tea.Msg {
		engine, err := bittorrent.GetGlobalEngine()
		if err != nil {
			return PlayerFinishedMsg{Err: fmt.Errorf("failed to start torrent engine: %w", err)}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		defer cancel()

		idx := -1
		if fileIndex != "" {
			idx, _ = strconv.Atoi(fileIndex)
		}

		session, err := engine.StartStreamServer(ctx, target, idx)
		if err != nil {
			return PlayerFinishedMsg{Err: fmt.Errorf("failed to start stream: %w", err)}
		}
		defer session.Close()

		c := exec.Command("mpv", session.StreamURL)
		c.Stdout = io.Discard
		c.Stderr = io.Discard
		err = c.Run()
		return PlayerFinishedMsg{Err: err}
	}
}
