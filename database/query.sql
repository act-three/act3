-- keep sorted by name

-- name: AudioTrackCreate :one
INSERT INTO AudioTrack (
	VideoID, StreamIndex, Language, Title,
	Channels, ChannelLayout, Codec
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: AudioTrackDeleteByVideoID :exec
DELETE FROM AudioTrack WHERE VideoID = ?;

-- name: AudioTrackDeleteByVideoIDList :exec
DELETE FROM AudioTrack WHERE VideoID IN (sqlc.slice(ids));

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

-- name: CollectionBannerIDSet :exec
UPDATE Collection SET BannerID = ? WHERE ID = ?;

-- name: CollectionCreate :one
INSERT INTO Collection (Slug, Title)
VALUES (?, ?)
RETURNING *;

-- name: CollectionGet :one
SELECT * FROM Collection WHERE ID = ?;

-- name: CollectionGetBySlug :one
SELECT * FROM Collection WHERE Slug = ? AND DeletedAt IS NULL;

-- name: CollectionGetStats :one
SELECT COUNT(*) AS ItemCount, CAST(COALESCE(SUM(Runtime), 0) AS INTEGER) AS RuntimeMinutes FROM (
	SELECT MovieEdition.Runtime FROM MovieEdition
	WHERE MovieEdition.DeletedAt IS NULL
	AND MovieEdition.Slug = '' AND MovieEdition.MovieID IN (
		SELECT CollectionMovie.MovieID FROM CollectionMovie
		WHERE CollectionMovie.DeletedAt IS NULL
		AND CollectionMovie.CollectionID = sqlc.arg(ID)
		AND CollectionMovie.MovieID IN (SELECT ID FROM Movie WHERE DeletedAt IS NULL)
	)
	UNION ALL
	SELECT Episode.Runtime FROM Episode
	WHERE Episode.DeletedAt IS NULL
	AND Episode.Type != 'insignificant_special' AND Episode.ID IN (
		SELECT SeasonEpisode.EpisodeID FROM SeasonEpisode
		WHERE SeasonEpisode.DeletedAt IS NULL
		AND SeasonEpisode.SeasonID IN (
			SELECT Season.ID FROM Season
			WHERE Season.DeletedAt IS NULL
			AND Season.EditionID IN (
				SELECT SeriesEdition.ID FROM SeriesEdition
				WHERE SeriesEdition.DeletedAt IS NULL
				AND SeriesEdition.Slug = '' AND SeriesEdition.SeriesID IN (
					SELECT CollectionSeries.SeriesID FROM CollectionSeries
					WHERE CollectionSeries.DeletedAt IS NULL
					AND CollectionSeries.CollectionID = sqlc.arg(ID)
					AND CollectionSeries.SeriesID IN (SELECT ID FROM Series WHERE DeletedAt IS NULL)
				)
			)
		)
	)
);

-- name: CollectionList :many
SELECT * FROM Collection
WHERE DeletedAt IS NULL
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
AND cm.DeletedAt IS NULL
AND m.DeletedAt IS NULL
ORDER BY m.Slug;

-- name: CollectionMoviePurgeByCascade :exec
DELETE FROM CollectionMovie
WHERE CollectionID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
   OR MovieID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: CollectionMovieRestoreByCascade :exec
UPDATE CollectionMovie SET DeletedAt = NULL
WHERE DeletedAt IS NOT NULL
AND (CollectionID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
  OR MovieID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID)))
AND CollectionID IN (SELECT ID FROM Collection WHERE DeletedAt IS NULL)
AND MovieID IN (SELECT ID FROM Movie WHERE DeletedAt IS NULL);

-- name: CollectionMovieSoftDelete :exec
UPDATE CollectionMovie
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE CollectionID = sqlc.arg(CollectionID) AND MovieID = sqlc.arg(MovieID) AND DeletedAt IS NULL;

-- name: CollectionMovieSoftDeleteByCollectionID :exec
UPDATE CollectionMovie
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE CollectionID = sqlc.arg(CollectionID) AND DeletedAt IS NULL;

-- name: CollectionMovieSoftDeleteByMovieID :exec
UPDATE CollectionMovie
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE MovieID = sqlc.arg(MovieID) AND DeletedAt IS NULL;

-- name: CollectionPurgeByCascade :exec
DELETE FROM Collection
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: CollectionRestore :exec
UPDATE Collection SET DeletedAt = NULL
WHERE ID = ?;

-- name: CollectionRestoreByCascade :exec
UPDATE Collection SET DeletedAt = NULL
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: CollectionSeriesAdd :exec
INSERT INTO CollectionSeries (CollectionID, SeriesID)
VALUES (?, ?);

-- name: CollectionSeriesDelete :exec
DELETE FROM CollectionSeries WHERE CollectionID = ? AND SeriesID = ?;

-- name: CollectionSeriesList :many
SELECT s.* FROM Series s
JOIN CollectionSeries cs ON cs.SeriesID = s.ID
WHERE cs.CollectionID = ?
AND cs.DeletedAt IS NULL
AND s.DeletedAt IS NULL
ORDER BY s.Title;

-- name: CollectionSeriesPurgeByCascade :exec
DELETE FROM CollectionSeries
WHERE CollectionID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
   OR SeriesID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: CollectionSeriesRestoreByCascade :exec
UPDATE CollectionSeries SET DeletedAt = NULL
WHERE DeletedAt IS NOT NULL
AND (CollectionID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
  OR SeriesID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID)))
