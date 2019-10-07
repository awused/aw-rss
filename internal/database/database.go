package database

import (
	"database/sql"
	"errors"
	"strconv"
	"sync"

	"github.com/awused/aw-rss/internal/config"
	log "github.com/sirupsen/logrus"

	// Imported for side effects
	_ "github.com/mattn/go-sqlite3"
)

var (
	once      sync.Once
	singleton *Database
)

type dbOrTx interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Database is the database for storing all persistent data for aw-rss
type Database struct {
	conf      config.Config
	db        *sql.DB
	lock      sync.RWMutex // sqlite3 should be generally threadsafe but don't take chances
	closed    bool
	closeLock sync.Mutex
}

// NewDatabase creates a new database instances around the provided sqlite3 DB
func NewDatabase(c config.Config) (dbase *Database, err error) {
	d := Database{conf: c}

	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	log.Infof("Using database %s", d.conf.Database)
	if d.conf.Database == ":memory:" {
		log.Warning("Using in-memory database, state will not persist between runs")
	}

	db, err := sql.Open("sqlite3", d.conf.Database)
	if err != nil {
		log.Panic(err)
	}

	err = db.Ping()
	if err != nil {
		log.Panic(err)
	}

	d.db = db

	d.init()

	return &d, nil
}

// Close closes the database, freeing all resources
// Requests in flight are allowed to complete but those that haven't started
// are cancelled
func (d *Database) Close() error {
	log.Info("Closing database")

	if d.closed {
		log.Warning("Tried to close database that has already been closed")
		return nil
	}
	d.closeLock.Lock()
	defer d.closeLock.Unlock()
	if d.closed {
		log.Warning("Tried to close database that has already been closed")
		return nil
	}

	d.lock.Lock()
	defer d.lock.Unlock()
	d.closed = true

	return d.db.Close()
}

func (d *Database) readVersion() int {
	var version string
	err := d.db.QueryRow("SELECT value FROM metadata WHERE key = ?", "dbversion").Scan(&version)
	if err != nil {
		log.Panic(err)
	}
	i, err := strconv.Atoi(version)
	if err != nil {
		log.Panic(err)
	}
	return i
}

func (d *Database) getVersion() int {
	rows, err := d.db.Query("SELECT name FROM sqlite_master WHERE type = 'table';")
	if err != nil {
		log.Panic(err)
	}

	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		if err != nil {
			log.Panic(err)
		}

		if tableName == "metadata" {
			err = rows.Close()
			if err != nil {
				log.Panic(err)
			}

			return d.readVersion()
		}
	}

	return 0
}

