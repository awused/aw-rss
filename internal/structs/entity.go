package structs

// TODO -- do more when Generics are more supported

// Entity is an object that represents a single row in the database
type Entity interface {
	ID() int64

	UpdateSQL() EntityUpdate
}

// EntityUpdate holds the SQL needed to update (but not insert) an entity
type EntityUpdate struct {
	sql   string
	binds []interface{}
}

// Get returns the contents of the EntityUpdate
func (eu EntityUpdate) Get() (string, []interface{}) {
	return eu.sql, eu.binds
}
