package mongo

import (
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/globalsign/mgo"
)

// ConnDetails is a convenient way to store connection details for a mongo
// server
type ConnDetails struct {
	Host     string
	Port     int
	User     string
	Password string
	Timeout  time.Duration
	SSL      bool
}

func (c ConnDetails) String() string {
	return fmt.Sprintf(`Host: %s, Port: %d, User: %s, Password: <redacted>, Timeout: %s`, c.Host, c.Port, c.User, c.Timeout)
}

// GetSession is a utility to create a session from a DB connection string
// and credentials. The caller is responsible for calling Close on the returned
// session.
//
// This function always establishes connections via TLS
func GetSession(deets *ConnDetails, db string) (*mgo.Session, error) {
	dialInfo := &mgo.DialInfo{
		Addrs:    []string{fmt.Sprintf("%s:%d", deets.Host, deets.Port)},
		Timeout:  deets.Timeout,
		Database: db,
		Username: deets.User,
		Password: deets.Password,
	}
	if deets.SSL {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{})
		}
	}

	return mgo.DialWithInfo(dialInfo)
}
