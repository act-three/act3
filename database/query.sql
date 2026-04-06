-- keep sorted by name

-- name: AudioTrackCreate :one
INSERT INTO AudioTrack (
	VideoID, StreamIndex, Language, Title,
	Channels, ChannelLayout, Codec
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: AudioTrackDeleteByVideoID :exec
DELETE FROM AudioTrack WHERE VideoID = ?;

-- name: AudioTrackListByVideoID :many
SELECT * FROM AudioTrack
WHERE VideoID = ?
ORDER BY StreamIndex;

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

-- name: CollectionCreate :one
INSERT INTO Collection (Slug, Title)
VALUES (?, ?)
RETURNING *;

-- name: CollectionGet :one
SELECT * FROM Collection WHERE ID = ?;

-- name: CollectionGetBySlug :one
SELECT * FROM Collection WHERE Slug = ?;

-- name: CollectionGetStats :one
SELECT COUNT(*) AS ItemCount, CAST(COALESCE(SUM(Runtime), 0) AS INTEGER) AS RuntimeMinutes FROM (
	SELECT MovieEdition.Runtime FROM MovieEdition
	WHERE MovieEdition.Slug = '' AND MovieEdition.MovieID IN (
		SELECT CollectionMovie.MovieID FROM CollectionMovie WHERE CollectionMovie.CollectionID = sqlc.arg(ID)
	)
	UNION ALL
	SELECT Episode.Runtime FROM Episode
	WHERE Episode.Type != 'insignificant_special' AND Episode.ID IN (
		SELECT SeasonEpisode.EpisodeID FROM SeasonEpisode WHERE SeasonEpisode.SeasonID IN (
			SELECT Season.ID FROM Season WHERE Season.EditionID IN (
				SELECT SeriesEdition.ID FROM SeriesEdition WHERE SeriesEdition.Slug = '' AND SeriesEdition.SeriesID IN (
					SELECT CollectionSeries.SeriesID FROM CollectionSeries WHERE CollectionSeries.CollectionID = sqlc.arg(ID)
				)
			)
		)
	)
);

-- name: CollectionList :many
SELECT * FROM Collection
ORDER BY Title;

-- name: CollectionMovieAdd :exec
INSERT INTO CollectionMovie (CollectionID, MovieID)
VALUES (?, ?);

-- name: CollectionMovieDelete :exec
DELETE FROM CollectionMovie WHERE CollectionID = ? AND MovieID = ?;

-- name: CollectionMovieList :many
SELECT m.* FROM Movie m
JOIN CollectionMovie cm ON cm.MovieID = m.ID
WHERE cm.CollectionID = ?
ORDER BY m.Slug;

-- name: CollectionSeriesAdd :exec
INSERT INTO CollectionSeries (CollectionID, SeriesID)
VALUES (?, ?);

-- name: CollectionSeriesDelete :exec
DELETE FROM CollectionSeries WHERE CollectionID = ? AND SeriesID = ?;

-- name: CollectionSeriesList :many
SELECT s.* FROM Series s
JOIN CollectionSeries cs ON cs.SeriesID = s.ID
WHERE cs.CollectionID = ?
ORDER BY s.Title;

-- name: CollectionSetBannerID :exec
UPDATE Collection SET BannerID = ? WHERE ID = ?;

-- name: CollectionSetSlug :exec
UPDATE Collection SET Slug = ? WHERE ID = ?;

-- name: CollectionSetTitle :exec
UPDATE Collection SET Title = ? WHERE ID = ?;

-- name: DownloadCreate :one
INSERT INTO Download
(
	InfoHash,
	State,
	Title,
	Torrent
)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: DownloadGet :one
SELECT * FROM Download WHERE InfoHash = ?;

-- name: DownloadList :many
SELECT * FROM Download
ORDER BY CreatedAt DESC;

-- name: DownloadListByPlanMovieEditionID :many
SELECT * FROM Download
WHERE PlanMovieEditionID = ?
ORDER BY CreatedAt DESC;

-- name: DownloadListByPlanSeriesEditionID :many
SELECT * FROM Download
WHERE PlanSeriesEditionID = ?
ORDER BY CreatedAt DESC;

-- name: DownloadListInfoHashesDownloading :many
SELECT InfoHash FROM Download
WHERE State = 'downloading';

-- name: DownloadPlanCountActiveByInfoHash :one
SELECT COUNT(*) FROM DownloadPlan WHERE InfoHash = ? AND State != 'imported';

-- name: DownloadPlanCountByInfoHash :one
SELECT COUNT(*) FROM DownloadPlan WHERE InfoHash = ?;

-- name: DownloadPlanCreate :exec
INSERT INTO DownloadPlan (InfoHash, Path, EpisodeID, MovieEditionID)
VALUES (?, ?, ?, ?);

-- name: DownloadPlanDeleteByInfoHash :exec
DELETE FROM DownloadPlan WHERE InfoHash = ?;

-- name: DownloadPlanListByInfoHash :many
SELECT * FROM DownloadPlan WHERE InfoHash = ?;

-- name: DownloadPlanUpdateState :exec
UPDATE DownloadPlan SET State = ? WHERE InfoHash = ? AND Path = ?;

-- name: DownloadUpdateAutoImport :one
UPDATE Download SET AutoImport = ? WHERE InfoHash = ? RETURNING *;

-- name: DownloadUpdateError :one
UPDATE Download SET State = 'error', Error = ? WHERE InfoHash = ? RETURNING *;

-- name: DownloadUpdatePlanMovie :one
UPDATE Download SET
	PlanMovieEditionID = ?
WHERE InfoHash = ?
RETURNING *;

-- name: DownloadUpdatePlanSeries :one
UPDATE Download SET
	PlanSeriesEditionID = ?
WHERE InfoHash = ?
RETURNING *;

-- name: DownloadUpdateProgress :one
UPDATE Download SET
	State = ?,
	Progress = ?,
	Error = ''
WHERE InfoHash = ? RETURNING *;

-- name: EpisodeAirdateSet :exec
UPDATE Episode SET Airdate = ? WHERE ID = ?;

-- name: EpisodeCreate :one
INSERT INTO Episode
(
	Title,
	Summary,
	Type,
	Airdate,
	Runtime
)
VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: EpisodeGet :one
SELECT * FROM Episode WHERE ID = ?;

-- name: EpisodeListByEditionID :many
SELECT * FROM Episode
WHERE ID IN (
	SELECT EpisodeID FROM SeasonEpisode WHERE EditionID = ?
)
ORDER BY ID;

-- name: EpisodeListBySeriesID :many
SELECT * FROM Episode
WHERE ID IN (
	SELECT EpisodeID FROM SeasonEpisode
	WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
)
ORDER BY ID;

-- name: EpisodeSummarySet :exec
UPDATE Episode SET Summary = ? WHERE ID = ?;

-- name: EpisodeThumbnailIDSet :exec
UPDATE Episode SET ThumbnailID = ? WHERE ID = ?;

-- name: EpisodeTitleSet :exec
UPDATE Episode SET Title = ? WHERE ID = ?;

-- name: EpisodeTypeSet :exec
UPDATE Episode SET Type = ? WHERE ID = ?;

-- name: EpisodeVideoCreate :one
INSERT INTO EpisodeVideo (EpisodeID, VideoID)
VALUES (?, ?)
RETURNING *;

-- name: EpisodeVideoListByEditionID :many
SELECT * FROM EpisodeVideo
WHERE EpisodeID IN (
	SELECT EpisodeID FROM SeasonEpisode WHERE EditionID = ?
);

-- name: EpisodeVideoListBySeriesID :many
SELECT * FROM EpisodeVideo
WHERE EpisodeID IN (
	SELECT EpisodeID FROM SeasonEpisode
	WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
);

-- name: EpisodeVideoListByVideoID :many
SELECT * FROM EpisodeVideo
WHERE VideoID = ?;

-- name: MovieCreate :one
INSERT INTO Movie (ID, Slug, TMDBID, IMDBID)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: MovieEditionCreate :one
INSERT INTO MovieEdition (Title, Label, Slug, MovieID, Summary, Year, Runtime)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: MovieEditionGet :one
SELECT * FROM MovieEdition WHERE ID = ?;

-- name: MovieEditionGetDefault :one
SELECT * FROM MovieEdition WHERE MovieID = ? AND Slug = '';

-- name: MovieEditionLabelSet :exec
UPDATE MovieEdition SET Label = ? WHERE ID = ?;

-- name: MovieEditionListByMovieID :many
SELECT * FROM MovieEdition WHERE MovieID = ?;

-- name: MovieEditionListDefault :many
SELECT * FROM MovieEdition WHERE Slug = '';

-- name: MovieEditionPosterIDSet :exec
UPDATE MovieEdition SET PosterID = ? WHERE ID = ?;

-- name: MovieEditionRuntimeSet :exec
UPDATE MovieEdition SET Runtime = ? WHERE ID = ?;

-- name: MovieEditionSlugExists :one
SELECT COUNT(*) FROM MovieEdition WHERE MovieID = ? AND Slug = ?;

-- name: MovieEditionSlugSet :exec
UPDATE MovieEdition SET Slug = ? WHERE ID = ?;

-- name: MovieEditionSummarySet :exec
UPDATE MovieEdition SET Summary = ? WHERE ID = ?;

-- name: MovieEditionTitleSet :exec
UPDATE MovieEdition SET Title = ? WHERE ID = ?;

-- name: MovieEditionYearSet :exec
UPDATE MovieEdition SET Year = ? WHERE ID = ?;

-- name: MovieGet :one
SELECT * FROM Movie WHERE ID = ?;

-- name: MovieGetByEditionID :one
SELECT * FROM Movie
WHERE ID IN (SELECT MovieID FROM MovieEdition WHERE MovieEdition.ID = ?);

-- name: MovieGetBySlug :one
SELECT * FROM Movie WHERE Slug = ?;

-- name: MovieList :many
SELECT * FROM Movie
ORDER BY Slug;

-- name: MovieListByTMDBID :many
SELECT * FROM Movie WHERE TMDBID IN (sqlc.slice(ids));

-- name: MovieSlugExists :one
SELECT COUNT(*) FROM Movie WHERE Slug = ?;

-- name: MovieSlugSet :exec
UPDATE Movie SET Slug = ? WHERE ID = ?;

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

-- name: RenditionForStreamingCountUnencoded :one
SELECT COUNT(*) FROM RenditionForStreaming
WHERE VideoID = ? AND Hash = '';

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

-- name: RenditionForStreamingDeleteByVideoID :exec
DELETE FROM RenditionForStreaming WHERE VideoID = ?;

-- name: RenditionForStreamingGet :one
SELECT * FROM RenditionForStreaming
WHERE ID = ?;

-- name: RenditionForStreamingListByMovieEditionID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID IN (SELECT VideoID FROM MovieVideo WHERE MovieEditionID = ?);

-- name: RenditionForStreamingListByMovieID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID IN (
	SELECT VideoID FROM MovieVideo
	WHERE MovieEditionID IN (SELECT ID FROM MovieEdition WHERE MovieID = ?)
);

-- name: RenditionForStreamingListByVideoID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID  IN (SELECT VideoID FROM EpisodeVideo WHERE EpisodeID = ?);

-- name: RenditionForStreamingListDirectByVideoID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID = ?;

-- name: RenditionForStreamingListEncodedByVideoID :many
SELECT * FROM RenditionForStreaming
WHERE VideoID = ? AND Hash != '';

-- name: RenditionForStreamingNextUnencoded :one
SELECT * FROM RenditionForStreaming
WHERE VideoID = ? AND Hash = ''
ORDER BY Priority ASC LIMIT 1;

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
	Title,
	Number
)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: SeasonEpisodeCreate :exec
INSERT INTO SeasonEpisode (EditionID, SeasonID, EpisodeID, SortKey, Label, Number, Slug)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: SeasonEpisodeDelete :exec
DELETE FROM SeasonEpisode WHERE SeasonID = ? AND EpisodeID = ?;

