package tvmaze

type Show struct {
	ID        int        `json:"id"`
	URL       string     `json:"URL"`
	Name      string     `json:"name"`
	Language  string     `json:"language"`
	Genres    []string   `json:"genres"`
	Status    string     `json:"status"`
	Premiered *string    `json:"premiered"`
	Ended     *string    `json:"ended"`
	Network   Network    `json:"network"`
	Externals *Externals `json:"externals"`
	Image     *Image     `json:"image"`
	Summary   string     `json:"summary"`
	Updated   int        `json:"updated"`
}

type Network struct {
	ID           int     `json:"id"`
	Name         string  `json:"name"`
	Country      Country `json:"country"`
	OfficialSite *string `json:"officialSite"`
}

type Country struct {
	Name     string `json:"name"`
	Code     string `json:"code"`
	Timezone string `json:"timezone"`
}

type Externals struct {
	TVRage  *int64  `json:"tvrage"`
	TheTVDB *int64  `json:"thetvdb"`
	IMDB    *string `json:"imdb"`
}

type Image struct {
	MediumURL   string `json:"medium"`
	OriginalURL string `json:"original"`
}

func (im *Image) Medium() string {
	if im == nil {
		return ""
	}
	return im.MediumURL
}

type Result struct {
	Score float64 `json:"score"`
	Show  Show    `json:"show"`
}

type Season struct {
	ID           int      `json:"id"`
	URL          string   `json:"url"`
	Number       int      `json:"number"`
	Name         string   `json:"name"`
	EpisodeOrder int64    `json:"episodeOrder"` // count of episodes???
	PremiereDate string   `json:"premiereDate"`
	EndDate      string   `json:"endDate"`
	Network      *Network `json:"network"`
	Image        *Image   `json:"image"`
	Summary      string   `json:"summary"`
}

type Episode struct {
	ID      int    `json:"id"`
	URL     string `json:"url"`
	Name    string `json:"name"`
	Season  int    `json:"season"`
	Number  *int   `json:"number"`
	Type    string `json:"type"`
	Airdate string `json:"airdate"`
	Airtime string `json:"airtime"`
	Runtime int    `json:"runtime"`
	Image   *Image `json:"image"`
	Summary string `json:"summary"`
}
