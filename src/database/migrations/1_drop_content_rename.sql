ALTER TABLE feeds RENAME COLUMN siteurl TO site_url;
ALTER TABLE feeds RENAME COLUMN usertitle TO user_title;
ALTER TABLE feeds RENAME COLUMN categoryid TO category_id;

-- I haven't used this once in a decade, and the content of old items is of low value
-- To re-add this, I'd want to add both content and description and actually fill them in.
ALTER TABLE items DROP COLUMN content;
ALTER TABLE items RENAME COLUMN feedid TO feed_id;

PRAGMA foreign_key_check;