AND CollectionID IN (SELECT ID FROM Collection WHERE DeletedAt IS NULL)
AND SeriesID IN (SELECT ID FROM Series WHERE DeletedAt IS NULL);

-- name: CollectionSeriesSoftDelete :exec
UPDATE CollectionSeries
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE CollectionID = sqlc.arg(CollectionID) AND SeriesID = sqlc.arg(SeriesID) AND DeletedAt IS NULL;

-- name: CollectionSeriesSoftDeleteByCollectionID :exec
UPDATE CollectionSeries
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE CollectionID = sqlc.arg(CollectionID) AND DeletedAt IS NULL;

-- name: CollectionSeriesSoftDeleteBySeriesID :exec
UPDATE CollectionSeries
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE SeriesID = sqlc.arg(SeriesID) AND DeletedAt IS NULL;

-- name: CollectionSetSlug :exec
UPDATE Collection SET Slug = ? WHERE ID = ?;

-- name: CollectionSetTitle :exec
UPDATE Collection SET Title = ? WHERE ID = ?;

-- name: CollectionSoftDelete :exec
UPDATE Collection
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE ID = sqlc.arg(ID) AND DeletedAt IS NULL;

-- DownloadBumpActivity bumps LastActivityAt on the live Download with
-- the given InfoHash. Callers use it after mutating an EpisodeVideo or
-- MovieVideo junction owned by the Download, to reset its auto-trash
-- timer while the user is actively curating the download's videos.
-- name: DownloadBumpActivity :exec
UPDATE Download SET LastActivityAt = sqlc.arg(LastActivityAt)
WHERE InfoHash = sqlc.arg(InfoHash) AND DeletedAt IS NULL;

-- name: DownloadCreate :one
INSERT INTO Download
(
	InfoHash,
	State,
	Title,
	Torrent,
	SeriesEditionID,
	MovieEditionID
)
VALUES (?, ?, ?, ?, ?, ?)
RETURNING *;

-- DownloadGet returns a Download row regardless of DeletedAt state.
-- DownloadCreate uses this to spot trashed duplicates for restore; other
-- callers (polling, planning) only reach here for live Downloads via the
-- Transmission-synchronised active-InfoHash set.
-- name: DownloadGet :one
SELECT * FROM Download WHERE InfoHash = ?;

-- name: DownloadList :many
SELECT * FROM Download
WHERE DeletedAt IS NULL
ORDER BY CreatedAt DESC;

-- DownloadListAutoTrashCandidates returns InfoHashes of live Downloads
-- in terminal states whose last activity is older than the threshold.
-- Active-state Downloads (queued/downloading/downloaded) are never
-- auto-trashed: those are still in flight and belong to the user until
-- they explicitly delete them.
-- name: DownloadListAutoTrashCandidates :many
SELECT InfoHash FROM Download
WHERE DeletedAt IS NULL
AND State IN ('imported', 'error')
AND LastActivityAt < ?;

-- name: DownloadListByMovieEditionID :many
SELECT * FROM Download
WHERE MovieEditionID = ? AND DeletedAt IS NULL
ORDER BY CreatedAt DESC;

-- name: DownloadListBySeriesEditionID :many
SELECT * FROM Download
WHERE SeriesEditionID = ? AND DeletedAt IS NULL
ORDER BY CreatedAt DESC;

-- name: DownloadListInfoHashesActive :many
SELECT InfoHash FROM Download
WHERE State IN ('downloading', 'downloaded') AND DeletedAt IS NULL;

-- name: DownloadPurgeByCascade :exec
DELETE FROM Download
WHERE InfoHash IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: DownloadRestore :exec
UPDATE Download SET DeletedAt = NULL, LastActivityAt = sqlc.arg(LastActivityAt)
WHERE InfoHash = sqlc.arg(InfoHash);

-- name: DownloadRestoreByCascade :exec
UPDATE Download SET DeletedAt = NULL
WHERE InfoHash IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: DownloadSoftDelete :exec
UPDATE Download SET DeletedAt = sqlc.arg(DeletedAt)
WHERE InfoHash = sqlc.arg(InfoHash) AND DeletedAt IS NULL;

-- name: DownloadUpdateAutoImport :one
UPDATE Download SET AutoImport = ?, LastActivityAt = sqlc.arg(LastActivityAt) WHERE InfoHash = sqlc.arg(InfoHash) RETURNING *;

-- name: DownloadUpdateError :one
UPDATE Download SET State = 'error', Error = ?, LastActivityAt = sqlc.arg(LastActivityAt) WHERE InfoHash = sqlc.arg(InfoHash) RETURNING *;

-- name: DownloadUpdateProgress :one
UPDATE Download SET
	State = ?,
	Progress = ?,
	Error = '',
	LastActivityAt = sqlc.arg(LastActivityAt)
WHERE InfoHash = sqlc.arg(InfoHash) RETURNING *;

-- name: DownloadUpdateTargeting :exec
UPDATE Download SET SeriesEditionID = ?, MovieEditionID = ? WHERE InfoHash = ?;

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
SELECT * FROM Episode WHERE ID = ? AND DeletedAt IS NULL;

-- EpisodeGetAny returns an Episode regardless of trash state.
-- Used by the trash/restore code to inspect rows that may be soft-deleted.
-- name: EpisodeGetAny :one
SELECT * FROM Episode WHERE ID = ?;

-- name: EpisodeListByEditionID :many
SELECT * FROM Episode
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT EpisodeID FROM SeasonEpisode
	WHERE DeletedAt IS NULL AND EditionID = ?
)
ORDER BY ID;

