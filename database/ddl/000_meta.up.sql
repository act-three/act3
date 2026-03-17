-- meta-schema: version string
CREATE TABLE schema (
	single INTEGER PRIMARY KEY CHECK (single = 0),
	version TEXT NOT NULL,
	digest TEXT NOT NULL
) STRICT;
INSERT INTO
	schema (single, version, digest)
VALUES
	(0, '###', '0000000000000000');
