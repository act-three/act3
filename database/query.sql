
-- name: AuthorCreate :one
INSERT INTO User (Name) VALUES (?)
RETURNING ID;

-- name: AuthorDelete :exec
DELETE FROM User
WHERE ID = ?;

-- name: AuthorGet :one
SELECT * FROM User
WHERE ID = ?
LIMIT 1;

-- name: AuthorList :many
SELECT * FROM User
ORDER BY Name;

-- name: DownloadCreate :one
INSERT INTO Download
(
	State,
	Title,
	Torrent,
	InfoHash,
	PlanSeriesEditionID,
	Plan
)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: DownloadGet :one
SELECT * FROM Download WHERE ID = ?;

-- name: DownloadGetByInfoHash :one
SELECT * FROM Download WHERE InfoHash = ?;

-- name: DownloadList :many
SELECT * FROM Download
ORDER BY ID DESC;

-- name: DownloadListByPlanSeriesEditionID :many
SELECT * FROM Download
WHERE PlanSeriesEditionID = ?
ORDER BY ID DESC;

-- name: DownloadListInfoHashesActive :many
SELECT InfoHash FROM Download
WHERE State = 'active';

-- name: DownloadUpdateError :one
UPDATE Download SET State = 'error', Error = ? WHERE ID = ? RETURNING *;

-- name: DownloadUpdatePlan :one
UPDATE Download SET
	PlanSeriesEditionID = ?,
	Plan = ?
WHERE ID = ?
RETURNING *;

-- name: DownloadUpdateProgress :one
UPDATE Download SET
	State = ?,
	Plan = ?,
	Progress = ?,
	Error = ''
WHERE ID = ? RETURNING *;

-- name: EpisodeCreate :one
INSERT INTO Episode
(
	Title,
	Summary,
	Type,
	Airdate,
	Runtime,
	TVmazeURL,
	TVmazeImageURL
)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: EpisodeGet :one
SELECT * FROM Episode WHERE ID = ?;

-- name: EpisodeListByEditionID :many
SELECT * FROM Episode
WHERE ID IN (
	SELECT ID FROM SeasonEpisode
	WHERE SeasonID IN (SELECT ID FROM Season WHERE EditionID = ?)
)
ORDER BY ID;

-- name: EpisodeListBySeriesID :many
SELECT * FROM Episode
WHERE ID IN (
	SELECT ID FROM SeasonEpisode
	WHERE SeasonID IN (
		SELECT ID FROM Season
		WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
	)
)
ORDER BY ID;

-- name: EpisodeVideoCreate :one
INSERT INTO EpisodeVideo (EpisodeID, VideoID)
VALUES (?, ?)
RETURNING *;

-- name: EpisodeVideoListByVideoID :many
SELECT * FROM EpisodeVideo
WHERE VideoID = ?;

-- name: MovieList :many
SELECT ID, Title, ArtworkKey FROM Movie;

-- name: ReleaseCreate :one
INSERT INTO Release
(
	Name,
	InfoHash
)
VALUES (?, ?)
RETURNING *;

-- name: ReleaseGetByInfoHash :one
SELECT * FROM Release WHERE InfoHash = ?;