-- name: EpisodeListBySeriesID :many
SELECT * FROM Episode
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT EpisodeID FROM SeasonEpisode
	WHERE DeletedAt IS NULL
	AND EditionID IN (SELECT ID FROM SeriesEdition WHERE DeletedAt IS NULL AND SeriesID = ?)
)
ORDER BY ID;

-- EpisodeListOrphans returns live Episode IDs with no live
-- SeasonEpisode junctions. Called after soft-deleting a cascade's
-- junctions to reap episodes the cascade just stranded.
-- name: EpisodeListOrphans :many
SELECT ep.ID FROM Episode ep
WHERE ep.DeletedAt IS NULL
AND NOT EXISTS (
	SELECT 1 FROM SeasonEpisode se
	WHERE se.EpisodeID = ep.ID AND se.DeletedAt IS NULL
);

-- name: EpisodePurgeByCascade :exec
DELETE FROM Episode
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: EpisodeRestore :exec
UPDATE Episode SET DeletedAt = NULL
WHERE ID = ?;

-- name: EpisodeRestoreByCascade :exec
UPDATE Episode SET DeletedAt = NULL
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: EpisodeSoftDelete :exec
UPDATE Episode
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE ID = sqlc.arg(ID) AND DeletedAt IS NULL;

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

-- name: EpisodeVideoDelete :exec
DELETE FROM EpisodeVideo WHERE EpisodeID = ? AND VideoID = ?;

-- name: EpisodeVideoDeleteByVideoID :exec
DELETE FROM EpisodeVideo WHERE VideoID = ?;

-- name: EpisodeVideoDistinctEpisodesByVideo :many
SELECT DISTINCT EpisodeID FROM EpisodeVideo WHERE VideoID = ?;

-- name: EpisodeVideoEnsure :exec
INSERT OR IGNORE INTO EpisodeVideo (EpisodeID, VideoID)
VALUES (?, ?);

-- name: EpisodeVideoListByEditionID :many
SELECT * FROM EpisodeVideo
WHERE DeletedAt IS NULL
AND EpisodeID IN (
	SELECT EpisodeID FROM SeasonEpisode
	WHERE DeletedAt IS NULL AND EditionID = ?
);

-- name: EpisodeVideoListByInfoHash :many
SELECT * FROM EpisodeVideo
WHERE VideoID IN (SELECT ID FROM Video WHERE InfoHash = ?);

-- name: EpisodeVideoListBySeriesID :many
SELECT * FROM EpisodeVideo
WHERE DeletedAt IS NULL
AND EpisodeID IN (
	SELECT EpisodeID FROM SeasonEpisode
	WHERE DeletedAt IS NULL
	AND EditionID IN (SELECT ID FROM SeriesEdition WHERE DeletedAt IS NULL AND SeriesID = ?)
);

-- name: EpisodeVideoListByVideoID :many
SELECT * FROM EpisodeVideo
WHERE VideoID = ?;

-- name: EpisodeVideoPurgeByCascade :exec
DELETE FROM EpisodeVideo
WHERE EpisodeID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
   OR VideoID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- EpisodeVideoReassign re-points live junctions from one Video to
-- another, used during duplicate-content merge. Existing (EpisodeID,
-- ToVideoID) junctions are kept via INSERT OR IGNORE; callers should
-- precede with EpisodeVideoRestoreForReassign so that any soft-
-- deleted winner junctions at conflict points are revived first, and
-- follow with EpisodeVideoDeleteByVideoID on the from-side to remove
-- stale rows.
-- name: EpisodeVideoReassign :exec
INSERT OR IGNORE INTO EpisodeVideo (EpisodeID, VideoID)
SELECT src.EpisodeID, sqlc.arg(ToVideoID) FROM EpisodeVideo AS src
WHERE src.VideoID = sqlc.arg(FromVideoID) AND src.DeletedAt IS NULL;

-- name: EpisodeVideoRestoreByCascade :exec
UPDATE EpisodeVideo SET DeletedAt = NULL
WHERE DeletedAt IS NOT NULL
AND (EpisodeID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
  OR VideoID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID)))
AND EpisodeID IN (SELECT ID FROM Episode WHERE DeletedAt IS NULL)
AND VideoID IN (SELECT ID FROM Video WHERE DeletedAt IS NULL);

-- EpisodeVideoRestoreForReassign clears DeletedAt on any ToVideoID
-- junctions that collide with live FromVideoID junctions. Run
-- immediately before EpisodeVideoReassign during duplicate-content
-- merge so that a loser's live junction revives a previously
-- soft-deleted winner junction for the same episode. The loser's
-- live junction represents current user intent, overriding the
-- older detach.
-- name: EpisodeVideoRestoreForReassign :exec
UPDATE EpisodeVideo SET DeletedAt = NULL
WHERE EpisodeVideo.VideoID = sqlc.arg(ToVideoID) AND EpisodeVideo.DeletedAt IS NOT NULL
AND EpisodeVideo.EpisodeID IN (
	SELECT src.EpisodeID FROM EpisodeVideo AS src
	WHERE src.VideoID = sqlc.arg(FromVideoID) AND src.DeletedAt IS NULL
);

-- name: EpisodeVideoSoftDelete :exec
UPDATE EpisodeVideo
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE EpisodeID = sqlc.arg(EpisodeID) AND VideoID = sqlc.arg(VideoID) AND DeletedAt IS NULL;

-- name: EpisodeVideoSoftDeleteByEpisodeID :exec
UPDATE EpisodeVideo
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE EpisodeID = sqlc.arg(EpisodeID) AND DeletedAt IS NULL;

