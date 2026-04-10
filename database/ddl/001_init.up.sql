-- initial schema

CREATE TABLE User
(
	ID   TEXT PRIMARY KEY DEFAULT ('u'||newID()),
	Name TEXT NOT NULL
)
STRICT;

CREATE TABLE Slug
(
	Slug   TEXT PRIMARY KEY,
	Kind   TEXT NOT NULL CHECK (Kind IN ('movie', 'series', 'collection')),
	Target TEXT NOT NULL UNIQUE
)
STRICT;

CREATE TABLE Image
(
	ID          TEXT PRIMARY KEY DEFAULT ('io'||newID()),
	OriginalKey TEXT NOT NULL UNIQUE, -- blob store key for the as-uploaded bytes
	Type        TEXT NOT NULL CHECK (Type IN ('image/png', 'image/webp', 'image/jpeg'))
)
STRICT;

CREATE TABLE ImageRendition
(
	Key     TEXT PRIMARY KEY, -- blob store key
	ImageID TEXT NOT NULL REFERENCES Image ON DELETE CASCADE,
	Type    TEXT NOT NULL CHECK (Type IN ('image/webp', 'image/avif')),
	Width   INTEGER NOT NULL, -- physical pixels
	Height  INTEGER NOT NULL  -- physical pixels
)
STRICT;
CREATE INDEX Index_ImageRendition_ImageID ON ImageRendition (ImageID);

CREATE TABLE Series
(
	ID     TEXT PRIMARY KEY,
	Slug   TEXT NOT NULL UNIQUE,
	Title  TEXT NOT NULL,
	Status TEXT NOT NULL CHECK (Status IN (
		'In Development',
		'Running',
		'Ended',
		'To Be Determined'
	)),
	PremieredOn TEXT NOT NULL DEFAULT (''),
	EndedOn     TEXT NOT NULL DEFAULT (''),

	TVmazeID INTEGER UNIQUE,
	IMDBID   TEXT UNIQUE,
	TVDBID   INTEGER UNIQUE,
	TVRageID INTEGER UNIQUE
)
STRICT;

CREATE TABLE SeriesEdition
(
	ID             TEXT PRIMARY KEY DEFAULT ('sed'||newID()),
	SeriesID       TEXT NOT NULL REFERENCES Series,
	Slug           TEXT NOT NULL,
	Label          TEXT NOT NULL,
	Summary        TEXT NOT NULL,
	PosterID       TEXT NOT NULL DEFAULT 'ioplaceholderposter' REFERENCES Image,
	UNIQUE (SeriesID, Slug)
)
STRICT;

CREATE TABLE Season
(
	ID        TEXT PRIMARY KEY DEFAULT ('sn'||newID()),
	EditionID TEXT NOT NULL REFERENCES SeriesEdition,
	SortKey   TEXT NOT NULL,
	Title     TEXT NOT NULL, -- for display eg "Season 5"
	Number    INTEGER NOT NULL, -- for episode codes eg "s05e02"
	UNIQUE (EditionID, SortKey)
)
STRICT;

CREATE TABLE Episode
(
	ID      TEXT PRIMARY KEY DEFAULT ('ep'||newID()),
	Title   TEXT NOT NULL,
	Summary TEXT NOT NULL,
	Type    TEXT NOT NULL CHECK (Type IN (
		'regular',
		'significant_special',
		'insignificant_special'
	)),
	Airdate        TEXT NOT NULL, -- can be empty if unaired/unreleased
	Runtime        INTEGER NOT NULL, -- minutes
	ThumbnailID    TEXT NOT NULL DEFAULT 'ioplaceholderthumbnail' REFERENCES Image
)
STRICT;

CREATE TABLE SeasonEpisode
(
	EditionID TEXT NOT NULL REFERENCES SeriesEdition,
	SeasonID  TEXT NOT NULL REFERENCES Season,
	EpisodeID TEXT NOT NULL REFERENCES Episode,
	SortKey   INTEGER NOT NULL,
	Label     TEXT NOT NULL, -- episode number e.g. "5", or "Special"
	Number    INTEGER NOT NULL, -- 0 for specials
	Slug      TEXT NOT NULL,
	UNIQUE (SeasonID, SortKey),
	UNIQUE (EditionID, Slug),
	PRIMARY KEY (SeasonID, EpisodeID)
)
STRICT, WITHOUT ROWID;
CREATE INDEX Index_SeasonEpisode_EpisodeID ON SeasonEpisode (EpisodeID);

CREATE TABLE Movie
(
	ID     TEXT PRIMARY KEY,
	Slug   TEXT NOT NULL UNIQUE,
	TMDBID INTEGER UNIQUE,
	IMDBID TEXT UNIQUE
)
STRICT;

