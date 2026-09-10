package bittorrent

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"

	"bubble-stream/internal/config"
	"bubble-stream/internal/sysutil"
	g "github.com/anacrolix/generics"
	analog "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
)

func init() {
	analog.DefaultHandler = analog.DiscardHandler
	analog.Default.SetHandlers(analog.DiscardHandler)
}

// Supplemental high-speed public trackers to maximize peer discovery
var supplementalTrackers = []string{
	"udp://explodie.org:6969/announce",
	"udp://tracker.moeking.me:6969/announce",
	"udp://p4p.arenabg.com:1337/announce",
	"udp://movies.zsw.ca:6969/announce",
	"http://tracker.openbittorrent.com:80/announce",
}

// AllTrackers returns the combined set of default and supplemental high-speed trackers.
func AllTrackers() []string {
	res := make([]string, 0, len(config.DefaultTrackers)+len(supplementalTrackers))
	res = append(res, config.DefaultTrackers...)
	res = append(res, supplementalTrackers...)
	return res
}

// Engine wraps the anacrolix/torrent client with clean lifecycle management.
type Engine struct {
	client       *torrent.Client
	defaultStore storage.ClientImplCloser
	dataDir      string
	mu           sync.Mutex
}

// NewEngine initializes a new high-throughput torrent engine with DHT, PEX, and multi-tracker support.
func NewEngine() (*Engine, error) {
	tempDir, err := os.MkdirTemp("", "goovie_torrent_engine_*")
	if err != nil {
		return nil, fmt.Errorf("failed to create engine temp dir: %w", err)
	}
	sysutil.RegisterTempDir(tempDir)

	analog.DefaultHandler = analog.DiscardHandler
	analog.Default.SetHandlers(analog.DiscardHandler)

	cfg := torrent.NewDefaultClientConfig()
	cfg.DataDir = tempDir
	cfg.ListenPort = 0 // Ephemeral port to prevent port conflicts and ensure clean close

	discardLogger := analog.NewLogger()
	discardLogger.SetHandlers(analog.DiscardHandler)
	cfg.Logger = discardLogger
	cfg.Slogger = slog.New(slog.NewTextHandler(io.Discard, nil))

	fileStore := storage.NewFileOpts(storage.NewFileClientOpts{
		ClientBaseDir:   tempDir,
		UsePartFiles:    g.Some(false),
		PieceCompletion: storage.NewMapPieceCompletion(),
	})
	cfg.DefaultStorage = fileStore

	cfg.NoUpload = false
	cfg.DisableAggressiveUpload = true
	cfg.Seed = false
	cfg.EstablishedConnsPerTorrent = 150
	cfg.HalfOpenConnsPerTorrent = 60
	cfg.TotalHalfOpenConns = 120
	cfg.TorrentPeersHighWater = 1000
	cfg.TorrentPeersLowWater = 100
	cfg.DialForPeerConns = true
	cfg.AcceptPeerConnections = true
	cfg.DisableUTP = false
	cfg.DisableTCP = false
	cfg.NoDHT = false
	cfg.DisableTrackers = false
	cfg.DisablePEX = false
	cfg.DisableWebtorrent = false
	cfg.DisableWebseeds = false
	cfg.DropDuplicatePeerIds = true

	cl, err := torrent.NewClient(cfg)
	if err != nil {
		_ = fileStore.Close()
		sysutil.RemoveTempDir(tempDir)
		return nil, fmt.Errorf("failed to initialize torrent client: %w", err)
	}

	return &Engine{
		client:       cl,
		defaultStore: fileStore,
		dataDir:      tempDir,
	}, nil
}

// DataDir returns the temporary storage directory for this engine.
func (e *Engine) DataDir() string {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.dataDir
}

// Client returns the underlying anacrolix torrent client.
func (e *Engine) Client() *torrent.Client {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.client
}

// Close gracefully terminates all torrent activity and purges engine data.
func (e *Engine) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.client != nil {
		e.client.Close()
		e.client = nil
	}
	if e.defaultStore != nil {
		_ = e.defaultStore.Close()
		e.defaultStore = nil
	}
	if e.dataDir != "" {
		sysutil.RemoveTempDir(e.dataDir)
		e.dataDir = ""
	}
}

var (
	globalEngine   *Engine
	globalEngineMu sync.Mutex
)

// GetGlobalEngine returns the singleton Engine instance, initializing it if necessary.
func GetGlobalEngine() (*Engine, error) {
	globalEngineMu.Lock()
	defer globalEngineMu.Unlock()
	if globalEngine == nil || globalEngine.Client() == nil {
		eng, err := NewEngine()
		if err != nil {
			return nil, err
		}
		globalEngine = eng
	}
	return globalEngine, nil
}

// CloseGlobalEngine shuts down the singleton engine if running.
func CloseGlobalEngine() {
	globalEngineMu.Lock()
	defer globalEngineMu.Unlock()
	if globalEngine != nil {
		globalEngine.Close()
		globalEngine = nil
	}
}