-- name: EpisodeVideoSoftDeleteByVideoID :exec
UPDATE EpisodeVideo
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE VideoID = sqlc.arg(VideoID) AND DeletedAt IS NULL;

-- name: ImageCreate :one
INSERT INTO Image (OriginalKey, Type)
VALUES (?, ?)
RETURNING *;

-- name: ImageCreateWithID :exec
INSERT INTO Image (ID, OriginalKey, Type)
VALUES (?, ?, ?)
ON CONFLICT (ID) DO NOTHING;

-- name: ImageDelete :one
DELETE FROM Image WHERE ID = ? RETURNING OriginalKey;

-- name: ImageGet :one
SELECT * FROM Image WHERE ID = ?;

-- name: ImageRenditionCreate :exec
INSERT INTO ImageRendition (Key, ImageID, Type, Width, Height)
VALUES (?, ?, ?, ?, ?);

-- name: ImageRenditionDeleteByImageID :many
DELETE FROM ImageRendition WHERE ImageID = ? RETURNING Key;

-- name: ImageRenditionListByImageID :many
SELECT * FROM ImageRendition WHERE ImageID = ? ORDER BY Width;

-- name: MovieCreate :one
INSERT INTO Movie (ID, Slug, TMDBID, IMDBID)
VALUES (?, ?, ?, ?)
RETURNING *;

-- name: MovieEditionCreate :one
INSERT INTO MovieEdition (Title, Label, Slug, MovieID, Summary, ReleaseDate, Runtime)
VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- MovieEditionDefaultSuccessor returns the lex-smallest live non-default
-- edition of the given movie, for promotion when the current default is
-- trashed. Errors with sql.ErrNoRows if none exists.
-- name: MovieEditionDefaultSuccessor :one
SELECT * FROM MovieEdition
WHERE MovieID = ? AND DeletedAt IS NULL AND Slug != ''
ORDER BY ID
LIMIT 1;

-- name: MovieEditionGet :one
SELECT * FROM MovieEdition WHERE ID = ?;

-- name: MovieEditionGetDefault :one
SELECT * FROM MovieEdition WHERE MovieID = ? AND Slug = '';

-- name: MovieEditionLabelSet :exec
UPDATE MovieEdition SET Label = ? WHERE ID = ?;

-- name: MovieEditionListByDownload :many
SELECT * FROM MovieEdition
WHERE DeletedAt IS NULL
AND ID IN (SELECT MovieEditionID FROM Download);

-- name: MovieEditionListByMovieID :many
SELECT * FROM MovieEdition WHERE MovieID = ? AND DeletedAt IS NULL;

-- name: MovieEditionListDefault :many
SELECT * FROM MovieEdition WHERE Slug = '' AND DeletedAt IS NULL;

-- name: MovieEditionPosterIDSet :exec
UPDATE MovieEdition SET PosterID = ? WHERE ID = ?;

-- name: MovieEditionPurgeByCascade :exec
DELETE FROM MovieEdition
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: MovieEditionReleaseDateSet :exec
UPDATE MovieEdition SET ReleaseDate = ? WHERE ID = ?;

-- name: MovieEditionRestore :exec
UPDATE MovieEdition SET DeletedAt = NULL
WHERE ID = ?;

-- name: MovieEditionRestoreByCascade :exec
UPDATE MovieEdition SET DeletedAt = NULL
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: MovieEditionRuntimeSet :exec
UPDATE MovieEdition SET Runtime = ? WHERE ID = ?;

-- name: MovieEditionSlugExists :one
SELECT COUNT(*) FROM MovieEdition
WHERE MovieID = ? AND Slug = ? AND DeletedAt IS NULL;

-- name: MovieEditionSlugSet :exec
UPDATE MovieEdition SET Slug = ? WHERE ID = ?;

-- name: MovieEditionSoftDelete :exec
UPDATE MovieEdition
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE ID = sqlc.arg(ID) AND DeletedAt IS NULL;

-- name: MovieEditionSummarySet :exec
UPDATE MovieEdition SET Summary = ? WHERE ID = ?;

-- name: MovieEditionTitleSet :exec
UPDATE MovieEdition SET Title = ? WHERE ID = ?;

-- name: MovieGet :one
SELECT * FROM Movie WHERE ID = ?;

-- name: MovieGetByEditionID :one
SELECT * FROM Movie
WHERE ID IN (SELECT MovieID FROM MovieEdition WHERE MovieEdition.ID = ?);

-- name: MovieGetBySlug :one
SELECT * FROM Movie WHERE Slug = ? AND DeletedAt IS NULL;

-- name: MovieList :many
SELECT * FROM Movie
WHERE DeletedAt IS NULL
ORDER BY Slug;

-- name: MovieListByDownload :many
SELECT * FROM Movie
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT MovieID FROM MovieEdition
	WHERE DeletedAt IS NULL
	AND MovieEdition.ID IN (SELECT MovieEditionID FROM Download)
);

-- name: MovieListByTMDBID :many
SELECT * FROM Movie WHERE DeletedAt IS NULL AND TMDBID IN (sqlc.slice(ids));

-- name: MoviePurgeByCascade :exec
DELETE FROM Movie
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: MovieRestore :exec
UPDATE Movie SET DeletedAt = NULL
WHERE ID = ?;

-- name: MovieRestoreByCascade :exec
UPDATE Movie SET DeletedAt = NULL
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: MovieSlugExists :one
SELECT COUNT(*) FROM Movie WHERE Slug = ? AND DeletedAt IS NULL;