-- name: SeasonEpisodeDeleteBySeasonID :exec
DELETE FROM SeasonEpisode WHERE SeasonID = ?;

-- name: SeasonEpisodeGet :one
SELECT * FROM SeasonEpisode WHERE SeasonID = ? AND EpisodeID = ?;

-- name: SeasonEpisodeGetBySlug :one
SELECT * FROM SeasonEpisode WHERE EditionID = ? AND Slug = ?;

-- name: SeasonEpisodeListByEditionID :many
SELECT * FROM SeasonEpisode
WHERE EditionID = ?
ORDER BY SortKey;

-- name: SeasonEpisodeListByEpisodeID :many
SELECT * FROM SeasonEpisode WHERE EpisodeID = ?;

-- name: SeasonEpisodeListBySeasonID :many
SELECT * FROM SeasonEpisode
WHERE SeasonID = ?
ORDER BY SortKey;

-- name: SeasonEpisodeListBySeriesID :many
SELECT * FROM SeasonEpisode
WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
ORDER BY SortKey;

-- name: SeasonEpisodeNumberingSet :exec
UPDATE SeasonEpisode SET Number = ?, Label = ?, Slug = ? WHERE SeasonID = ? AND EpisodeID = ?;

-- name: SeasonEpisodeSlugExists :one
SELECT COUNT(*) FROM SeasonEpisode WHERE EditionID = ? AND Slug = ?;

