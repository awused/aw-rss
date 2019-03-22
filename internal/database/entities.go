package database

import (
	"errors"

	"github.com/awused/aw-rss/internal/structs"
	"github.com/golang/glog"
)

func entityGetSQL(table string, columns string) string {
	return "SELECT " + columns + " FROM " + table + " WHERE id = ?;\n"
}

func insertSQL(table string, columns string, placeholders string) string {
	return "INSERT OR IGNORE INTO " + table + " (" + columns +
		") VALUES (" + placeholders + ");\n"
}

// Generic Get and Mutate methods
// TODO -- these can eventually become real generic  methods

func updateEntity(dot dbOrTx, eu structs.EntityUpdate) error {
	glog.V(2).Infof("Writing updated entity [%s]", eu)
	if eu.Noop() {
		return errors.New("Tried to update using noop entity update")
	}

	sql, binds := eu.Get()
	result, err := dot.Exec(sql, binds...)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return errors.New("Update error: 0 rows affected")
	}
	return nil
}