-- name: MovieSlugSet :exec
UPDATE Movie SET Slug = ? WHERE ID = ?;

-- name: MovieSoftDelete :exec
UPDATE Movie
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE ID = sqlc.arg(ID) AND DeletedAt IS NULL;

-- name: MovieVideoCreate :one
INSERT INTO MovieVideo (MovieEditionID, VideoID)
VALUES (?, ?)
RETURNING *;

-- name: MovieVideoDeleteByVideoID :exec
DELETE FROM MovieVideo WHERE VideoID = ?;

-- name: MovieVideoDistinctEditionsByVideo :many
SELECT DISTINCT MovieEditionID FROM MovieVideo WHERE VideoID = ?;

-- name: MovieVideoListByInfoHash :many
SELECT * FROM MovieVideo
WHERE VideoID IN (SELECT ID FROM Video WHERE InfoHash = ?);

-- name: MovieVideoListByMovieEditionID :many
SELECT * FROM MovieVideo
WHERE DeletedAt IS NULL AND MovieEditionID = ?;

-- name: MovieVideoListByMovieID :many
SELECT * FROM MovieVideo
WHERE DeletedAt IS NULL
AND MovieEditionID IN (SELECT ID FROM MovieEdition WHERE DeletedAt IS NULL AND MovieID = ?);

-- name: MovieVideoPurgeByCascade :exec
DELETE FROM MovieVideo
WHERE MovieEditionID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
   OR VideoID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: MovieVideoReassign :exec
INSERT OR IGNORE INTO MovieVideo (MovieEditionID, VideoID)
SELECT src.MovieEditionID, sqlc.arg(ToVideoID) FROM MovieVideo AS src
WHERE src.VideoID = sqlc.arg(FromVideoID) AND src.DeletedAt IS NULL;

-- name: MovieVideoRestoreByCascade :exec
UPDATE MovieVideo SET DeletedAt = NULL
WHERE DeletedAt IS NOT NULL
AND (MovieEditionID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
  OR VideoID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID)))
AND MovieEditionID IN (SELECT ID FROM MovieEdition WHERE DeletedAt IS NULL)
AND VideoID IN (SELECT ID FROM Video WHERE DeletedAt IS NULL);

-- MovieVideoRestoreForReassign clears DeletedAt on any ToVideoID
-- junctions that collide with live FromVideoID junctions. See
-- EpisodeVideoRestoreForReassign for context.
-- name: MovieVideoRestoreForReassign :exec
UPDATE MovieVideo SET DeletedAt = NULL
WHERE MovieVideo.VideoID = sqlc.arg(ToVideoID) AND MovieVideo.DeletedAt IS NOT NULL
AND MovieVideo.MovieEditionID IN (
	SELECT src.MovieEditionID FROM MovieVideo AS src
	WHERE src.VideoID = sqlc.arg(FromVideoID) AND src.DeletedAt IS NULL
);

-- name: MovieVideoSoftDelete :exec
UPDATE MovieVideo
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE MovieEditionID = sqlc.arg(MovieEditionID) AND VideoID = sqlc.arg(VideoID) AND DeletedAt IS NULL;

-- name: MovieVideoSoftDeleteByMovieEditionID :exec
UPDATE MovieVideo
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE MovieEditionID = sqlc.arg(MovieEditionID) AND DeletedAt IS NULL;

-- name: MovieVideoSoftDeleteByVideoID :exec
UPDATE MovieVideo
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE VideoID = sqlc.arg(VideoID) AND DeletedAt IS NULL;

