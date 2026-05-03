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
	ID          TEXT PRIMARY KEY DEFAULT ('i'||newID()),
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
	Slug   TEXT NOT NULL,
	Title  TEXT NOT NULL,
	Status TEXT NOT NULL CHECK (Status IN (
		'In Development',
		'Running',
		'Ended',
		'To Be Determined'
	)),
	PremieredOn TEXT NOT NULL DEFAULT (''),
	EndedOn     TEXT NOT NULL DEFAULT (''),

	TVmazeID INTEGER,
	IMDBID   TEXT,
	TVDBID   INTEGER,
	TVRageID INTEGER,

	DeletedAt INTEGER
)
STRICT;
CREATE UNIQUE INDEX UQ_Series_Slug     ON Series (Slug)     WHERE DeletedAt IS NULL;
CREATE UNIQUE INDEX UQ_Series_TVmazeID ON Series (TVmazeID) WHERE DeletedAt IS NULL AND TVmazeID IS NOT NULL;
CREATE UNIQUE INDEX UQ_Series_IMDBID   ON Series (IMDBID)   WHERE DeletedAt IS NULL AND IMDBID   IS NOT NULL;
CREATE UNIQUE INDEX UQ_Series_TVDBID   ON Series (TVDBID)   WHERE DeletedAt IS NULL AND TVDBID   IS NOT NULL;
CREATE UNIQUE INDEX UQ_Series_TVRageID ON Series (TVRageID) WHERE DeletedAt IS NULL AND TVRageID IS NOT NULL;
CREATE INDEX Idx_Series_Trash          ON Series (DeletedAt) WHERE DeletedAt IS NOT NULL;

CREATE TABLE SeriesEdition
(
	ID             TEXT PRIMARY KEY DEFAULT ('sed'||newID()),
	SeriesID       TEXT NOT NULL REFERENCES Series,
	Slug           TEXT NOT NULL,
	Label          TEXT NOT NULL,
	Summary        TEXT NOT NULL,
	PosterID       TEXT NOT NULL DEFAULT 'iplaceholderposter' REFERENCES Image,

	DeletedAt INTEGER
)
STRICT;
CREATE UNIQUE INDEX UQ_SeriesEdition_SeriesID_Slug ON SeriesEdition (SeriesID, Slug) WHERE DeletedAt IS NULL;
CREATE INDEX Idx_SeriesEdition_Trash ON SeriesEdition (DeletedAt) WHERE DeletedAt IS NOT NULL;

CREATE TABLE Season
(
	ID        TEXT PRIMARY KEY DEFAULT ('sn'||newID()),
	EditionID TEXT NOT NULL REFERENCES SeriesEdition,
	SortKey   TEXT NOT NULL,
	Title     TEXT NOT NULL, -- for display eg "Season 5"
	Number    INTEGER NOT NULL, -- for episode codes eg "s05e02"

	DeletedAt INTEGER
)
STRICT;
CREATE UNIQUE INDEX UQ_Season_EditionID_SortKey ON Season (EditionID, SortKey) WHERE DeletedAt IS NULL;
CREATE INDEX Idx_Season_Trash ON Season (DeletedAt) WHERE DeletedAt IS NOT NULL;

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
	ThumbnailID    TEXT NOT NULL DEFAULT 'iplaceholderthumbnail' REFERENCES Image,

	DeletedAt INTEGER
)
STRICT;
CREATE INDEX Idx_Episode_Trash ON Episode (DeletedAt) WHERE DeletedAt IS NOT NULL;

