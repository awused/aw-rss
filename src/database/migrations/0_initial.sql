CREATE TABLE IF NOT EXISTS categories(
                id INTEGER PRIMARY KEY,
                disabled INT NOT NULL DEFAULT 0,
                name TEXT UNIQUE NOT NULL,
                title TEXT NOT NULL,
                hidden_nav INTEGER NOT NULL DEFAULT 0,
                hidden_main INTEGER NOT NULL DEFAULT 0,
                commit_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                sort_position INTEGER DEFAULT NULL);


CREATE TABLE IF NOT EXISTS feeds(
                id INTEGER PRIMARY KEY,
                url TEXT UNIQUE NOT NULL,
                disabled INT NOT NULL DEFAULT 0,
                title TEXT NOT NULL DEFAULT '',
                siteurl TEXT NOT NULL DEFAULT '',
                usertitle TEXT NOT NULL DEFAULT '',
                commit_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                create_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                failing_since TIMESTAMP DEFAULT NULL,
                categoryid INTEGER DEFAULT NULL,
                FOREIGN KEY(categoryid) REFERENCES categories(id));

CREATE TABLE IF NOT EXISTS items(
                id INTEGER PRIMARY KEY,
                feedid INTEGER NOT NULL,
                key TEXT NOT NULL,
                title TEXT NOT NULL,
                url TEXT NOT NULL,
                content TEXT NOT NULL,
                timestamp TIMESTAMP NOT NULL,
                read INT NOT NULL DEFAULT 0,
                commit_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
                UNIQUE(feedid, key),
                FOREIGN KEY(feedid) REFERENCES feeds(id));

CREATE INDEX IF NOT EXISTS categories_disabled_index ON categories(disabled);
CREATE INDEX IF NOT EXISTS categories_commit_index ON categories(commit_timestamp);

CREATE INDEX IF NOT EXISTS feeds_disabled_index ON feeds(disabled);
CREATE INDEX IF NOT EXISTS feeds_commit_index ON feeds(commit_timestamp);

CREATE INDEX IF NOT EXISTS items_read_feed_index ON items(read, feedid);
CREATE INDEX IF NOT EXISTS items_url_index ON items(url);
CREATE INDEX IF NOT EXISTS items_feed_timestamp_index ON items(feedid, timestamp);
CREATE INDEX IF NOT EXISTS items_commit_index ON items(commit_timestamp);

PRAGMA foreign_key_check;

