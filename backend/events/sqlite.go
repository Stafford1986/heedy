package events

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"github.com/mattn/go-sqlite3"
	"github.com/sirupsen/logrus"
	//"github.com/heedy/heedy/backend/database"
)

const sqlSchema = `
	CREATE TABLE subscriptions (

	)
`

type Event struct {
	Event      string `json:"event"`
	User       string `json:"user,omitempty"`
	Connection string `json:"connection,omitempty"`
	Source     string `json:"source,omitempty"`
	Key        string `json:"key,omitempty"`
	Type       string `json:"type,omitempty"`
}

func (e *Event) String() string {
	b, err := json.Marshal(e)
	if err != nil {
		panic(err)
	}
	return string(b)
}

type sqliteEvent struct {
	TableName string // The table
	Op        int    // The op on the table
}

var sqliteEventType = map[sqliteEvent]string{
	sqliteEvent{"users", 18}:       "user_create",
	sqliteEvent{"connections", 18}: "connection_create",
	sqliteEvent{"sources", 18}:     "source_create",
	sqliteEvent{"users", 23}:       "user_update",
	sqliteEvent{"connections", 23}: "connection_update",
	sqliteEvent{"sources", 23}:     "source_update",
	sqliteEvent{"users", 9}:        "user_delete",
	sqliteEvent{"connections", 9}:  "connection_delete",
	sqliteEvent{"sources", 9}:      "source_delete",
}

var dbEvents = make(chan *Event, 1000)

// getIDs returns the username, connection id, and source id associated with the given event.
// The associated stmt should automatically return empty strings for inapplicable values
func getEvent(stmt driver.Stmt, rowid int64) (*Event, error) {
	rows, err := stmt.Query([]driver.Value{rowid})
	if err != nil {
		return nil, fmt.Errorf("Sqlite hook error %w", err)
	}
	vals := make([]driver.Value, 5)
	for i := 0; i < 5; i++ {
		var v interface{}
		vals[i] = v

	}
	err = rows.Next(vals)
	if err != nil {
		return nil, fmt.Errorf("Error reading row from sqlite hook %w", err)
	}
	if len(vals) != 5 {
		return nil, fmt.Errorf("Sqlite hook: Incorrect number of returned results")
	}

	tsel := func(v interface{}) string {
		switch vv := v.(type) {
		case string:
			return vv
		case []byte:
			return string(vv)
		default:
			return ""
		}
	}

	return &Event{
		User:       tsel(vals[0]),
		Connection: tsel(vals[1]),
		Source:     tsel(vals[2]),
		Key:        tsel(vals[3]),
		Type:       tsel(vals[4]),
	}, nil
}

func connectHook(conn *sqlite3.SQLiteConn) error {
	username, err := conn.Prepare("SELECT username,'','','','' FROM users WHERE rowid=?")
	if err != nil {
		return err
	}
	connection, err := conn.Prepare("SELECT owner,id,'',plugin,'' FROM connections WHERE rowid=?")
	if err != nil {
		return err
	}
	source, err := conn.Prepare("SELECT owner,connection,id,key,type FROM sources WHERE rowid=?")
	if err != nil {
		return err
	}

	getStmt := func(tblname string) driver.Stmt {
		switch tblname {
		case "users":
			return username
		case "connections":
			return connection
		case "sources":
			return source
		default:
			panic("Unrecognized table name in getStmt")

		}
	}

	conn.RegisterUpdateHook(func(op int, dbname string, tblname string, rowid int64) {
		if dbname != "main" || op == 9 {
			// We don't handle deletes here
			return
		}
		ename, ok := sqliteEventType[sqliteEvent{tblname, op}]
		if ok {
			evt, err := getEvent(getStmt(tblname), rowid)
			if err != nil {
				logrus.Error(err)
				return

			}
			evt.Event = ename
			dbEvents <- evt
		}
	})

	conn.RegisterPreUpdateHook(func(pud sqlite3.SQLitePreUpdateData) {
		if pud.Op != 9 || pud.DatabaseName != "main" {
			return
		}

		// We need pre-updates to handle DELETEs, since we need to know the
		// values before they are deleted
		ename, ok := sqliteEventType[sqliteEvent{pud.TableName, pud.Op}]
		if ok {
			evt, err := getEvent(getStmt(pud.TableName), pud.OldRowID)
			if err != nil {
				logrus.Error(err)
				return

			}
			evt.Event = ename
			dbEvents <- evt
		}
	})
	return nil
}

func eventLoop() {
	for {
		select {
		case evt := <-dbEvents:
			logrus.Debug(evt.String())
		}
	}
}

func init() {
	sql.Register("sqlite3_heedy", &sqlite3.SQLiteDriver{
		ConnectHook: connectHook,
	})
	go eventLoop()
}