-- name: SeasonEpisodeSlugSet :exec
UPDATE SeasonEpisode SET Slug = ? WHERE SeasonID = ? AND EpisodeID = ?;

-- name: SeasonGet :one
SELECT * FROM Season WHERE ID = ?;

-- name: SeasonListByEditionID :many
SELECT * FROM Season WHERE EditionID = ?
ORDER BY SortKey;

-- name: SeasonListBySeriesID :many
SELECT * FROM Season
WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
ORDER BY SortKey;

-- name: SeasonTitleSet :exec
UPDATE Season SET Title = ? WHERE ID = ?;

-- name: SeriesCreate :one
INSERT INTO Series
(
	ID,
	Slug,
	Title,
	Status,
	PremieredOn,
	EndedOn,
	TVmazeID,
	IMDBID,
	TVDBID,
	TVRageID
)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: SeriesEditionCreate :one
INSERT INTO SeriesEdition (Label, Slug, SeriesID, Summary)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: SeriesEditionGet :one
SELECT * FROM SeriesEdition WHERE ID = ?;

-- name: SeriesEditionGetBySlug :one
SELECT SeriesEdition.* FROM SeriesEdition
JOIN Series ON Series.ID = SeriesEdition.SeriesID
WHERE Series.Slug = sqlc.arg(SeriesSlug) AND SeriesEdition.Slug = sqlc.arg(EditionSlug);

