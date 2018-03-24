package mongo

import (
	"github.com/globalsign/mgo"
)

type ModuleStore struct {
	s   *mgo.Session
	d   string // database
	c   string // collection
	url string
}

// NewMongoBackend returns an unconnected Mongo Module Backend
// that satisfies the Backend interface.  You must call
// Connect() on the returned store before using it.
func NewMongoBackend(url string) *ModuleStore {
	return &ModuleStore{url: url}
}

func (m *ModuleStore) Connect() error {
	s, err := mgo.Dial(m.url)
	if err != nil {
		return err
	}
	m.s = s

	// TODO: database and collection as env vars, or params to New()? together with user/mongo
	m.d = "athens"
	m.c = "modules"

	index := mgo.Index{
		Key:        []string{"base_url", "module", "version"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	c := m.s.DB(m.d).C(m.c)
	return c.EnsureIndex(index)
}
