package database

import (
	"database/sql"
	"flag"
	"fmt"
	"strconv"
	"sync"

	"github.com/golang/glog"
	// Imported for side effects
	_ "github.com/mattn/go-sqlite3"
)

var dbfile = flag.String("db", ":memory:", "The file used to persist the database. Defaults to an in-memory database")

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
	db        *sql.DB
	lock      sync.RWMutex // sqlite3 should be generally threadsafe but don't take chances
	closed    bool
	closeLock sync.Mutex
}

// GetDatabase gets an existing DB instance, if any, or creates a new one
func GetDatabase() (d *Database, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	glog.V(5).Info("GetDatabase() started")

	once.Do(func() {
		glog.V(1).Info("No existing instance, creating new database")

		glog.V(1).Infof("Using database %s", *dbfile)
		if *dbfile == ":memory:" {
			glog.Warning("Using in-memory database, state will not persist between runs")
		}

		db, err := sql.Open("sqlite3", *dbfile)
		checkErr(err)

		err = db.Ping()
		checkErr(err)

		var dbase Database
		dbase.db = db

		dbase.init()

		singleton = &dbase
	})

	glog.V(5).Info("GetDatabase() completed")
	return singleton, nil
}

// Close closes the database, freeing all resources
// Requests in flight are allowed to complete but those that haven't started
// are cancelled
func (d *Database) Close() error {
	glog.Info("Closing database")

	if d.closed {
		glog.Warning("Tried to close database that has already been closed")
		return nil
	}
	d.closeLock.Lock()
	defer d.closeLock.Unlock()
	if d.closed {
		glog.Warning("Tried to close database that has already been closed")
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
	checkErr(err)
	i, err := strconv.Atoi(version)
	checkErr(err)
	return i
}

func (d *Database) getVersion() int {
	rows, err := d.db.Query("SELECT name FROM sqlite_master WHERE type = 'table';")
	checkErr(err)

	for rows.Next() {
		var tableName string
		err := rows.Scan(&tableName)
		checkErr(err)

		if tableName == "metadata" {
			err = rows.Close()
			checkErr(err)

			return d.readVersion()
		}
	}

	return 0
}

func (d *Database) upgradeFrom(version int) {
	if version < 1 {
		d.upgradeTo(1, `
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
	} // version < 1
	if version < 2 {
		d.upgradeTo(2, `
				ALTER TABLE feeds ADD COLUMN usertitle TEXT NOT NULL DEFAULT '';`)
	} // version < 2
	if version < 3 {
		d.upgradeTo(3, `
				ALTER TABLE feeds ADD COLUMN
						lastsuccesstime TIMESTAMP NOT NULL DEFAULT '1970-01-01 00:00:00+00:00';`)
	} // version < 3
	if version < 4 {
		d.upgradeTo(4, `CREATE INDEX items_read_feed_index ON items(read, feedid);`)
	} // version < 4
	if version < 5 {
		d.upgradeTo(5, `CREATE INDEX feeds_disabled_index ON feeds(disabled);`)
	} // version < 5
	if version < 6 {
		d.upgradeTo(6, `
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
	} // version < 6
	if version < 7 {
		d.upgradeTo(7, `
				DROP TABLE items_old;
				DROP TABLE feeds_old;`)
	} // version < 7
	if version < 8 {
		d.upgradeTo(8, `
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
	} // version < 8
	if version < 9 {
		d.upgradeTo(9, `
				DROP TABLE items_old;
				DROP TABLE feeds_old;`)
		_, err := d.db.Exec("VACUUM")
		checkErr(err)
	} // version < 9
	// TODO -- drop all these progressive updates before open sourcing
	if version < 10 {
		d.upgradeTo(10, `
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
						id,url,
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
	} // version < 10
	if version < 11 {
		d.upgradeTo(11, `
				DROP TABLE items_old;
				DROP TABLE feeds_old;`)
		_, err := d.db.Exec("VACUUM")
		checkErr(err)
	} // version < 11

}

func (d *Database) upgradeTo(version int, sql string) {
	glog.Infof("Upgrading database to version %d", version)

	tx, err := d.db.Begin()
	checkErr(err)

	_, err = tx.Exec(sql)
	if err != nil {
		glog.Error(err)
		tx.Rollback()
		panic(err)
	}

	_, err = tx.Exec(`INSERT OR REPLACE INTO metadata(key, value) VALUES ('dbversion', ?);`, strconv.Itoa(version))

	if err != nil {
		glog.Error(err)
		tx.Rollback()
		panic(err)
	}

	checkErr(tx.Commit())
}

func (d *Database) init() {
	glog.V(5).Info("init() started")

	version := d.getVersion()

	glog.Infof("Database is version %d", version)

	_, err := d.db.Exec(`
			PRAGMA foreign_keys = ON;`)
	checkErr(err)

	d.upgradeFrom(version)

	glog.V(5).Info("init() completed")
}

func (d *Database) checkClosed() error {
	if d.closed {
		err := fmt.Errorf("Database already closed")
		glog.ErrorDepth(1, err)
		return err
	}
	return nil
}

func checkErr(err error) {
	if err != nil {
		glog.ErrorDepth(1, err)
		panic(err)
	}
}