-- name: RenditionCreate :one
INSERT INTO Rendition (
	VideoID, Purpose, Remux, Codec, TargetBitrate,
	MaxHeight, MaxFPS, CopyAudio, SurroundAudio, Priority
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING *;

-- name: RenditionDeleteByVideoID :exec
DELETE FROM Rendition WHERE VideoID = ?;

-- name: RenditionDeleteByVideoIDList :exec
DELETE FROM Rendition WHERE VideoID IN (sqlc.slice(ids));

-- name: RenditionGet :one
SELECT * FROM Rendition WHERE ID = ?;

-- name: RenditionGetDownloadByVideoID :one
SELECT * FROM Rendition
WHERE VideoID = ? AND Purpose = 'download'
LIMIT 1;

-- name: RenditionListByVideoID :many
SELECT * FROM Rendition WHERE VideoID = ?;

-- name: RenditionListEncodedStreamingByVideoID :many
SELECT * FROM Rendition
WHERE VideoID = ? AND Purpose = 'streaming' AND Key != '';

-- name: RenditionListKeysByVideoIDs :many
SELECT Key FROM Rendition
WHERE VideoID IN (sqlc.slice(ids)) AND Key != '';

-- name: RenditionListStreamingByEpisodeID :many
SELECT * FROM Rendition
WHERE Purpose = 'streaming'
AND VideoID IN (SELECT VideoID FROM EpisodeVideo WHERE EpisodeID = ?);

-- name: RenditionListStreamingByMovieEditionID :many
SELECT * FROM Rendition
WHERE Purpose = 'streaming'
AND VideoID IN (SELECT VideoID FROM MovieVideo WHERE MovieEditionID = ?);

-- name: RenditionListStreamingByMovieID :many
SELECT * FROM Rendition
WHERE Purpose = 'streaming'
AND VideoID IN (
	SELECT VideoID FROM MovieVideo
	WHERE MovieEditionID IN (SELECT ID FROM MovieEdition WHERE MovieID = ?)
);

-- name: RenditionListStreamingByVideoID :many
SELECT * FROM Rendition
WHERE VideoID = ? AND Purpose = 'streaming';

-- name: RenditionNextUnencodedStreaming :one
SELECT * FROM Rendition
WHERE VideoID = ? AND Purpose = 'streaming' AND Key = ''
ORDER BY Priority ASC LIMIT 1;

-- name: RenditionUpdateEncode :one
UPDATE Rendition
SET Key = ?, Playlist = ?
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

-- name: SeasonEpisodeDistinctSeasonsByEpisode :many
SELECT DISTINCT SeasonID FROM SeasonEpisode WHERE EpisodeID = ?;

-- name: SeasonEpisodeGet :one
SELECT * FROM SeasonEpisode
WHERE SeasonID = ? AND EpisodeID = ? AND DeletedAt IS NULL;

-- name: SeasonEpisodeGetBySlug :one
SELECT * FROM SeasonEpisode
WHERE EditionID = ? AND Slug = ? AND DeletedAt IS NULL;

-- name: SeasonEpisodeListByEditionID :many
SELECT * FROM SeasonEpisode
WHERE EditionID = ? AND DeletedAt IS NULL
ORDER BY SortKey;

-- name: SeasonEpisodeListByEpisodeID :many
SELECT * FROM SeasonEpisode WHERE EpisodeID = ? AND DeletedAt IS NULL;

-- name: SeasonEpisodeListBySeasonID :many
SELECT * FROM SeasonEpisode
WHERE SeasonID = ? AND DeletedAt IS NULL
ORDER BY SortKey;

-- name: SeasonEpisodeListBySeriesID :many
SELECT * FROM SeasonEpisode
WHERE DeletedAt IS NULL
AND EditionID IN (SELECT ID FROM SeriesEdition WHERE DeletedAt IS NULL AND SeriesID = ?)
ORDER BY SortKey;

-- SeasonEpisodeListRestorableByEpisode returns soft-deleted junctions
-- for the given episode where the season side is live, i.e. junctions
-- that will be restored. Used to read SortKey positions for bumping
-- before the generic junction restore runs.
-- name: SeasonEpisodeListRestorableByEpisode :many
SELECT * FROM SeasonEpisode
WHERE EpisodeID = ? AND DeletedAt IS NOT NULL
AND SeasonID IN (SELECT ID FROM Season WHERE DeletedAt IS NULL);

-- name: SeasonEpisodeNumberingSet :exec
UPDATE SeasonEpisode SET Number = ?, Label = ?, Slug = ? WHERE SeasonID = ? AND EpisodeID = ?;

-- name: SeasonEpisodePurgeByCascade :exec
DELETE FROM SeasonEpisode
WHERE SeasonID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
   OR EpisodeID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: SeasonEpisodeRestore :exec
UPDATE SeasonEpisode SET DeletedAt = NULL
WHERE SeasonID = ? AND EpisodeID = ?;

-- name: SeasonEpisodeRestoreByCascade :exec
UPDATE SeasonEpisode SET DeletedAt = NULL
WHERE DeletedAt IS NOT NULL
AND (SeasonID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID))
  OR EpisodeID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID)))
AND SeasonID IN (SELECT ID FROM Season WHERE DeletedAt IS NULL)
AND EpisodeID IN (SELECT ID FROM Episode WHERE DeletedAt IS NULL);

-- name: SeasonEpisodeSlugExists :one
SELECT COUNT(*) FROM SeasonEpisode
WHERE EditionID = ? AND Slug = ? AND DeletedAt IS NULL;

-- name: SeasonEpisodeSlugSet :exec
UPDATE SeasonEpisode SET Slug = ? WHERE SeasonID = ? AND EpisodeID = ?;

-- name: SeasonEpisodeSoftDelete :exec
UPDATE SeasonEpisode
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE SeasonID = sqlc.arg(SeasonID) AND EpisodeID = sqlc.arg(EpisodeID) AND DeletedAt IS NULL;

-- name: SeasonEpisodeSoftDeleteByEpisodeID :exec
UPDATE SeasonEpisode
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE EpisodeID = sqlc.arg(EpisodeID) AND DeletedAt IS NULL;

-- name: SeasonEpisodeSoftDeleteByEpisodeIDList :exec
UPDATE SeasonEpisode
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE EpisodeID IN (sqlc.slice(ids)) AND DeletedAt IS NULL;

-- name: SeasonEpisodeSoftDeleteBySeasonID :exec
UPDATE SeasonEpisode
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE SeasonID = sqlc.arg(SeasonID) AND DeletedAt IS NULL;

-- SeasonEpisodeSortKeyBump shifts live rows up by one to make room
-- at the given SortKey. Done in two phases so the partial unique
-- index on (SeasonID, SortKey) doesn't see a transient collision:
-- first negate+offset the affected rows (guaranteed-unique interim
-- values), then negate back to get the final +1.
-- name: SeasonEpisodeSortKeyBump :exec
UPDATE SeasonEpisode SET SortKey = -(SortKey + 1)
WHERE SeasonID = sqlc.arg(SeasonID) AND SortKey >= sqlc.arg(SortKey) AND DeletedAt IS NULL;

-- name: SeasonEpisodeSortKeyBumpFinish :exec
UPDATE SeasonEpisode SET SortKey = -SortKey
WHERE SeasonID = ? AND SortKey < 0;

-- name: SeasonGet :one
SELECT * FROM Season WHERE ID = ?;

-- name: SeasonListByEditionID :many
SELECT * FROM Season
WHERE EditionID = ? AND DeletedAt IS NULL
ORDER BY SortKey;