-- name: SeriesEditionLabelSet :exec
UPDATE SeriesEdition SET Label = ? WHERE ID = ?;

-- name: SeriesEditionListBySeriesID :many
SELECT * FROM SeriesEdition WHERE SeriesID = ?;

-- name: SeriesEditionListDefault :many
SELECT * FROM SeriesEdition WHERE Slug = '';

-- name: SeriesEditionPosterIDSet :exec
UPDATE SeriesEdition SET PosterID = ? WHERE ID = ?;

-- name: SeriesEditionSlugExists :one
SELECT COUNT(*) FROM SeriesEdition WHERE SeriesID = ? AND Slug = ?;

-- name: SeriesEditionSlugSet :exec
UPDATE SeriesEdition SET Slug = ? WHERE ID = ?;

-- name: SeriesEditionSummarySet :exec
UPDATE SeriesEdition SET Summary = ? WHERE ID = ?;

-- name: SeriesGet :one
SELECT * FROM Series WHERE ID = ?;

-- name: SeriesGetByEditionID :one
SELECT * FROM Series
WHERE ID IN (SELECT SeriesID FROM SeriesEdition WHERE SeriesEdition.ID = ?);

-- name: SeriesGetBySlug :one
SELECT * FROM Series WHERE Slug = ?;

-- name: SeriesGetByTVmazeID :one
SELECT * FROM Series WHERE TVmazeID = ?;

-- name: SeriesList :many
SELECT * FROM Series;

-- name: SeriesListByTVmazeID :many
SELECT * FROM Series WHERE TVmazeID IN (sqlc.slice(ids));

-- name: SeriesSlugExists :one
SELECT COUNT(*) FROM Series WHERE Slug = ?;

-- name: SeriesSlugSet :exec
UPDATE Series SET Slug = ? WHERE ID = ?;

-- name: SeriesTitleSet :exec
UPDATE Series SET Title = ? WHERE ID = ?;

