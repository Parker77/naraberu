-- validate-csv.sql - sanity checks for a Naraberu database
-- Usage (Git Bash):
--   sqlite3 data/naraberu.db < scripts/validate-csv.sql

SELECT 'empty_name' AS issue, COUNT(*) FROM series WHERE TRIM(english_name) = '';
SELECT 'bad_status' AS issue, COUNT(*) FROM series
  WHERE status NOT IN ('', 'to-watch', 'watched', 'in-progress', 'abandoned');
SELECT 'bad_start_date' AS issue, COUNT(*) FROM series
  WHERE start_date != '' AND start_date NOT GLOB '[0-9][0-9][0-9][0-9]'
    AND start_date NOT GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]'
    AND start_date NOT GLOB '[0-9][0-9][0-9][0-9]-[0-9][0-9]-[0-9][0-9]'
    AND start_date NOT GLOB '[0-9][0-9][0-9][0-9]/[0-9][0-9]'
    AND start_date NOT GLOB '[0-9][0-9][0-9][0-9]/[0-9][0-9]/[0-9][0-9]';
SELECT 'score_out_of_range' AS issue, COUNT(*) FROM series WHERE score < 0 OR score > 10;
SELECT 'dup_english_name' AS issue, COUNT(*) FROM (
  SELECT LOWER(english_name) FROM series GROUP BY LOWER(english_name) HAVING COUNT(*) > 1
);
