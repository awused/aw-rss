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
	_, err := dot.Exec(sql, binds...)
	return err
}
