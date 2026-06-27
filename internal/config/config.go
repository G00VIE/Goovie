package config

var (
	ProwlarrAPIKey = "3507472ebfd145ce9d9a71eaadf46eaa"
	ProwlarrURL    = "http://localhost:9696"
	MinimumSeeders = 5
	AnikotoBaseURL = "https://anikototv.to"
	UserAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"

	DefaultTrackers = []string{
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://open.stealth.si:80/announce",
		"udp://tracker.torrent.eu.org:451/announce",
	}

	QualityOptions   = []string{"All", "720p", "1080p", "4K"}
	AnimeTypeOptions = []string{"All", "TV", "Movie", "OVA", "Special"}
)