CREATE TABLE SeasonEpisode
(
	EditionID TEXT NOT NULL REFERENCES SeriesEdition,
	SeasonID  TEXT NOT NULL REFERENCES Season,
	EpisodeID TEXT NOT NULL REFERENCES Episode,
	SortKey   INTEGER NOT NULL,
	Label     TEXT NOT NULL, -- episode number e.g. "5", or "Special"
	Number    INTEGER NOT NULL, -- 0 for specials
	Slug      TEXT NOT NULL,

	DeletedAt INTEGER,

	PRIMARY KEY (SeasonID, EpisodeID)
)
STRICT, WITHOUT ROWID;
CREATE INDEX Index_SeasonEpisode_EpisodeID ON SeasonEpisode (EpisodeID);
CREATE UNIQUE INDEX UQ_SeasonEpisode_SeasonID_SortKey ON SeasonEpisode (SeasonID, SortKey) WHERE DeletedAt IS NULL;
CREATE UNIQUE INDEX UQ_SeasonEpisode_EditionID_Slug  ON SeasonEpisode (EditionID, Slug)   WHERE DeletedAt IS NULL;

CREATE TABLE Movie
(
	ID     TEXT PRIMARY KEY,
	Slug   TEXT NOT NULL,
	TMDBID INTEGER,
	IMDBID TEXT,

	DeletedAt INTEGER
)
STRICT;
CREATE UNIQUE INDEX UQ_Movie_Slug   ON Movie (Slug)   WHERE DeletedAt IS NULL;
CREATE UNIQUE INDEX UQ_Movie_TMDBID ON Movie (TMDBID) WHERE DeletedAt IS NULL AND TMDBID IS NOT NULL;
CREATE UNIQUE INDEX UQ_Movie_IMDBID ON Movie (IMDBID) WHERE DeletedAt IS NULL AND IMDBID IS NOT NULL;
CREATE INDEX Idx_Movie_Trash        ON Movie (DeletedAt) WHERE DeletedAt IS NOT NULL;

CREATE TABLE MovieEdition
(
	ID       TEXT PRIMARY KEY DEFAULT ('med'||newID()),
	MovieID  TEXT NOT NULL REFERENCES Movie,
	Slug     TEXT NOT NULL,
	Title    TEXT NOT NULL,
	Label    TEXT NOT NULL,
	Summary  TEXT NOT NULL,
	ReleaseDate TEXT NOT NULL, -- YYYY-MM-DD; may be empty
	Runtime  INTEGER NOT NULL,    -- minutes
	PosterID TEXT NOT NULL DEFAULT 'iplaceholderposter' REFERENCES Image,

	DeletedAt INTEGER
)
STRICT;
CREATE UNIQUE INDEX UQ_MovieEdition_MovieID_Slug ON MovieEdition (MovieID, Slug) WHERE DeletedAt IS NULL;
CREATE INDEX Idx_MovieEdition_Trash ON MovieEdition (DeletedAt) WHERE DeletedAt IS NOT NULL;

CREATE TABLE EpisodeVideo
(
	EpisodeID TEXT NOT NULL REFERENCES Episode,
	VideoID   TEXT NOT NULL REFERENCES Video,
	Active    INTEGER NOT NULL DEFAULT 0 CHECK (Active IN (0, 1)),

	DeletedAt INTEGER,

	PRIMARY KEY (EpisodeID, VideoID)
)
STRICT;
CREATE UNIQUE INDEX UQ_EpisodeVideo_Active ON EpisodeVideo (EpisodeID)
WHERE Active = 1 AND DeletedAt IS NULL;

CREATE TABLE MovieVideo
(
	MovieEditionID TEXT NOT NULL REFERENCES MovieEdition,
	VideoID        TEXT NOT NULL REFERENCES Video,
	Active         INTEGER NOT NULL DEFAULT 0 CHECK (Active IN (0, 1)),

	DeletedAt INTEGER,

	PRIMARY KEY (MovieEditionID, VideoID)
)
STRICT, WITHOUT ROWID;
CREATE UNIQUE INDEX UQ_MovieVideo_Active ON MovieVideo (MovieEditionID)
WHERE Active = 1 AND DeletedAt IS NULL;

