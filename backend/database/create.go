package database

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/heedy/heedy/backend/assets"

	// Make sure we include sqlite support
	_ "github.com/mattn/go-sqlite3"
)

var schema = `

-- A user is a group with an additional password. The id is a group id, we will
-- add the foreign key constraint once the groups table is created.
CREATE TABLE users (
	name VARCHAR(36) PRIMARY KEY NOT NULL,
	password VARCHAR NOT NULL
);


-- Groups are the underlying container for access control and sharing
CREATE TABLE groups (
	id VARCHAR(36) UNIQUE NOT NULL PRIMARY KEY,
	name VARCHAR NOT NULL,
	fullname VARCHAR DEFAULT '',
	description VARCHAR DEFAULT '',
	icon VARCHAR DEFAULT '',
	owner VARCHAR(36) NOT NULL,

	CONSTRAINT groupowner
		FOREIGN KEY(owner) 
		REFERENCES users(name)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);


-- We explictly exploit deferred constraints to allow the user's name to be 
-- constrained to a group id. When adding a user, we first add the user,
-- and then the group, which defers the user foreign key check to commit.
--
-- EDIT: Holy shit sqlite doesn't support ALTER TABLE ADD CONSTRAINT! That means we just have
-- to be very careful to explicitly manipulate the database in the correct way so that
-- the user and group are modified/deleted correctly
--
-- ALTER TABLE users ADD CONSTRAINT usergroup
--	FOREIGN KEY (name) 
--	REFERENCES groups(id)
--	ON UPDATE CASCADE
--	ON DELETE CASCADE
--	DEFERRABLE INITIALLY DEFERRED;
	

CREATE TABLE connections (
	id VARCHAR(36) UNIQUE NOT NULL PRIMARY KEY,
	name VARCHAR NOT NULL,
	fullname VARCHAR DEFAULT '',
	description VARCHAR DEFAULT '',
	icon VARCHAR DEFAULT '',
	owner VARACHAR(36) NOT NULL,

	-- Can (but does not have to) have an API key
	apikey VARCHAR UNIQUE DEFAULT NULL,

	settings VARCHAR DEFAULT '{}',
	setting_schema VARCHAR DEFAULT '{}',

	CONSTRAINT connectionowner
		FOREIGN KEY(owner) 
		REFERENCES users(name)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);
-- We will want to list connections by owner 
CREATE INDEX connectionowner ON connections(owner,name);
-- A lot of querying will happen by API key
CREATE INDEX connectionapikey ON connections(apikey);


CREATE TABLE streams (
	id VARCHAR(36) UNIQUE NOT NULL PRIMARY KEY,
	name VARCHAR NOT NULL,
	fullname VARCHAR DEFAULT '',
	description VARCHAR DEFAULT '',
	icon VARCHAR DEFAULT '',
	connection VARACHAR(36) DEFAULT NULL,
	owner VARCHAR(36) NOT NULL,

	-- json schema
	schema VARCHAR DEFAULT '{}',

	-- Set to '' when the stream is internal, and gives the rest url/plugin uri for querying if external
	external VARCHAR DEFAULT '',

	actor BOOLEAN DEFAULT FALSE, -- Whether the stream is also an actor, ie, it can take action, meaning that it performs interventions

	-- What access is given to the user and others who have access to the stream
	access INTEGER DEFAULT 2, -- 0 hidden, 1 read, 2 insert actions, 3 insert, 4 remove data, 5 modify, 6 delete

	CONSTRAINT streamconnection
		FOREIGN KEY(connection) 
		REFERENCES connections(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE,

	CONSTRAINT streamowner
		FOREIGN KEY(owner) 
		REFERENCES users(name)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

------------------------------------------------------------------------------------
-- GROUP ACCESS
------------------------------------------------------------------------------------

-- The scopes available to the group
CREATE TABLE group_scopes (
	groupid VARCHAR(36) NOT NULL,
	scope VARCHAR NOT NULL,
	PRIMARY KEY (groupid,scope),
	UNIQUE (groupid,scope),
	CONSTRAINT fk_groupid
		FOREIGN KEY(groupid)
		REFERENCES groups(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

CREATE TABLE group_members (
	groupid VARCHAR(36),
	id VARCHAR(36),

	access INTEGER DEFAULT 2, -- 1 is readonly, 2 gives stream insert access, 3 allows adding streams/sources, 4 allows removing streams/sources

	UNIQUE(groupid,id),
	PRIMARY KEY (groupid,id),

	CONSTRAINT idid
		FOREIGN KEY(id)
		REFERENCES users(name)
		ON UPDATE CASCADE
		ON DELETE CASCADE,
	
	CONSTRAINT groupid
		FOREIGN KEY(groupid)
		REFERENCES groups(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

CREATE TABLE group_streams (
	groupid VARCHAR(36),
	id VARCHAR(36),

	access INTEGER DEFAULT 1, -- Same as stream access

	UNIQUE(id,groupid),
	PRIMARY KEY (id,groupid),

	CONSTRAINT idid
		FOREIGN KEY(id)
		REFERENCES streams(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE,
	
	CONSTRAINT groupid
		FOREIGN KEY(groupid)
		REFERENCES groups(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

CREATE TABLE group_connections (
	groupid VARCHAR(36),
	id VARCHAR(36),

	access INTEGER DEFAULT 1, -- Same as stream access

	UNIQUE(id,groupid),
	PRIMARY KEY (id,groupid),

	CONSTRAINT idid
		FOREIGN KEY(id)
		REFERENCES connections(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE,
	
	CONSTRAINT groupid
		FOREIGN KEY(groupid)
		REFERENCES groups(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

------------------------------------------------------------------------------------
-- CONNECTION ACCESS
------------------------------------------------------------------------------------

-- The scopes available to the connection
CREATE TABLE connection_scopes (
	connectionid VARCHAR(36) NOT NULL,
	scope VARCHAR NOT NULL,
	PRIMARY KEY (connectionid,scope),
	UNIQUE (connectionid,scope),
	CONSTRAINT fk_connectionid
		FOREIGN KEY(connectionid)
		REFERENCES connections(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);


CREATE TABLE connection_streams (
	connection VARCHAR(36),
	id VARCHAR(36),

	access INTEGER DEFAULT 1, -- Same as stream access

	UNIQUE(connection,id),
	PRIMARY KEY (connection,id),

	CONSTRAINT idid
		FOREIGN KEY(id)
		REFERENCES streams(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE,
	
	CONSTRAINT connectionid
		FOREIGN KEY(connection)
		REFERENCES connections(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

CREATE TABLE connection_connections (
	connection VARCHAR(36),
	id VARCHAR(36),

	access INTEGER DEFAULT 1, -- Same as stream access

	UNIQUE(connection,id),
	PRIMARY KEY (connection,id),

	CONSTRAINT idid
		FOREIGN KEY(id)
		REFERENCES connections(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE,
	
	CONSTRAINT connectionid
		FOREIGN KEY(connection)
		REFERENCES connections(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

CREATE TABLE connection_groups (
	connection VARCHAR(36),
	id VARCHAR(36),

	access INTEGER DEFAULT 1, -- Same as stream access

	UNIQUE(connection,id),
	PRIMARY KEY (connection,id),

	CONSTRAINT idid
		FOREIGN KEY(id)
		REFERENCES groups(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE,
	
	CONSTRAINT connectionid
		FOREIGN KEY(connection)
		REFERENCES connections(id)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

------------------------------------------------------------------
-- User Login Tokens
------------------------------------------------------------------
-- These are used to control manually logged in devices,
-- so that we don't need to put passwords in cookies

CREATE TABLE logins (
	user VARCHAR(36) NOT NULL,
	token VARCHAR UNIQUE NOT NULL,

	CONSTRAINT fk_user
		FOREIGN KEY(user) 
		REFERENCES users(name)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

-- This will be requested on every single query
CREATE INDEX login_tokens ON logins(token);

------------------------------------------------------------------
-- Key-Value Storage for Plugins & Frontend
------------------------------------------------------------------

-- The given storage allows the frontend to save settings and such
CREATE TABLE user_kv (
	user VARCHAR(36) NOT NULL,
	key VARCHAR NOT NULL,
	value VARCHAR DEFAULT '',

	PRIMARY KEY(user,key),
	UNIQUE(user,key),

	CONSTRAINT kvuser
		FOREIGN KEY(user) 
		REFERENCES users(name)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

CREATE TABLE plugin_kv (
	plugin VARCHAR,
	-- Plugins can optionally save keys by user, where the key
	-- is automatically life-cycled with the user
	user VARCHAR DEFAULT NULL,
	key VARCHAR NOT NULL,
	value VARCHAR DEFAULT '',

	PRIMARY KEY(plugin,user,key),
	UNIQUE(plugin,user,key),

	CONSTRAINT kvuser
		FOREIGN KEY(user) 
		REFERENCES users(name)
		ON UPDATE CASCADE
		ON DELETE CASCADE
);

`

// Create sets up a new heedy instance
func Create(a *assets.Assets) error {

	if a.Config.SQL == nil {
		return errors.New("Configuration does not specify an sql database")
	}

	// Split the sql string into database type and connection string
	sqlInfo := strings.SplitAfterN(*a.Config.SQL, "://", 2)
	if len(sqlInfo) != 2 {
		return errors.New("Invalid sql connection string")
	}
	sqltype := strings.TrimSuffix(sqlInfo[0], "://")

	if sqltype != "sqlite3" {
		return fmt.Errorf("Database type '%s' not supported", sqltype)
	}

	// We use the sql as location of our sqlite database
	sqlpath := a.Abs(sqlInfo[1])

	// Create any necessary directories
	sqlfolder := filepath.Dir(sqlpath)
	if err := os.MkdirAll(sqlfolder, 0750); err != nil {
		return err
	}

	db, err := sql.Open(sqltype, sqlpath)
	if err != nil {
		return err
	}

	_, err = db.Exec(schema)
	if err != nil {
		return err
	}

	if sqltype == "sqlite3" {
		_, err = db.Exec(sqliteAddonSchema)
		if err != nil {
			return err
		}
	}

	return db.Close()
}
