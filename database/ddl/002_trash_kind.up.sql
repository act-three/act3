-- record each trashed entity's kind (its table name) on its Trash row
PRAGMA defer_foreign_keys = ON;
CREATE TABLE TrashNew
(
	ID        TEXT PRIMARY KEY,
	Kind      TEXT NOT NULL CHECK (Kind IN (
		'Collection', 'Download', 'Episode', 'Movie', 'MovieEdition',
		'Season', 'Series', 'SeriesEdition', 'Video'
	)),
	Title     TEXT NOT NULL,
	Subtitle  TEXT NOT NULL,
	DeletedAt INTEGER NOT NULL,
	CascadeOf TEXT REFERENCES TrashNew(ID) ON DELETE CASCADE
)
STRICT;
-- Backfill from the entity tables. A trash row's entity row still
-- exists (soft-deleted), so its kind is wherever its ID turns up.
-- A row matching no table leaves Kind NULL and fails the update.
INSERT INTO TrashNew (ID, Kind, Title, Subtitle, DeletedAt, CascadeOf)
SELECT ID,
	CASE
	WHEN EXISTS (SELECT 1 FROM Collection    WHERE Collection.ID = Trash.ID)    THEN 'Collection'
	WHEN EXISTS (SELECT 1 FROM Download      WHERE Download.InfoHash = Trash.ID) THEN 'Download'
	WHEN EXISTS (SELECT 1 FROM Episode       WHERE Episode.ID = Trash.ID)       THEN 'Episode'
	WHEN EXISTS (SELECT 1 FROM Movie         WHERE Movie.ID = Trash.ID)         THEN 'Movie'
	WHEN EXISTS (SELECT 1 FROM MovieEdition  WHERE MovieEdition.ID = Trash.ID)  THEN 'MovieEdition'
	WHEN EXISTS (SELECT 1 FROM Season        WHERE Season.ID = Trash.ID)        THEN 'Season'
	WHEN EXISTS (SELECT 1 FROM Series        WHERE Series.ID = Trash.ID)        THEN 'Series'
	WHEN EXISTS (SELECT 1 FROM SeriesEdition WHERE SeriesEdition.ID = Trash.ID) THEN 'SeriesEdition'
	WHEN EXISTS (SELECT 1 FROM Video         WHERE Video.ID = Trash.ID)         THEN 'Video'
	END,
	Title, Subtitle, DeletedAt, CascadeOf
FROM Trash;
DROP TABLE Trash;
ALTER TABLE TrashNew RENAME TO Trash;
CREATE INDEX Idx_Trash_DirectDeletedAt ON Trash (DeletedAt DESC) WHERE CascadeOf IS NULL;
CREATE INDEX Idx_Trash_CascadeOf       ON Trash (CascadeOf)     WHERE CascadeOf IS NOT NULL;
