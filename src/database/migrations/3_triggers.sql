-- These mess up UPDATE .. RETURNING * queries, but doing "BEFORE UPDATE" triggers updating the
-- same row are undefined in sqlite. They seem to work, but best not to risk it.
-- Since the frontend will treat new values with the same commit_timestamp as newer, this is fine

CREATE TRIGGER update_items
    AFTER UPDATE
    ON items
    FOR EACH ROW
    WHEN NEW.commit_timestamp < CURRENT_TIMESTAMP
BEGIN
    UPDATE items SET commit_timestamp = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_feeds
    AFTER UPDATE
    ON feeds
    FOR EACH ROW
    WHEN NEW.commit_timestamp < CURRENT_TIMESTAMP
BEGIN
    UPDATE feeds SET commit_timestamp = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER update_categories
    AFTER UPDATE
    ON categories
    FOR EACH ROW
    WHEN NEW.commit_timestamp < CURRENT_TIMESTAMP
BEGIN
    UPDATE categories SET commit_timestamp = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

