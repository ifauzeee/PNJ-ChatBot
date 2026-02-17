-- migrations/sqlite/000002_add_report_evidence.up.sql
ALTER TABLE reports ADD COLUMN evidence TEXT DEFAULT '';