-- name: SeasonListBySeriesID :many
SELECT * FROM Season
WHERE DeletedAt IS NULL
AND EditionID IN (SELECT ID FROM SeriesEdition WHERE DeletedAt IS NULL AND SeriesID = ?)
ORDER BY SortKey;

-- name: SeasonPurgeByCascade :exec
DELETE FROM Season
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: SeasonRestore :exec
UPDATE Season SET DeletedAt = NULL
WHERE ID = ?;

-- name: SeasonRestoreByCascade :exec
UPDATE Season SET DeletedAt = NULL
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: SeasonSoftDelete :exec
UPDATE Season
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE ID = sqlc.arg(ID) AND DeletedAt IS NULL;

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

-- SeriesEditionDefaultSuccessor returns the lex-smallest live non-default
-- edition of the given series, for promotion when the current default is
-- trashed. Errors with sql.ErrNoRows if none exists.
-- name: SeriesEditionDefaultSuccessor :one
SELECT * FROM SeriesEdition
WHERE SeriesID = ? AND DeletedAt IS NULL AND Slug != ''
ORDER BY ID
LIMIT 1;

-- name: SeriesEditionGet :one
SELECT * FROM SeriesEdition WHERE ID = ?;

-- name: SeriesEditionGetBySlug :one
SELECT SeriesEdition.* FROM SeriesEdition
JOIN Series ON Series.ID = SeriesEdition.SeriesID
WHERE Series.Slug = sqlc.arg(SeriesSlug) AND SeriesEdition.Slug = sqlc.arg(EditionSlug)
AND SeriesEdition.DeletedAt IS NULL AND Series.DeletedAt IS NULL;

-- name: SeriesEditionGetDefault :one
SELECT * FROM SeriesEdition WHERE SeriesID = ? AND Slug = '';

-- name: SeriesEditionLabelSet :exec
UPDATE SeriesEdition SET Label = ? WHERE ID = ?;

-- name: SeriesEditionListByDownload :many
SELECT * FROM SeriesEdition
WHERE DeletedAt IS NULL
AND ID IN (SELECT SeriesEditionID FROM Download);

-- name: SeriesEditionListBySeriesID :many
SELECT * FROM SeriesEdition WHERE SeriesID = ? AND DeletedAt IS NULL;

-- name: SeriesEditionListDefault :many
SELECT * FROM SeriesEdition WHERE Slug = '' AND DeletedAt IS NULL;

-- name: SeriesEditionPosterIDSet :exec
UPDATE SeriesEdition SET PosterID = ? WHERE ID = ?;

-- name: SeriesEditionPurgeByCascade :exec
DELETE FROM SeriesEdition
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: SeriesEditionRestore :exec
UPDATE SeriesEdition SET DeletedAt = NULL
WHERE ID = ?;

-- name: SeriesEditionRestoreByCascade :exec
UPDATE SeriesEdition SET DeletedAt = NULL
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: SeriesEditionSlugExists :one
SELECT COUNT(*) FROM SeriesEdition
WHERE SeriesID = ? AND Slug = ? AND DeletedAt IS NULL;

-- name: SeriesEditionSlugSet :exec
UPDATE SeriesEdition SET Slug = ? WHERE ID = ?;

-- name: SeriesEditionSoftDelete :exec
UPDATE SeriesEdition
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE ID = sqlc.arg(ID) AND DeletedAt IS NULL;

-- name: SeriesEditionSummarySet :exec
UPDATE SeriesEdition SET Summary = ? WHERE ID = ?;

-- name: SeriesGet :one
SELECT * FROM Series WHERE ID = ?;

-- name: SeriesGetByEditionID :one
SELECT * FROM Series
WHERE ID IN (SELECT SeriesID FROM SeriesEdition WHERE SeriesEdition.ID = ?);

-- name: SeriesGetBySlug :one
SELECT * FROM Series WHERE Slug = ? AND DeletedAt IS NULL;

-- name: SeriesGetByTVmazeID :one
SELECT * FROM Series WHERE TVmazeID = ? AND DeletedAt IS NULL;

-- name: SeriesList :many
SELECT * FROM Series
WHERE DeletedAt IS NULL;

-- name: SeriesListByDownload :many
SELECT * FROM Series
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT SeriesID FROM SeriesEdition
	WHERE DeletedAt IS NULL
	AND SeriesEdition.ID IN (SELECT SeriesEditionID FROM Download)
);

-- name: SeriesListByTVmazeID :many
SELECT * FROM Series WHERE DeletedAt IS NULL AND TVmazeID IN (sqlc.slice(ids));

-- name: SeriesPurgeByCascade :exec
DELETE FROM Series
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: SeriesRestore :exec
UPDATE Series SET DeletedAt = NULL
WHERE ID = ?;

-- name: SeriesRestoreByCascade :exec
UPDATE Series SET DeletedAt = NULL
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: SeriesSlugExists :one
SELECT COUNT(*) FROM Series WHERE Slug = ? AND DeletedAt IS NULL;

-- name: SeriesSlugSet :exec
UPDATE Series SET Slug = ? WHERE ID = ?;

-- name: SeriesSoftDelete :exec
UPDATE Series
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE ID = sqlc.arg(ID) AND DeletedAt IS NULL;

-- name: SeriesTitleSet :exec
UPDATE Series SET Title = ? WHERE ID = ?;

-- name: SettingListByGroup :many
SELECT * FROM Setting WHERE "Group" = ?;

-- name: SettingSet :exec
INSERT INTO Setting (Key, "Group", Value) VALUES (?, ?, ?)
ON CONFLICT (Key) DO UPDATE SET Value = ?3;

