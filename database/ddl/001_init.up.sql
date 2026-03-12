-- initial schema

CREATE TABLE User
(
	ID   TEXT PRIMARY KEY DEFAULT ('u'||newID()),
	Name TEXT NOT NULL
)
STRICT;

CREATE TABLE Storage
(
	Path     TEXT PRIMARY KEY,
	Contents TEXT NOT NULL CHECK (Contents IN ('Movie', 'Series'))
)
STRICT;

CREATE TABLE Series
(
	ID      TEXT PRIMARY KEY,
	Slug    TEXT NOT NULL UNIQUE,
	Title   TEXT NOT NULL,
	Summary TEXT NOT NULL,
	Status  TEXT NOT NULL CHECK (Status IN (
		'In Development',
		'Running',
		'Ended',
		'To Be Determined'
	)),
	Language    TEXT NOT NULL,
	PremieredOn TEXT,
	EndedOn     TEXT,

	TVmazeID        INTEGER UNIQUE,
	TVmazeURL       TEXT,
	TVmazeImageURL  TEXT NOT NULL,
	TVmazeUpdatedAt INTEGER NOT NULL DEFAULT (0),
	IMDBID          TEXT UNIQUE,
	TVDBID          INTEGER UNIQUE,
	TVRageID        INTEGER UNIQUE
)
STRICT;

CREATE TABLE SeriesGenre
(
	SeriesID  TEXT NOT NULL REFERENCES Series,
	GenreName TEXT NOT NULL,
	PRIMARY KEY (SeriesID, GenreName)
)
STRICT, WITHOUT ROWID;

CREATE TABLE SeriesEdition
(
	ID       TEXT PRIMARY KEY DEFAULT ('sed'||newID()),
	Title    TEXT NOT NULL,
	SeriesID TEXT NOT NULL REFERENCES Series,
	UNIQUE (SeriesID, Title)
)
STRICT;

CREATE TABLE Season
(
	ID           TEXT PRIMARY KEY DEFAULT ('sn'||newID()),
	EditionID    TEXT NOT NULL REFERENCES SeriesEdition,
	SortKey      TEXT NOT NULL,
	Name         TEXT NOT NULL,
	Number       INTEGER NOT NULL,
	TVmazeURL    TEXT,
	Summary      TEXT NOT NULL,
	EpisodeOrder INTEGER NOT NULL,
	PremieredOn  TEXT,
	EndedOn      TEXT,
	TVmazeImageURL TEXT NOT NULL,
	UNIQUE (EditionID, SortKey)
)
STRICT;

CREATE TABLE Episode
(
	ID      TEXT PRIMARY KEY DEFAULT ('ep'||newID()),
	Slug    TEXT NOT NULL UNIQUE DEFAULT '',
	Title   TEXT NOT NULL,
	Summary TEXT NOT NULL,
	Type    TEXT NOT NULL CHECK (Type IN (
		'regular',
		'significant_special',
		'insignificant_special'
	)),
	Airdate        TEXT NOT NULL, -- can be empty if unaired/unreleased
	Runtime        INTEGER NOT NULL, -- minutes
	TVmazeURL      TEXT,
	TVmazeImageURL TEXT NOT NULL
)
STRICT;

CREATE TABLE SeasonEpisode
(
	SeasonID  TEXT NOT NULL REFERENCES Season,
	EpisodeID TEXT NOT NULL REFERENCES Episode,
	SortKey   TEXT NOT NULL,
	Label     TEXT NOT NULL, -- episode number e.g. "5", or "Special"
	Number    INTEGER, -- NULL for specials
	UNIQUE (SeasonID, SortKey),
	PRIMARY KEY (SeasonID, EpisodeID)
)
STRICT, WITHOUT ROWID;
CREATE INDEX Index_SeasonEpisode_EpisodeID ON SeasonEpisode (EpisodeID);

CREATE TABLE Movie
(
	ID       TEXT PRIMARY KEY DEFAULT ('mo'||newID()),
	Slug     TEXT NOT NULL UNIQUE,
	Title    TEXT NOT NULL,
	Summary  TEXT NOT NULL DEFAULT (''),
	Year     INTEGER NOT NULL DEFAULT (0),    -- 0 = unknown
	Runtime  INTEGER NOT NULL DEFAULT (0),    -- minutes
	ImageURL TEXT NOT NULL DEFAULT (''),
	TMDBID   INTEGER UNIQUE,
	IMDBID   TEXT UNIQUE
)
STRICT;

CREATE TABLE Tag
(
	ID      TEXT PRIMARY KEY DEFAULT ('t'||newID()),
	Name    TEXT NOT NULL,
	OwnerID INTEGER
)
STRICT;

CREATE TABLE TagSeries
(
	TagID    TEXT NOT NULL REFERENCES Tag,
	SeriesID TEXT NOT NULL REFERENCES Series,
	PRIMARY KEY (TagID, SeriesID)
)
STRICT, WITHOUT ROWID;
CREATE INDEX Index_TagSeries_SeriesID ON TagSeries (SeriesID);

