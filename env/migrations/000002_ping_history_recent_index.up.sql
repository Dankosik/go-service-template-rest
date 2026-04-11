CREATE INDEX idx_ping_history_created_at_id
ON ping_history (created_at DESC, id DESC);
