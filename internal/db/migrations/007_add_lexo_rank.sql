ALTER TABLE tickets ADD COLUMN lexo_rank TEXT;
-- Initial migration to populate lexo_rank from numeric position
UPDATE tickets SET lexo_rank = printf('%010d', CAST(position AS INTEGER));
