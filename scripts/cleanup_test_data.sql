-- cleanup_test_data.sql
-- Remove smoke test / test data from all tables.
-- Run against MySQL (port 3308): mysql -h 127.0.0.1 -P 3308 -u root -p linkstash < scripts/cleanup_test_data.sql

-- 1. Identify test URL IDs
SET @test_url_ids = (
    SELECT GROUP_CONCAT(id) FROM t_urls
    WHERE link LIKE '%smoke-%'
       OR link LIKE '%example.com/mysql-smoke%'
       OR link LIKE '%test-soft-delete%'
);

-- 2. Delete orphaned embeddings for test URLs
DELETE FROM t_embeddings WHERE url_id IN (
    SELECT id FROM t_urls
    WHERE link LIKE '%smoke-%'
       OR link LIKE '%example.com/mysql-smoke%'
       OR link LIKE '%test-soft-delete%'
);

-- 3. Delete orphaned visit records for test URLs
DELETE FROM t_visit_records WHERE url_id IN (
    SELECT id FROM t_urls
    WHERE link LIKE '%smoke-%'
       OR link LIKE '%example.com/mysql-smoke%'
       OR link LIKE '%test-soft-delete%'
);

-- 4. Delete LLM logs for test URLs
DELETE FROM t_llm_logs WHERE url_id IN (
    SELECT id FROM t_urls
    WHERE link LIKE '%smoke-%'
       OR link LIKE '%example.com/mysql-smoke%'
       OR link LIKE '%test-soft-delete%'
);

-- 5. Delete the test URLs themselves
DELETE FROM t_urls
WHERE link LIKE '%smoke-%'
   OR link LIKE '%example.com/mysql-smoke%'
   OR link LIKE '%test-soft-delete%';

-- 6. Hard-delete all soft-deleted records across all tables
DELETE FROM t_embeddings   WHERE deleted_at IS NOT NULL;
DELETE FROM t_visit_records WHERE deleted_at IS NOT NULL;
DELETE FROM t_llm_logs     WHERE deleted_at IS NOT NULL;
DELETE FROM t_urls         WHERE deleted_at IS NOT NULL;

-- 7. Clean up orphaned embeddings (url_id no longer exists)
DELETE FROM t_embeddings WHERE url_id NOT IN (SELECT id FROM t_urls);

-- 8. Clean up orphaned visit records
DELETE FROM t_visit_records WHERE url_id NOT IN (SELECT id FROM t_urls);

-- 9. Clean up orphaned LLM logs
DELETE FROM t_llm_logs WHERE url_id NOT IN (SELECT id FROM t_urls);
