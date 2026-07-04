-- record object kinds as table names on Trash and Slug rows;
-- add tombstones preserving slugs across renames
PRAGMA defer_foreign_keys = ON;

-- Trash gains a Kind column naming the trashed entity's table.
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

-- Slug.Kind moves from ad-hoc lowercase names to the same table-name
-- vocabulary. An unknown stored kind leaves NULL and fails the update.
--
-- Slug also gains a Tombstone flag. A tombstone row preserves a slug
-- its target no longer answers to: resolving one still finds the
-- target, and the server redirects to the target's current canonical
-- slug. A target has exactly one live slug but any number of
-- tombstones. Tombstones reserve nothing; whoever wants a tombstoned
-- slug simply claims it.
CREATE TABLE SlugNew
(
	Slug      TEXT PRIMARY KEY,
	Kind      TEXT NOT NULL CHECK (Kind IN ('Collection', 'Movie', 'Series')),
	Target    TEXT NOT NULL,
	Tombstone INTEGER NOT NULL DEFAULT 0 CHECK (Tombstone IN (0, 1))
)
STRICT;
INSERT INTO SlugNew (Slug, Kind, Target)
SELECT Slug,
	CASE Kind
	WHEN 'collection' THEN 'Collection'
	WHEN 'movie'      THEN 'Movie'
	WHEN 'series'     THEN 'Series'
	END,
	Target
FROM Slug;
DROP TABLE Slug;
ALTER TABLE SlugNew RENAME TO Slug;
CREATE UNIQUE INDEX UQ_Slug_Target ON Slug (Target) WHERE Tombstone = 0;

-- Edition slugs are scoped to their parent movie or series rather than
-- the global Slug table, so their tombstones get a per-parent table of
-- their own. Same semantics as Slug tombstones; live edition slugs
-- stay on the edition rows.
CREATE TABLE EditionSlugTombstone
(
	ParentID TEXT NOT NULL, -- MovieID or SeriesID
	Slug     TEXT NOT NULL,
	TargetID TEXT NOT NULL, -- the MovieEdition or SeriesEdition that held Slug
	PRIMARY KEY (ParentID, Slug)
)
STRICT;
