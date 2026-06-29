package prowlarr

type ProwlarrResult struct {
	Title       string `json:"title"`
	MagnetUri   string `json:"magnetUrl"`
	InfoHash    string `json:"infoHash"`
	Seeders     int    `json:"seeders"`
	Peers       int    `json:"peers"`
	Size        int64  `json:"size"`
	Indexer     string `json:"indexer"`
	DownloadUrl string `json:"downloadUrl"`
}

type Indexer struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type IndexerGroup struct {
	ID      int
	Name    string
	Status  string
	Results []ProwlarrResult
	Order   int
}

// --- Western Models ---
type TVMazeShow struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Premiered string `json:"premiered"`
}

type TVMazeSearchResult struct {
	Show TVMazeShow `json:"show"`
}

type TVMazeSeason struct {
	ID     int `json:"id"`
	Number int `json:"number"`
}

type TVMazeEpisode struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Number int    `json:"number"`
}

type CinemetaMovie struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Year        string `json:"year"`
	ReleaseInfo string `json:"releaseInfo"`
}

type CinemetaCatalog struct {
	Metas []CinemetaMovie `json:"metas"`
}

// --- Anime Models ---
type JikanAnime struct {
	MalID    int    `json:"mal_id"`
	Title    string `json:"title"`
	Type     string `json:"type"`
	Year     int    `json:"year"`
	Episodes int    `json:"episodes"`
}

type JikanResponse struct {
	Data []JikanAnime `json:"data"`
}