CREATE TABLE Video
(
	ID           TEXT PRIMARY KEY DEFAULT ('vid'||newID()),
	InfoHash     TEXT REFERENCES Download (InfoHash) ON DELETE SET NULL,
	Name         TEXT NOT NULL, -- torrent path or file name
	State        TEXT NOT NULL DEFAULT ('pending') CHECK (State IN ('pending', 'importing')),
	OriginalKey  TEXT NOT NULL DEFAULT (''), -- empty during ingest
	OriginalType TEXT NOT NULL DEFAULT (''), -- MIME type of original; empty until probed
	Format       TEXT NOT NULL DEFAULT (''), -- ffprobe format_name; empty until probed
	Duration     INTEGER NOT NULL DEFAULT (0), -- milliseconds; 0 until probed
	Width        INTEGER NOT NULL DEFAULT (0), -- source video pixels; 0 until probed
	Height       INTEGER NOT NULL DEFAULT (0), -- source video pixels; 0 until probed
	FrameRateNum INTEGER NOT NULL DEFAULT (0), -- source frame rate numerator; 0 until probed
	FrameRateDen INTEGER NOT NULL DEFAULT (0), -- source frame rate denominator; 0 until probed
	Playable     INTEGER NOT NULL DEFAULT (0), -- 1 once all renditions needed for an MV playlist are present
	ContentHash  BLOB, -- blake3 of the original bytes; null until copied

	DeletedAt INTEGER
)
STRICT;
CREATE UNIQUE INDEX UQ_Video_InfoHash_Name ON Video (InfoHash, Name) WHERE DeletedAt IS NULL;
CREATE UNIQUE INDEX UQ_Video_ContentHash ON Video (ContentHash) WHERE ContentHash IS NOT NULL AND DeletedAt IS NULL;
CREATE INDEX Idx_Video_Trash ON Video (DeletedAt) WHERE DeletedAt IS NOT NULL;

CREATE TABLE AudioTrack
(
	ID            TEXT PRIMARY KEY DEFAULT ('at'||newID()),
	VideoID       TEXT NOT NULL REFERENCES Video,
	StreamIndex   INTEGER NOT NULL,
	Language      TEXT NOT NULL,
	Title         TEXT NOT NULL,
	Channels      INTEGER NOT NULL,
	ChannelLayout TEXT NOT NULL,
	SampleRate    INTEGER NOT NULL DEFAULT (0),
	Codec         TEXT NOT NULL,
	Profile       TEXT NOT NULL DEFAULT (''),
	UNIQUE (VideoID, StreamIndex)
)
STRICT;

CREATE TABLE SubtitleTrack
(
	ID            TEXT PRIMARY KEY DEFAULT ('sub'||newID()),
	VideoID       TEXT NOT NULL REFERENCES Video,
	StreamIndex   INTEGER NOT NULL,
	Language      TEXT NOT NULL,
	Title         TEXT NOT NULL,
	OriginalCodec TEXT NOT NULL,
	OriginalKey   TEXT NOT NULL DEFAULT (''), -- empty during ingest
	WebVTTKey     TEXT NOT NULL DEFAULT (''), -- empty during ingest
	Forced        INTEGER NOT NULL DEFAULT 0 CHECK (Forced IN (0, 1)),
	UNIQUE (VideoID, StreamIndex)
)
STRICT;

CREATE TABLE Rendition
(
	ID            TEXT PRIMARY KEY DEFAULT ('rend'||newID()),
	VideoID       TEXT NOT NULL REFERENCES Video,
	Purpose       TEXT NOT NULL CHECK (Purpose IN ('streaming', 'download')),
	Remux         INTEGER NOT NULL, -- 1: copy video stream; 0: reencode
	Codec         TEXT NOT NULL, -- "h264" or "hevc"
	TargetBitrate INTEGER NOT NULL, -- kbit/s
	MaxHeight     INTEGER NOT NULL, -- 0 = source
	MaxFPS        INTEGER NOT NULL, -- 0 = source
	Key           TEXT NOT NULL DEFAULT (''), -- empty during ingest
	Playlist      TEXT NOT NULL DEFAULT (''), -- HLS media playlist; empty for download
	Priority      INTEGER NOT NULL -- 0 = highest priority (best rendition)
)
STRICT;