func (d *Database) upgradeFrom(version int) {
	d.upgradeTo(1, version, `
				CREATE TABLE metadata(key TEXT, value TEXT, PRIMARY KEY(key));
				CREATE TABLE feeds(
						id INTEGER PRIMARY KEY,
						url TEXT UNIQUE NOT NULL,
						disabled INT NOT NULL DEFAULT 0,
						title TEXT NOT NULL DEFAULT '',
						siteurl TEXT NOT NULL DEFAULT '',
						lastfetchfailed INT NOT NULL DEFAULT 0);
				CREATE TABLE items(
						id INTEGER PRIMARY KEY,
						feedid INTEGER NOT NULL,
						key TEXT NOT NULL,
						title TEXT NOT NULL,
						url TEXT NOT NULL,
						content TEXT NOT NULL,
						timestamp TIMESTAMP NOT NULL,
						read INT NOT NULL DEFAULT 0,
						UNIQUE(feedid, key),
						FOREIGN KEY(feedid) REFERENCES feeds(id));`)
	d.upgradeTo(2, version, `
				ALTER TABLE feeds ADD COLUMN usertitle TEXT NOT NULL DEFAULT '';`)
	d.upgradeTo(3, version, `
				ALTER TABLE feeds ADD COLUMN
						lastsuccesstime TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:00+00:00';`)
	d.upgradeTo(4, version,
		`CREATE INDEX items_read_feed_index ON items(read, feedid);`)
	d.upgradeTo(5, version,
		`CREATE INDEX feeds_disabled_index ON feeds(disabled);`)
	d.upgradeTo(6, version, `
				ALTER TABLE feeds RENAME TO feeds_old;
				ALTER TABLE items RENAME TO items_old;

				DROP INDEX items_read_feed_index;
				DROP INDEX feeds_disabled_index;


				CREATE TABLE feeds(
						id INTEGER PRIMARY KEY,
						url TEXT UNIQUE NOT NULL,
						disabled INT NOT NULL DEFAULT 0,
						title TEXT NOT NULL DEFAULT '',
						siteurl TEXT NOT NULL DEFAULT '',
						lastfetchfailed INT NOT NULL DEFAULT 0,
						usertitle TEXT NOT NULL DEFAULT '',
						lastsuccesstime TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:00+00:00',
						commit_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
				CREATE TABLE items(
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


				INSERT INTO feeds SELECT *, CURRENT_TIMESTAMP FROM feeds_old;
				INSERT INTO items SELECT *, CURRENT_TIMESTAMP FROM items_old;

				CREATE INDEX items_read_feed_index ON items(read, feedid);
				CREATE INDEX feeds_disabled_index ON feeds(disabled);
				CREATE INDEX items_commit_index ON items(commit_timestamp);
				CREATE INDEX feeds_commit_index ON feeds(commit_timestamp);`)
	d.upgradeTo(7, version, `
				DROP TABLE items_old;
				DROP TABLE feeds_old;`)
	d.upgradeTo(8, version, `
				ALTER TABLE feeds RENAME TO feeds_old;
				ALTER TABLE items RENAME TO items_old;

				DROP INDEX items_read_feed_index;
				DROP INDEX feeds_disabled_index;
				DROP INDEX items_commit_index;
				DROP INDEX feeds_commit_index;

				CREATE TABLE feeds(
						id INTEGER PRIMARY KEY,
						url TEXT UNIQUE NOT NULL,
						disabled INT NOT NULL DEFAULT 0,
						title TEXT NOT NULL DEFAULT '',
						siteurl TEXT NOT NULL DEFAULT '',
						lastfetchfailed INT NOT NULL DEFAULT 0,
						usertitle TEXT NOT NULL DEFAULT '',
						lastsuccesstime TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:00+00:00',
						commit_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						create_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
				CREATE TABLE items(
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


				INSERT INTO feeds SELECT *, CURRENT_TIMESTAMP FROM feeds_old;
				INSERT INTO items SELECT * FROM items_old;

				CREATE INDEX items_read_feed_index ON items(read, feedid);
				CREATE INDEX feeds_disabled_index ON feeds(disabled);
				CREATE INDEX items_commit_index ON items(commit_timestamp);
				CREATE INDEX feeds_commit_index ON feeds(commit_timestamp);`)
	d.upgradeTo(9, version, `
				DROP TABLE items_old;
				DROP TABLE feeds_old;`)
	d.upgradeTo(10, version, `
				ALTER TABLE feeds RENAME TO feeds_old;
				ALTER TABLE items RENAME TO items_old;

				DROP INDEX items_read_feed_index;
				DROP INDEX feeds_disabled_index;
				DROP INDEX items_commit_index;
				DROP INDEX feeds_commit_index;

				CREATE TABLE feeds(
						id INTEGER PRIMARY KEY,
						url TEXT UNIQUE NOT NULL,
						disabled INT NOT NULL DEFAULT 0,
						title TEXT NOT NULL DEFAULT '',
						siteurl TEXT NOT NULL DEFAULT '',
						usertitle TEXT NOT NULL DEFAULT '',
						commit_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						create_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
						failing_since TIMESTAMP);
				CREATE TABLE items(
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


				INSERT INTO feeds
					SELECT
						id,
						url,
						disabled,
						title,
						siteurl,
						usertitle,
						commit_timestamp,
						create_timestamp,
						NULL
					FROM feeds_old;
				INSERT INTO items SELECT * FROM items_old;

				CREATE INDEX items_read_feed_index ON items(read, feedid);
				CREATE INDEX feeds_disabled_index ON feeds(disabled);
				CREATE INDEX items_commit_index ON items(commit_timestamp);
				CREATE INDEX feeds_commit_index ON feeds(commit_timestamp);`)
	d.upgradeTo(11, version, `
				DROP TABLE items_old;
				DROP TABLE feeds_old;`)
	d.upgradeTo(12, version, `
				CREATE INDEX items_feed_timestamp_index ON items(feedid, timestamp);`)
	if version < 13 {
		_, err := d.db.Exec(`
				PRAGMA foreign_keys = OFF;`)
		if err != nil {
			log.Panic(err)
		}
		d.upgradeTo(13, version, `
				CREATE TABLE categories(
						id INTEGER PRIMARY KEY,
						disabled INT NOT NULL DEFAULT 0,
						name TEXT UNIQUE NOT NULL,
						title TEXT NOT NULL,
						hidden_nav INTEGER NOT NULL DEFAULT 0,
						hidden_main INTEGER NOT NULL DEFAULT 0,
						commit_timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);

				ALTER TABLE feeds RENAME TO feeds_old;
				ALTER TABLE items RENAME TO items_old;

				DROP INDEX items_read_feed_index;
				DROP INDEX feeds_disabled_index;
				DROP INDEX items_commit_index;
				DROP INDEX feeds_commit_index;
				DROP INDEX items_feed_timestamp_index;

				CREATE TABLE feeds(
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
				CREATE TABLE items(
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

				INSERT INTO feeds
						SELECT
								*, NULL
						FROM feeds_old;
				INSERT INTO items SELECT * FROM items_old;

				DROP TABLE items_old;
				DROP TABLE feeds_old;

				CREATE INDEX items_read_feed_index ON items(read, feedid);
				CREATE INDEX feeds_disabled_index ON feeds(disabled);
				CREATE INDEX items_commit_index ON items(commit_timestamp);
				CREATE INDEX feeds_commit_index ON feeds(commit_timestamp);
				CREATE INDEX items_feed_timestamp_index ON items(feedid, timestamp);

				PRAGMA foreign_key_check;`)
		_, err = d.db.Exec(`
				PRAGMA foreign_keys = ON;
				PRAGMA foreign_key_check;`)
		if err != nil {
			log.Panic(err)
		}
	}

	d.upgradeTo(14, version, `
				CREATE INDEX categories_disabled_index ON categories(disabled);
				CREATE INDEX categories_commit_index ON categories(commit_timestamp);`)

	d.upgradeTo(15, version, `
				CREATE INDEX items_url_index ON items(url);`)

	d.upgradeTo(16, version, `
				ALTER TABLE categories ADD COLUMN
						sort_position INTEGER DEFAULT NULL;`)

	if version < 13 {
		_, err := d.db.Exec("VACUUM")
		if err != nil {
			log.Panic(err)
		}
	}
}

func (d *Database) upgradeTo(version int, oldVersion int, sql string) {
	if version <= oldVersion {
		return
	}
	log.Infof("Upgrading database to version %d", version)

	tx, err := d.db.Begin()
	if err != nil {
		log.Panic(err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(sql)
	if err != nil {
		log.Error(err)
		panic(err)
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata(key, value) VALUES ('dbversion', ?);`, strconv.Itoa(version))

	if err != nil {
		log.Error(err)
		panic(err)
	}

	err = tx.Commit()
	if err != nil {
		log.Panic(err)
	}
}

func (d *Database) init() {
	log.Debug("init() started")

	version := d.getVersion()

	log.Infof("Database is version %d", version)

	_, err := d.db.Exec(`
			PRAGMA foreign_keys = ON;`)
	if err != nil {
		log.Panic(err)
	}

	d.upgradeFrom(version)

	log.Trace("init() completed")
}

var ErrClosed = errors.New("Database already closed")

func (d *Database) checkClosed() error {
	if d.closed {
		err := ErrClosed
		log.Error(err)
		return err
	}
	return nil
}