CREATE TABLE MovieEdition
(
	ID       TEXT PRIMARY KEY DEFAULT ('med'||newID()),
	MovieID  TEXT NOT NULL REFERENCES Movie,
	Slug     TEXT NOT NULL,
	Title    TEXT NOT NULL,
	Label    TEXT NOT NULL,
	Summary  TEXT NOT NULL,
	Year     TEXT NOT NULL,
	Runtime  INTEGER NOT NULL,    -- minutes
	PosterID TEXT NOT NULL DEFAULT 'ioplaceholderposter' REFERENCES Image,
	UNIQUE (MovieID, Slug)
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
	MovieEditionID TEXT NOT NULL REFERENCES MovieEdition,
	VideoID        TEXT NOT NULL REFERENCES Video,
	PRIMARY KEY (MovieEditionID, VideoID)
)
STRICT, WITHOUT ROWID;

CREATE TABLE Video
(
	ID           TEXT PRIMARY KEY DEFAULT ('vid'||newID()),
	InfoHash     TEXT REFERENCES Download (InfoHash),
	Name         TEXT NOT NULL, -- torrent path or file name
	State        TEXT NOT NULL DEFAULT ('pending') CHECK (State IN ('pending', 'importing')),
	OriginalKey  TEXT NOT NULL DEFAULT (''), -- empty during ingest
	MVPlaylist   TEXT NOT NULL DEFAULT (''), -- empty during ingest
	UNIQUE (InfoHash, Name)
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
	MaxHeight     INTEGER NOT NULL, -- 0 = source
	MaxFPS        INTEGER NOT NULL, -- 0 = source
	CopyAudio     INTEGER NOT NULL, -- 1: copy audio; 0: reencode to AAC
	SurroundAudio INTEGER NOT NULL, -- 1: encode as 5.1(back); 0: stereo downmix
	Key           TEXT NOT NULL DEFAULT (''), -- empty during ingest
	Playlist      TEXT NOT NULL DEFAULT (''), -- empty during ingest
	Priority      INTEGER NOT NULL -- 0 = highest priority (best rendition)
)
STRICT;

CREATE TABLE Task
(
	ID          TEXT PRIMARY KEY DEFAULT ('task'||newID()),
	Type        TEXT NOT NULL,
	Args        TEXT NOT NULL,
	Failures    INTEGER NOT NULL DEFAULT (0),
	NextRun     INTEGER NOT NULL,
	FailureDesc TEXT,
	Priority    INTEGER NOT NULL,
	Queue       TEXT NOT NULL,
	Running     INTEGER NOT NULL DEFAULT (0)
)
STRICT;
CREATE INDEX Index_Task_Queue ON Task (Queue, Running, NextRun, Priority, ID);

CREATE TABLE Download
(
	InfoHash            TEXT PRIMARY KEY,
	CreatedAt           INTEGER NOT NULL DEFAULT (unixepoch()),
	State               TEXT NOT NULL CHECK (State IN (
		'queued',
		'downloading',
		'downloaded',
		'imported',
		'error'
	)),
	Title               TEXT NOT NULL,
	Error               TEXT NOT NULL DEFAULT (''),
	Torrent             BLOB NOT NULL,
	Progress            REAL NOT NULL DEFAULT (0.0),
	AutoImport          INTEGER NOT NULL DEFAULT (0),
	PlanSeriesEditionID TEXT REFERENCES SeriesEdition,
	PlanMovieEditionID  TEXT REFERENCES MovieEdition
)
STRICT;
CREATE INDEX Index_Download_PlanSeriesEditionID ON Download (PlanSeriesEditionID)
WHERE PlanSeriesEditionID IS NOT NULL;
CREATE INDEX Index_Download_PlanMovieEditionID ON Download (PlanMovieEditionID)
WHERE PlanMovieEditionID IS NOT NULL;


CREATE TABLE Setting
(
	Key      TEXT PRIMARY KEY,
	"Group" TEXT NOT NULL,
	Value    TEXT NOT NULL  -- JSON-encoded
)
STRICT, WITHOUT ROWID;
CREATE INDEX Index_Setting_Group ON Setting ("Group");

CREATE TABLE Collection
(
	ID       TEXT PRIMARY KEY DEFAULT ('col'||newID()),
	Slug     TEXT NOT NULL UNIQUE,
	Title    TEXT NOT NULL,
	BannerID TEXT NOT NULL DEFAULT 'ioplaceholderbanner' REFERENCES Image
)
STRICT, WITHOUT ROWID;

CREATE TABLE CollectionMovie
(
	CollectionID TEXT NOT NULL REFERENCES Collection,
	MovieID      TEXT NOT NULL REFERENCES Movie,
	PRIMARY KEY (CollectionID, MovieID)
)
STRICT, WITHOUT ROWID;

CREATE TABLE CollectionSeries
(
	CollectionID TEXT NOT NULL REFERENCES Collection,
	SeriesID      TEXT NOT NULL REFERENCES Series,
	PRIMARY KEY (CollectionID, SeriesID)
)
STRICT, WITHOUT ROWID;