CREATE TABLE AudioRendition
(
	ID           TEXT PRIMARY KEY DEFAULT ('arend'||newID()),
	VideoID      TEXT NOT NULL REFERENCES Video,
	AudioTrackID TEXT NOT NULL REFERENCES AudioTrack,
	Channels     INTEGER NOT NULL, -- output channels (1 = mono, 2 = stereo, 6 = 5.1)
	Bitrate      INTEGER NOT NULL, -- kbit/s
	Codec        TEXT NOT NULL,    -- always "aac" for v1
	Key          TEXT NOT NULL DEFAULT (''), -- empty during ingest
	Playlist     TEXT NOT NULL DEFAULT (''), -- HLS media playlist; empty during ingest
	Priority     INTEGER NOT NULL, -- 0 = highest priority
	UNIQUE (VideoID, AudioTrackID, Channels)
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
	State       TEXT NOT NULL DEFAULT ('queued')
		CHECK (State IN ('queued', 'running', 'failed'))
)
STRICT;
CREATE INDEX Index_Task_Queue ON Task (Queue, State, NextRun, Priority, ID);

CREATE TABLE Download
(
	InfoHash            TEXT PRIMARY KEY,
	CreatedAt           INTEGER NOT NULL DEFAULT (unixepoch()),
	LastActivityAt      INTEGER NOT NULL DEFAULT (0),
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
	MovieEditionID  TEXT REFERENCES MovieEdition,
	SeriesEditionID TEXT REFERENCES SeriesEdition,
	DeletedAt           INTEGER,
	CHECK ((MovieEditionID IS NULL) != (SeriesEditionID IS NULL))
)
STRICT;
CREATE INDEX Index_Download_SeriesEditionID ON Download (SeriesEditionID)
WHERE SeriesEditionID IS NOT NULL;
CREATE INDEX Index_Download_MovieEditionID ON Download (MovieEditionID)
WHERE MovieEditionID IS NOT NULL;
CREATE INDEX Idx_Download_Trash            ON Download (DeletedAt)
	WHERE DeletedAt IS NOT NULL;
CREATE INDEX Idx_Download_LastActivity     ON Download (State, LastActivityAt)
	WHERE DeletedAt IS NULL;


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
	Slug     TEXT NOT NULL,
	Title    TEXT NOT NULL,
	BannerID TEXT NOT NULL DEFAULT 'iplaceholderbanner' REFERENCES Image,

	DeletedAt INTEGER
)
STRICT, WITHOUT ROWID;
CREATE UNIQUE INDEX UQ_Collection_Slug ON Collection (Slug) WHERE DeletedAt IS NULL;
CREATE INDEX Idx_Collection_Trash ON Collection (DeletedAt) WHERE DeletedAt IS NOT NULL;

CREATE TABLE CollectionMovie
(
	CollectionID TEXT NOT NULL REFERENCES Collection,
	MovieID      TEXT NOT NULL REFERENCES Movie,

	DeletedAt INTEGER,

	PRIMARY KEY (CollectionID, MovieID)
)
STRICT, WITHOUT ROWID;

CREATE TABLE CollectionSeries
(
	CollectionID TEXT NOT NULL REFERENCES Collection,
	SeriesID      TEXT NOT NULL REFERENCES Series,

	DeletedAt INTEGER,

	PRIMARY KEY (CollectionID, SeriesID)
)
STRICT, WITHOUT ROWID;

-- Trash table: centralized cascade-origin tracking for soft-deleted entities.
-- Junctions only carry DeletedAt; the Trash table tracks which root
-- caused the cascade and stores frozen title/subtitle for display.
CREATE TABLE Trash
(
	ID        TEXT PRIMARY KEY,
	Title     TEXT NOT NULL,
	Subtitle  TEXT NOT NULL,
	DeletedAt INTEGER NOT NULL,
	CascadeOf TEXT REFERENCES Trash(ID) ON DELETE CASCADE
)
STRICT;
CREATE INDEX Idx_Trash_DirectDeletedAt ON Trash (DeletedAt DESC) WHERE CascadeOf IS NULL;
CREATE INDEX Idx_Trash_CascadeOf       ON Trash (CascadeOf)     WHERE CascadeOf IS NOT NULL;