-- name: RenditionForStreamingCreate :one
INSERT INTO RenditionForStreaming (
	VideoID,
	Remux,
	Codec,
	TargetBitrate,
	MaxHeight,
	MaxFPS,
	CopyAudio,
	SurroundAudio
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: RenditionForStreamingGet :one
SELECT * FROM RenditionForStreaming
WHERE ID = ?;

-- name: RenditionForStreamingListByVideoID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID  IN (SELECT VideoID FROM EpisodeVideo WHERE EpisodeID = ?);

-- name: RenditionForStreamingListDirectByVideoID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID = ?;

-- name: RenditionForStreamingUpdateEncode :one
UPDATE RenditionForStreaming
SET Hash = ?, Playlist = ?
WHERE ID = ?
RETURNING *;

-- name: SchemaVersionGet :one
SELECT version, digest FROM schema LIMIT 1;

-- name: SchemaVersionSet :exec
UPDATE schema SET version = ?, digest = ?;

-- name: SeasonCreate :one
INSERT INTO Season
(
	EditionID,
	SortKey,
	Name,
	Number,
	TVmazeURL,
	Summary,
	EpisodeOrder,
	PremieredOn,
	EndedOn,
	TVmazeImageURL
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: SeasonEpisodeCreate :exec
INSERT INTO SeasonEpisode (SeasonID, EpisodeID, SortKey, Label, Number)
VALUES (?, ?, ?, ?, ?);

-- name: SeasonEpisodeListByEditionID :many
SELECT * FROM SeasonEpisode
WHERE SeasonID IN (SELECT ID FROM Season WHERE EditionID = ?)
ORDER BY SortKey;

-- name: SeasonEpisodeListByEpisodeID :many
SELECT * FROM SeasonEpisode WHERE EpisodeID = ?;

-- name: SeasonEpisodeListBySeriesID :many
SELECT * FROM SeasonEpisode
WHERE SeasonID IN (
	SELECT ID FROM Season
	WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
)
ORDER BY SortKey;

-- name: SeasonGet :one
SELECT * FROM Season WHERE ID = ?;

-- name: SeasonListByEditionID :many
SELECT * FROM Season WHERE EditionID = ?
ORDER BY SortKey;

-- name: SeasonListBySeriesID :many
SELECT * FROM Season
WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
ORDER BY SortKey;

-- name: SeriesCreate :one
INSERT INTO Series
(
	Title,
	Summary,
	Status,
	Language,
	PremieredOn,
	EndedOn,
	TVmazeID,
	TVmazeURL,
	TVmazeImageURL,
	TVmazeUpdatedAt,
	IMDBID,
	TVDBID,
	TVRageID
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: SeriesEditionCreate :one
INSERT INTO SeriesEdition (Title, SeriesID) VALUES (?, ?)
RETURNING *;

-- name: SeriesEditionGet :one
SELECT * FROM SeriesEdition WHERE ID = ?;

-- name: SeriesEditionListBySeriesID :many
SELECT * FROM SeriesEdition WHERE SeriesID = ?;

-- name: SeriesGenreAdd :exec
INSERT INTO SeriesGenre (SeriesID, GenreName) VALUES (?, ?);

-- name: SeriesGenreList :many
SELECT GenreName FROM SeriesGenre WHERE SeriesID = ?;

-- name: SeriesGet :one
SELECT * FROM Series WHERE ID = ?;

-- name: SeriesGetByEditionID :one
SELECT * FROM Series
WHERE ID IN (SELECT SeriesID FROM SeriesEdition WHERE SeriesEdition.ID = ?);

-- name: SeriesGetByTVmazeID :one
SELECT * FROM Series WHERE TVmazeID = ?;

-- name: SeriesList :many
SELECT * FROM Series;

-- name: SeriesListByTVmazeID :many
SELECT * FROM Series WHERE TVmazeID IN (sqlc.slice(ids));

-- name: StorageCreate :exec
INSERT INTO Storage (Path, Contents) VALUES (?, ?);

-- name: StorageList :many
SELECT * FROM Storage;

-- name: TaskCreate :one
INSERT INTO Task (Type, Args) VALUES (?, ?)
RETURNING *;

-- name: TaskDelete :exec
DELETE FROM Task WHERE ID = ?;

-- name: TaskGet :one
SELECT * FROM Task WHERE ID = ?;

-- name: TaskList :many
SELECT * FROM Task;

-- name: TaskReschedule :one
UPDATE Task SET
	Failures = ?,
	NextRun = ?,
	FailureDesc = ?
WHERE ID = ?
RETURNING *;

-- name: TaskSaveOneOff :exec
UPDATE Task SET
	Failures = ?,
	FailureDesc = ?
WHERE ID = ?;

-- name: TransmissionGet :one
SELECT Path, BaseURL FROM ConfigTransmission LIMIT 1;

-- name: TransmissionSet :exec
INSERT INTO ConfigTransmission (Single, Path, BaseURL) VALUES (0, ?1, ?2)
ON CONFLICT (Single) DO UPDATE SET Path = ?1, BaseURL = ?2;

-- name: VideoCreate :one
INSERT INTO Video
(
	ReleaseID,
	ReleasePath
)
VALUES (?, ?)
RETURNING *;

-- name: VideoGet :one
SELECT * FROM Video WHERE ID = ?;

-- name: VideoGetByReleasePath :one
SELECT * FROM Video WHERE ReleaseID = ? AND ReleasePath = ?;

-- name: VideoListByEpisodeID :many
SELECT * FROM Video
WHERE ID IN (SELECT VideoID FROM EpisodeVideo WHERE EpisodeID = ?);

-- name: VideoUpdateOriginalHash :one
UPDATE Video SET OriginalHash = ? WHERE ID = ?
RETURNING *;

-- name: VideoUpdateMVPlaylist :one
UPDATE Video SET MVPlaylist = ? WHERE ID = ?
RETURNING *;
