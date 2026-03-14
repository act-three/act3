package tmdb

// SearchResponse is the response from GET /3/search/movie.
type SearchResponse struct {
	Page         int            `json:"page"`
	TotalPages   int            `json:"total_pages"`
	TotalResults int            `json:"total_results"`
	Results      []SearchResult `json:"results"`
}

// SearchResult is a single movie in search results.
type SearchResult struct {
	ID               int     `json:"id"`
	Title            string  `json:"title"`
	OriginalTitle    string  `json:"original_title"`
	OriginalLanguage string  `json:"original_language"`
	Overview         string  `json:"overview"`
	ReleaseDate      string  `json:"release_date"` // "YYYY-MM-DD"
	PosterPath       *string `json:"poster_path"`
	BackdropPath     *string `json:"backdrop_path"`
	GenreIDs         []int   `json:"genre_ids"`
	Popularity       float64 `json:"popularity"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Adult            bool    `json:"adult"`
	Video            bool    `json:"video"`
}

// Movie is the response from GET /3/movie/{id}.
type Movie struct {
	ID               int     `json:"id"`
	IMDBID           *string `json:"imdb_id"`
	Title            string  `json:"title"`
	OriginalTitle    string  `json:"original_title"`
	OriginalLanguage string  `json:"original_language"`
	Overview         string  `json:"overview"`
	Tagline          string  `json:"tagline"`
	Status           string  `json:"status"`
	ReleaseDate      string  `json:"release_date"`
	Runtime          int     `json:"runtime"` // minutes
	Popularity       float64 `json:"popularity"`
	VoteAverage      float64 `json:"vote_average"`
	VoteCount        int     `json:"vote_count"`
	Adult            bool    `json:"adult"`
	PosterPath       *string `json:"poster_path"`
	BackdropPath     *string `json:"backdrop_path"`
	Genres           []Genre `json:"genres"`
}

// Genre is a movie genre.
type Genre struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}