-- name: SlugDelete :exec
DELETE FROM Slug WHERE Target = ?;

-- name: SlugExists :one
SELECT COUNT(*) FROM Slug WHERE Slug = ?;

-- name: SlugGet :one
SELECT * FROM Slug WHERE Slug = ?;

-- name: SlugUpsert :exec
INSERT INTO Slug (Slug, Kind, Target) VALUES (?, ?, ?)
ON CONFLICT (Target) DO UPDATE SET Slug = ?1;

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

-- name: TrashDelete :exec
DELETE FROM Trash WHERE ID = ?;

-- name: TrashGet :one
SELECT * FROM Trash WHERE ID = ?;

-- name: TrashInsert :exec
INSERT INTO Trash (ID, Title, Subtitle, DeletedAt, CascadeOf) VALUES (?, ?, ?, ?, ?);

-- name: TrashList :many
SELECT * FROM Trash WHERE CascadeOf IS NULL ORDER BY DeletedAt DESC;

-- name: TrashListByRoot :many
SELECT * FROM Trash WHERE CascadeOf = ? ORDER BY DeletedAt;

-- name: TrashRootsBefore :many
SELECT ID FROM Trash WHERE CascadeOf IS NULL AND DeletedAt < ?;

-- name: VideoCountByInfoHash :one
SELECT COUNT(*) FROM Video WHERE InfoHash = ?;

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

-- VideoHardDelete removes a Video row outright. Used during
-- duplicate-content merge after its junctions have been re-pointed.
-- name: VideoHardDelete :exec
DELETE FROM Video WHERE ID = ?;

-- name: VideoListByContentHash :many
SELECT * FROM Video WHERE ContentHash = ? AND DeletedAt IS NULL;

-- name: VideoListByEditionID :many
SELECT * FROM Video
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT VideoID FROM EpisodeVideo
	WHERE DeletedAt IS NULL
	AND EpisodeID IN (
		SELECT EpisodeID FROM SeasonEpisode
		WHERE DeletedAt IS NULL AND EditionID = ?
	)
);

-- name: VideoListByEpisodeID :many
SELECT * FROM Video
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT VideoID FROM EpisodeVideo
	WHERE DeletedAt IS NULL AND EpisodeID = ?
);

-- name: VideoListByInfoHash :many
SELECT * FROM Video WHERE InfoHash = ?;

-- name: VideoListByMovieEditionID :many
SELECT * FROM Video
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT VideoID FROM MovieVideo
	WHERE DeletedAt IS NULL AND MovieEditionID = ?
);

-- name: VideoListByMovieID :many
SELECT * FROM Video
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT VideoID FROM MovieVideo
	WHERE DeletedAt IS NULL
	AND MovieEditionID IN (SELECT ID FROM MovieEdition WHERE DeletedAt IS NULL AND MovieID = ?)
);

-- name: VideoListBySeriesID :many
SELECT * FROM Video
WHERE DeletedAt IS NULL
AND ID IN (
	SELECT VideoID FROM EpisodeVideo
	WHERE DeletedAt IS NULL
	AND EpisodeID IN (
		SELECT EpisodeID FROM SeasonEpisode
		WHERE DeletedAt IS NULL
		AND EditionID IN (SELECT ID FROM SeriesEdition WHERE DeletedAt IS NULL AND SeriesID = ?)
	)
);

-- VideoListOrphans returns live Video IDs with no live EpisodeVideo
-- or MovieVideo junctions. Called after soft-deleting a cascade's
-- junctions to reap videos the cascade just stranded.
-- name: VideoListOrphans :many
SELECT v.ID FROM Video v
WHERE v.DeletedAt IS NULL
AND NOT EXISTS (
	SELECT 1 FROM EpisodeVideo ev
	WHERE ev.VideoID = v.ID AND ev.DeletedAt IS NULL
)
AND NOT EXISTS (
	SELECT 1 FROM MovieVideo mv
	WHERE mv.VideoID = v.ID AND mv.DeletedAt IS NULL
)
AND NOT EXISTS (
	SELECT 1 FROM Download d
	WHERE d.InfoHash = v.InfoHash AND d.DeletedAt IS NULL
);

-- name: VideoListPurgeByCascade :many
SELECT ID, OriginalKey FROM Video
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: VideoPurgeByCascade :exec
DELETE FROM Video
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = sqlc.arg(RootID) OR Trash.ID = sqlc.arg(RootID));

-- name: VideoRestore :exec
UPDATE Video SET DeletedAt = NULL
WHERE ID = ?;

-- name: VideoRestoreByCascade :exec
UPDATE Video SET DeletedAt = NULL
WHERE ID IN (SELECT Trash.ID FROM Trash WHERE Trash.CascadeOf = ?);

-- name: VideoSoftDelete :exec
UPDATE Video
SET DeletedAt = sqlc.arg(DeletedAt)
WHERE ID = sqlc.arg(ID) AND DeletedAt IS NULL;

-- name: VideoUpdateMVPlaylist :one
UPDATE Video SET MVPlaylist = ? WHERE ID = ?
RETURNING *;

-- name: VideoUpdateOriginalKey :one
UPDATE Video SET OriginalKey = ?, ContentHash = ? WHERE ID = ?
RETURNING *;

-- name: VideoUpdateProbe :exec
UPDATE Video SET Duration = ?, OriginalType = ?, Format = ? WHERE ID = ?;

-- name: VideoUpdateState :exec
UPDATE Video SET State = ? WHERE ID = ?;
