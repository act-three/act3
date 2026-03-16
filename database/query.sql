
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
	PlanMovieEditionID,
	Plan
)
VALUES (?, ?, ?, ?, ?, ?, ?)
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

-- name: DownloadListByPlanMovieEditionID :many
SELECT * FROM Download
WHERE PlanMovieEditionID = ?
ORDER BY ID DESC;

-- name: DownloadListInfoHashesActive :many
SELECT InfoHash FROM Download
WHERE State = 'active';

-- name: DownloadUpdateError :one
UPDATE Download SET State = 'error', Error = ? WHERE ID = ? RETURNING *;

-- name: DownloadUpdatePlanSeries :one
UPDATE Download SET
	PlanSeriesEditionID = ?,
	Plan = ?
WHERE ID = ?
RETURNING *;

-- name: DownloadUpdatePlanMovie :one
UPDATE Download SET
	PlanMovieEditionID = ?,
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
	Slug,
	Title,
	Summary,
	Type,
	Airdate,
	Runtime,
	TVmazeURL,
	TVmazeImageURL
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: EpisodeGet :one
SELECT * FROM Episode WHERE ID = ?;

-- name: EpisodeGetBySlug :one
SELECT * FROM Episode WHERE Slug = ?;

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

-- name: EpisodeVideoListByEditionID :many
SELECT * FROM EpisodeVideo
WHERE EpisodeID IN (
	SELECT EpisodeID FROM SeasonEpisode
	WHERE SeasonID IN (SELECT ID FROM Season WHERE EditionID = ?)
);

-- name: EpisodeVideoListBySeriesID :many
SELECT * FROM EpisodeVideo
WHERE EpisodeID IN (
	SELECT EpisodeID FROM SeasonEpisode
	WHERE SeasonID IN (
		SELECT ID FROM Season
		WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
	)
);

