package player

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"bubble-stream/internal/config"
	"bubble-stream/internal/prowlarr"
	tea "github.com/charmbracelet/bubbletea"
)

var GlobalJar, _ = cookiejar.New(nil)
var GlobalProxy *VibeProxy

var pngIENDMarker = []byte{0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82}

type VibeSegment struct{ URL string }
type VibeSession struct {
	MasterURL string
	Referer   string
	Variants  map[string][]VibeSegment
}
type VibeProxy struct {
	BaseURL  string
	Sessions map[string]*VibeSession
	Mu       sync.Mutex
}

func InitProxy() {
	GlobalProxy = &VibeProxy{Sessions: make(map[string]*VibeSession)}
	mux := http.NewServeMux()
	mux.HandleFunc("/stream/", GlobalProxy.handle)
	server := &http.Server{Handler: mux}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err == nil {
		GlobalProxy.BaseURL = "http://" + listener.Addr().String()
		go server.Serve(listener)
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
	id := strconv.FormatInt(time.Now().UnixNano(), 36)
	p.Mu.Lock()
	p.Sessions[id] = &VibeSession{
		MasterURL: masterURL,
		Referer:   referer,
		Variants:  make(map[string][]VibeSegment),
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
	default:
		http.NotFound(w, r)
	}
}

func resolvePlaylistURL(baseURL, entry string) string {
	if strings.HasPrefix(entry, "http://") || strings.HasPrefix(entry, "https://") {
		return entry
	}
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return entry
	}
	if strings.HasPrefix(entry, "/") {
		parsed.Path = entry
		parsed.RawQuery = ""
		parsed.Fragment = ""
		return parsed.Scheme + "://" + parsed.Host + entry
	}
	if idx := strings.LastIndex(parsed.Path, "/"); idx >= 0 {
		parsed.Path = parsed.Path[:idx+1] + entry
	} else {
		parsed.Path = "/" + entry
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
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
	rewritten := rewritePlaylistLines(body, func(line string) string {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			return line
		}
		resolvedURL := resolvePlaylistURL(session.MasterURL, line)
		parsed, err := url.Parse(resolvedURL)
		var name string
		if err == nil {
			parts := strings.Split(parsed.Path, "/")
			if len(parts) > 0 {
				name = parts[len(parts)-1]
			}
		}
		if name == "" {
			name = "variant.m3u8"
		}
		return p.BaseURL + "/stream/" + sessionID + "/" + name
	})
	writePlaylist(w, rewritten)
}

func (p *VibeProxy) serveVariant(w http.ResponseWriter, sessionID string, session *VibeSession, variantName string) {
	client := &http.Client{Timeout: 10 * time.Second}
	variantURL := resolvePlaylistURL(session.MasterURL, variantName)
	bodyBytes, err := fetchHTTPWithReferer(client, variantURL, session.Referer)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	body := string(bodyBytes)

	var segments []VibeSegment
	rewritten := rewritePlaylistLines(body, func(line string) string {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			return line
		}
		segmentURL := resolvePlaylistURL(variantURL, line)
		index := len(segments)
		segments = append(segments, VibeSegment{URL: segmentURL})
		return fmt.Sprintf("%s/stream/%s/%s/seg/%d", p.BaseURL, sessionID, variantName, index)
	})

	p.Mu.Lock()
	session.Variants[variantName] = segments
	p.Mu.Unlock()

	writePlaylist(w, rewritten)
}

func (p *VibeProxy) serveSegment(w http.ResponseWriter, r *http.Request, session *VibeSession, variantName, indexText string) {
	index, err := strconv.Atoi(indexText)
	if err != nil || index < 0 {
		http.NotFound(w, r)
		return
	}

	p.Mu.Lock()
	segments := session.Variants[variantName]
	p.Mu.Unlock()

	if index >= len(segments) {
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

func LaunchPlayer(target string, fileIndex string, referer string) tea.Cmd {
	var c *exec.Cmd
	if strings.HasPrefix(target, "http") {
		// Anime path uses pure mpv with optional referer
		if referer != "" {
			c = exec.Command("mpv", "--referrer="+referer, target)
		} else {
			c = exec.Command("mpv", target)
		}
	} else {
		// Western path uses webtorrent pipeline
		if runtime.GOOS == "windows" {
			c = exec.Command("cmd")
			var rawCmd string
			if fileIndex != "" {
				rawCmd = fmt.Sprintf(`cmd /c webtorrent.cmd "%s" --select %s --mpv`, target, fileIndex)
			} else {
				rawCmd = fmt.Sprintf(`cmd /c webtorrent.cmd "%s" --mpv`, target)
			}
			c.SysProcAttr = &syscall.SysProcAttr{CmdLine: rawCmd}
		} else {
			if fileIndex != "" {
				c = exec.Command("webtorrent", target, "--select", fileIndex, "--mpv")
			} else {
				c = exec.Command("webtorrent", target, "--mpv")
			}
		}
	}

	return tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return prowlarr.ErrMsg{Err: fmt.Errorf("player execution failed: %v", err)}
		}
		return nil
	})
}
