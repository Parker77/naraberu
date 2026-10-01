-- inspect.sql - print a quick summary of the Naraberu database
-- Usage (Git Bash):
--   sqlite3 data/naraberu.db < scripts/inspect.sql

SELECT COUNT(*) AS series_count FROM series;
SELECT COUNT(*) AS favorites_count FROM series WHERE favorite = 1;
SELECT COUNT(*) AS owned_count FROM series WHERE owned = 1;

SELECT status, COUNT(*) AS n FROM series GROUP BY status ORDER BY n DESC;

SELECT key, value FROM settings ORDER BY key;

SELECT id, english_name, score, status, created_at, updated_at
FROM series
ORDER BY updated_at DESC
LIMIT 10;