-- name: SettingListByGroup :many
SELECT * FROM Setting WHERE "Group" = ?;

-- name: SettingSet :exec
INSERT INTO Setting (Key, "Group", Value) VALUES (?, ?, ?)
ON CONFLICT (Key) DO UPDATE SET Value = ?3;

-- name: SlugCreate :exec
INSERT INTO Slug (Slug, Kind, Target) VALUES (?, ?, ?);

-- name: SlugDelete :exec
DELETE FROM Slug WHERE Target = ?;

-- name: SlugExists :one
SELECT COUNT(*) FROM Slug WHERE Slug = ?;

-- name: SlugGet :one
SELECT * FROM Slug WHERE Slug = ?;

-- name: SlugUpdate :exec
UPDATE Slug SET Slug = ? WHERE Target = ?;

-- name: TaskCountError :one
SELECT COUNT(*) FROM Task WHERE Failures > 0;

-- name: TaskCountQueued :one
SELECT COUNT(*) FROM Task WHERE Running = 0;

-- name: TaskCreate :one
INSERT INTO Task (Type, Args, Priority, Queue, NextRun) VALUES (?, ?, ?, ?, ?)
RETURNING *;

-- name: TaskDelete :exec
DELETE FROM Task WHERE ID = ?;

-- name: TaskGet :one
SELECT * FROM Task WHERE ID = ?;

-- name: TaskList :many
SELECT * FROM Task
WHERE Running = 0;

-- name: TaskLock :one
UPDATE Task SET Running = 1
WHERE ID = ? AND Running = 0
RETURNING *;

-- name: TaskNext :one
SELECT * FROM Task
WHERE Queue = ? AND Running = 0 AND NextRun <= ?
ORDER BY Priority, ID
LIMIT 1;

-- name: TaskReschedule :one
UPDATE Task SET
	Failures = ?,
	NextRun = ?,
	FailureDesc = ?
WHERE ID = ?
RETURNING *;

-- name: TaskResetRunning :exec
UPDATE Task SET Running = 0 WHERE Running = 1;

-- name: TaskSaveOneOff :exec
UPDATE Task SET
	Failures = ?,
	FailureDesc = ?
WHERE ID = ?;

-- name: TaskSetNextRun :exec
UPDATE Task SET NextRun = ? WHERE ID = ?;

-- name: TaskUnlock :exec
UPDATE Task SET Running = 0 WHERE ID = ?;

-- name: VideoCreate :one
INSERT INTO Video
(
	InfoHash,
	Name
)
VALUES (?, ?)
RETURNING *;

-- name: VideoGet :one
SELECT * FROM Video WHERE ID = ?;

-- name: VideoGetByName :one
SELECT * FROM Video WHERE InfoHash = ? AND Name = ?;

-- name: VideoListByEditionID :many
SELECT * FROM Video
WHERE ID IN (
	SELECT VideoID FROM EpisodeVideo
	WHERE EpisodeID IN (
		SELECT EpisodeID FROM SeasonEpisode WHERE EditionID = ?
	)
);

-- name: VideoListByEpisodeID :many
SELECT * FROM Video
WHERE ID IN (SELECT VideoID FROM EpisodeVideo WHERE EpisodeID = ?);

-- name: VideoListByMovieEditionID :many
SELECT * FROM Video
WHERE ID IN (SELECT VideoID FROM MovieVideo WHERE MovieEditionID = ?);

-- name: VideoListByMovieID :many
SELECT * FROM Video
WHERE ID IN (
	SELECT VideoID FROM MovieVideo
	WHERE MovieEditionID IN (SELECT ID FROM MovieEdition WHERE MovieID = ?)
);

-- name: VideoListBySeriesID :many
SELECT * FROM Video
WHERE ID IN (
	SELECT VideoID FROM EpisodeVideo
	WHERE EpisodeID IN (
		SELECT EpisodeID FROM SeasonEpisode
		WHERE EditionID IN (SELECT ID FROM SeriesEdition WHERE SeriesID = ?)
	)
);

-- name: VideoUpdateMVPlaylist :one
UPDATE Video SET MVPlaylist = ? WHERE ID = ?
RETURNING *;

-- name: VideoUpdateOriginalHash :one
UPDATE Video SET OriginalHash = ? WHERE ID = ?
RETURNING *;
