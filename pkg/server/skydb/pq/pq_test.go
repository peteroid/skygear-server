// Copyright 2015-present Oursky Ltd.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pq

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/skygeario/skygear-server/pkg/server/skydb"
)

// NOTE(limouren): postgresql uses this error to signify a non-exist
// schema
func isInvalidSchemaName(err error) bool {
	if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "3F000" {
		return true
	}

	return false
}

func testAppName() string {
	// Generate a random app name so that schema is different each time.
	//
	// This is a workaround for the issue that schema is not reliably
	// created during testing. c.f. SkygearIO/skygear-server#171
	return fmt.Sprintf("io.skygear.test.%d", rand.Int())
}

func getTestConn(t *testing.T) *conn {
	defaultTo := func(envvar string, value string) {
		if os.Getenv(envvar) == "" {
			os.Setenv(envvar, value)
		}
	}
	defaultTo("PGDATABASE", "skygear_test")
	defaultTo("PGSSLMODE", "disable")
	appName := testAppName()
	c, err := Open(appName, skydb.RoleBasedAccess, "", true)
	if err != nil {
		t.Fatal(err)
	}

	// create schema
	err = mustInitDB(c.(*conn).Db().(*sqlx.DB), appName, true)
	if err != nil {
		t.Fatal(err)
	}
	return c.(*conn)
}

func cleanupConn(t *testing.T, c *conn) {
	schemaName := fmt.Sprintf("app_%s", toLowerAndUnderscore(c.appName))
	_, err := c.db.Exec(fmt.Sprintf("DROP SCHEMA if exists %s CASCADE", schemaName))
	if err != nil && !isInvalidSchemaName(err) {
		t.Fatal(err)
	}
}

func addUser(t *testing.T, c *conn, userid string) {
	_, err := c.Exec("INSERT INTO _user (id, password) VALUES ($1, 'somepassword')", userid)
	if err != nil {
		t.Fatal(err)
	}
}

func addUserWithInfo(t *testing.T, c *conn, userid string, email string) {
	_, err := c.Exec("INSERT INTO _user (id, password, email) VALUES ($1, 'somepassword', $2)", userid, email)
	if err != nil {
		t.Fatal(err)
	}
}

func addUserWithUsername(t *testing.T, c *conn, userid string, username string) {
	_, err := c.Exec("INSERT INTO _user (id, password, username) VALUES ($1, 'somepassword', $2)", userid, username)
	if err != nil {
		t.Fatal(err)
	}
}

type execor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

func insertRow(t *testing.T, db execor, query string, args ...interface{}) {
	result, err := db.Exec(query, args...)
	if err != nil {
		t.Fatal(err)
	}

	n, err := result.RowsAffected()
	if err != nil {
		t.Fatal(err)
	}

	if n != 1 {
		t.Fatalf("got rows affected = %v, want 1", n)
	}
}

func exhaustRows(rows *skydb.Rows, errin error) (records []skydb.Record, err error) {
	if errin != nil {
		err = errin
		return
	}

	for rows.Scan() {
		records = append(records, rows.Record())
	}

	err = rows.Err()
	return
}