CREATE TABLE TagMovie
(
	TagID   TEXT NOT NULL REFERENCES Tag,
	MovieID TEXT NOT NULL REFERENCES Movie,
	PRIMARY KEY (TagID, MovieID)
)
STRICT, WITHOUT ROWID;
CREATE INDEX Index_TagMovie_MovieID ON TagMovie (MovieID);

CREATE TABLE Release
(
	ID       TEXT PRIMARY KEY DEFAULT ('rel'||newID()),
	Name     TEXT NOT NULL,
	InfoHash TEXT UNIQUE
)
STRICT;

CREATE TABLE EpisodeVideo
(
	EpisodeID TEXT NOT NULL REFERENCES Episode,
	VideoID   TEXT NOT NULL REFERENCES Video,
	PRIMARY KEY (EpisodeID, VideoID)
)
STRICT;

CREATE TABLE MovieVideo
(
	MovieID   TEXT NOT NULL REFERENCES Movie,
	VideoID   TEXT NOT NULL REFERENCES Video,
	PRIMARY KEY (MovieID, VideoID)
)
STRICT, WITHOUT ROWID;

CREATE TABLE Video
(
	ID           TEXT PRIMARY KEY DEFAULT ('vid'||newID()),
	ReleaseID    TEXT NOT NULL REFERENCES Release,
	ReleasePath  TEXT NOT NULL,
	OriginalHash TEXT NOT NULL DEFAULT (''), -- empty during ingest
	MVPlaylist   TEXT NOT NULL DEFAULT (''), -- empty during ingest
	UNIQUE (ReleaseID, ReleasePath)
)
STRICT;

CREATE TABLE AudioTrack
(
	ID            TEXT PRIMARY KEY DEFAULT ('at'||newID()),
	VideoID       TEXT NOT NULL REFERENCES Video,
	StreamIndex   INTEGER NOT NULL,
	Language      TEXT NOT NULL,
	Title         TEXT NOT NULL,
	Channels      INTEGER NOT NULL,
	ChannelLayout TEXT NOT NULL,
	Codec         TEXT NOT NULL,
	UNIQUE (VideoID, StreamIndex)
)
STRICT;

CREATE TABLE RenditionForStreaming
(
	ID            TEXT PRIMARY KEY DEFAULT ('rfs'||newID()),
	VideoID       TEXT NOT NULL REFERENCES Video,
	Remux         INTEGER NOT NULL, -- 1: copy video stream; 0: reencode
	Codec         TEXT NOT NULL, -- "h264" or "hevc"
	TargetBitrate INTEGER NOT NULL, -- kbit/s
	MaxHeight     INTEGER NOT NULL DEFAULT (0), -- 0 = source
	MaxFPS        INTEGER NOT NULL DEFAULT (0), -- 0 = source
	CopyAudio     INTEGER NOT NULL, -- 1: copy audio; 0: reencode to AAC
	SurroundAudio INTEGER NOT NULL DEFAULT (0), -- 1: encode as 5.1(back); 0: stereo downmix
	Hash          TEXT NOT NULL DEFAULT (''), -- empty during ingest
	Playlist      TEXT NOT NULL DEFAULT (''), -- empty during ingest
	Priority      INTEGER NOT NULL DEFAULT (0) -- 0 = highest priority (best rendition)
)
STRICT;

CREATE TABLE Task
(
	ID          TEXT PRIMARY KEY DEFAULT ('task'||newID()),
	Type        TEXT NOT NULL,
	Args        TEXT NOT NULL,
	Failures    INTEGER NOT NULL DEFAULT (0),
	NextRun     INTEGER NOT NULL DEFAULT (0),
	FailureDesc TEXT,
	Priority    INTEGER NOT NULL DEFAULT (0),
	Queue       TEXT NOT NULL,
	Running     INTEGER NOT NULL DEFAULT (0)
)
STRICT;
CREATE INDEX Index_Task_Queue ON Task (Queue, Running, NextRun, Priority, ID);

CREATE TABLE Download
(
	ID        TEXT PRIMARY KEY DEFAULT ('dl'||newID()),
	CreatedAt INTEGER NOT NULL DEFAULT (unixepoch()),
	State     TEXT NOT NULL CHECK (State IN (
		'added',
		'active',
		'done',
		'error'
	)),
	Title               TEXT NOT NULL,
	Error               TEXT NOT NULL DEFAULT (''),
	Torrent             BLOB NOT NULL,
	InfoHash            TEXT NOT NULL UNIQUE,
	Progress            REAL NOT NULL DEFAULT (0.0),
	PlanSeriesEditionID TEXT REFERENCES SeriesEdition,
	Plan                TEXT NOT NULL DEFAULT ('{}')
)
STRICT;
CREATE INDEX Index_Download_PlanEditionID ON Download (PlanSeriesEditionID)
WHERE PlanSeriesEditionID IS NOT NULL;

CREATE TABLE Setting
(
	Key      TEXT PRIMARY KEY,
	"Group" TEXT NOT NULL,
	Value    TEXT NOT NULL  -- JSON-encoded
)
STRICT, WITHOUT ROWID;
CREATE INDEX Index_Setting_Group ON Setting ("Group");