-- name: AudioTrackCreate :one
INSERT INTO AudioTrack (
	VideoID, StreamIndex, Language, Title,
	Channels, ChannelLayout, Codec
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: AudioTrackListByVideoID :many
SELECT * FROM AudioTrack
WHERE VideoID = ?
ORDER BY StreamIndex;

-- name: AudioTrackDeleteByVideoID :exec
DELETE FROM AudioTrack WHERE VideoID = ?;

-- name: MovieCreate :one
INSERT INTO Movie (ID, Slug, Title, Summary, Year, Runtime, ImageURL, TMDBID, IMDBID)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: MovieGet :one
SELECT * FROM Movie WHERE ID = ?;

-- name: MovieGetBySlug :one
SELECT * FROM Movie WHERE Slug = ?;

-- name: MovieSlugExists :one
SELECT COUNT(*) FROM Movie WHERE Slug = ?;

-- name: MovieList :many
SELECT * FROM Movie
ORDER BY Title;

-- name: MovieListByTMDBID :many
SELECT * FROM Movie WHERE TMDBID IN (sqlc.slice(ids));

-- name: MovieEditionCreate :one
INSERT INTO MovieEdition (Title, MovieID) VALUES (?, ?)
RETURNING *;

-- name: MovieEditionGet :one
SELECT * FROM MovieEdition WHERE ID = ?;

-- name: MovieEditionListByMovieID :many
SELECT * FROM MovieEdition WHERE MovieID = ?;

-- name: MovieGetByEditionID :one
SELECT * FROM Movie
WHERE ID IN (SELECT MovieID FROM MovieEdition WHERE MovieEdition.ID = ?);

-- name: MovieVideoCreate :one
INSERT INTO MovieVideo (MovieEditionID, VideoID)
VALUES (?, ?)
RETURNING *;

-- name: MovieVideoListByMovieEditionID :many
SELECT * FROM MovieVideo
WHERE MovieEditionID = ?;

-- name: MovieVideoListByMovieID :many
SELECT * FROM MovieVideo
WHERE MovieEditionID IN (SELECT ID FROM MovieEdition WHERE MovieID = ?);

-- name: VideoListByMovieEditionID :many
SELECT * FROM Video
WHERE ID IN (SELECT VideoID FROM MovieVideo WHERE MovieEditionID = ?);

-- name: VideoListByMovieID :many
SELECT * FROM Video
WHERE ID IN (
	SELECT VideoID FROM MovieVideo
	WHERE MovieEditionID IN (SELECT ID FROM MovieEdition WHERE MovieID = ?)
);

-- name: RenditionForStreamingListByMovieEditionID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID IN (SELECT VideoID FROM MovieVideo WHERE MovieEditionID = ?);

-- name: RenditionForStreamingListByMovieID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID IN (
	SELECT VideoID FROM MovieVideo
	WHERE MovieEditionID IN (SELECT ID FROM MovieEdition WHERE MovieID = ?)
);

-- name: ReleaseCreate :one
INSERT INTO Release
(
	Name,
	InfoHash
)
VALUES (?, ?)
RETURNING *;

-- name: ReleaseGet :one
SELECT * FROM Release WHERE ID = ?;

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
	SurroundAudio,
	Priority
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
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

-- name: RenditionForStreamingDeleteByVideoID :exec
DELETE FROM RenditionForStreaming WHERE VideoID = ?;

-- name: RenditionForStreamingUpdateEncode :one
UPDATE RenditionForStreaming
SET Hash = ?, Playlist = ?
WHERE ID = ?
RETURNING *;

-- name: RenditionForStreamingNextUnencoded :one
SELECT * FROM RenditionForStreaming
WHERE VideoID = ? AND Hash = ''
ORDER BY Priority ASC LIMIT 1;

-- name: RenditionForStreamingCountUnencoded :one
SELECT COUNT(*) FROM RenditionForStreaming
WHERE VideoID = ? AND Hash = '';

-- name: RenditionForStreamingListEncodedByVideoID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID = ? AND Hash != '';

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
	ID,
	Slug,
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
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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

-- name: SeriesGetBySlug :one
SELECT * FROM Series WHERE Slug = ?;

-- name: SeriesSlugExists :one
SELECT COUNT(*) FROM Series WHERE Slug = ?;

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
INSERT INTO Task (Type, Args, Priority, Queue) VALUES (?, ?, ?, ?)
RETURNING *;

-- name: TaskDelete :exec
DELETE FROM Task WHERE ID = ?;

-- name: TaskGet :one
SELECT * FROM Task WHERE ID = ?;

-- name: TaskList :many
SELECT * FROM Task
WHERE Running = 0;

-- name: TaskReschedule :one
UPDATE Task SET
	Failures = ?,
	NextRun = ?,
	FailureDesc = ?
WHERE ID = ?
RETURNING *;

-- name: TaskSetNextRun :exec
UPDATE Task SET NextRun = ? WHERE ID = ?;

-- name: TaskNext :one
SELECT * FROM Task
WHERE Queue = ? AND Running = 0 AND NextRun <= ?
ORDER BY Priority, ID
LIMIT 1;

-- name: TaskLock :one
UPDATE Task SET Running = 1
WHERE ID = ? AND Running = 0
RETURNING *;

-- name: TaskUnlock :exec
UPDATE Task SET Running = 0 WHERE ID = ?;

-- name: TaskResetRunning :exec
UPDATE Task SET Running = 0 WHERE Running = 1;

-- name: TaskSaveOneOff :exec
UPDATE Task SET
	Failures = ?,
	FailureDesc = ?
WHERE ID = ?;

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

-- name: VideoListByEditionID :many
SELECT * FROM Video
WHERE ID IN (
	SELECT VideoID FROM EpisodeVideo
	WHERE EpisodeID IN (
		SELECT EpisodeID FROM SeasonEpisode
		WHERE SeasonID IN (SELECT ID FROM Season WHERE EditionID = ?)
	)
);

-- name: VideoListBySeriesID :many
SELECT * FROM Video
WHERE ID IN (
	SELECT VideoID FROM EpisodeVideo
	WHERE EpisodeID IN (
		SELECT EpisodeID FROM SeasonEpisode
		WHERE SeasonID IN (
			SELECT ID FROM Season
			WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
		)
	)
);

-- name: VideoUpdateOriginalHash :one
UPDATE Video SET OriginalHash = ? WHERE ID = ?
RETURNING *;

-- name: VideoUpdateMVPlaylist :one
UPDATE Video SET MVPlaylist = ? WHERE ID = ?
RETURNING *;

-- name: SettingListByGroup :many
SELECT * FROM Setting WHERE "Group" = ?;

-- name: SettingSet :exec
INSERT INTO Setting (Key, "Group", Value) VALUES (?, ?, ?)
ON CONFLICT (Key) DO UPDATE SET Value = ?3;
