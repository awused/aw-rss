package structs

// TODO -- do more when Generics are more supported

// Entity is an object that represents a single row in the database
type Entity interface {
	ID() int64

	String() string
}

// EntityUpdate holds the SQL needed to update (but not insert) an entity
type EntityUpdate struct {
	e     Entity
	noop  bool
	sql   string
	binds []interface{}
}

func (eu EntityUpdate) String() string {
	return eu.e.String()
}

// Noop returns whether this update is a no-op that should not be written
func (eu EntityUpdate) Noop() bool {
	return eu.noop
}

// Get returns the contents of the EntityUpdate
func (eu EntityUpdate) Get() (string, []interface{}) {
	return eu.sql, eu.binds
}

func noopEntityUpdate(e Entity) EntityUpdate {
	return EntityUpdate{
		e:    e,
		noop: true,
	}
}
