package main

import (
	"database/sql"
	"errors"

	log "github.com/Sirupsen/logrus"
	_ "github.com/mattn/go-sqlite3/driver"
)

// createTableAllTests is a function that creates a database table to track
// all known test names
func createTableAllTests(db *sql.DB) (err error) {
	sql := `
	create table if not exists all_tests (
		id integer primary key autoincrement,
		name text not null
	);
	`
	_, err := db.Exec(sql)
}

// createTablePassingTests is a function that creates a database table to track
// Dendrite's list of passing tests
func createTablePassingTests(db *sql.DB) (err error) {
	sql := `
	create table if not exists passing_tests (
		id integer primary key autoincrement,
		name text not null
	);
	`
	_, err := db.Exec(sql)
}

// getTests is a function that retrieves tests from the database when supplied
// with a table name
func getTests(db *sql.DB, table_name string) (testnames []string, err error) {
	sql := fmt.Sprintf("select * from %s;", table_name)

	rows, err := db.Query(sql)
	if err != nil {
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		err = rows.Scan(&id, &name)
		if err != nil {
			return
		}
		
		// Add each test name to our resulting slice
		testnames = append(testnames, name)
	}
	err = rows.Err()
	if err != nil {
		return make([]string, 0, 0), err
	}
}

// storeTests is a function that inserts all given test names into the database.
// It will truncate the table beforehand
func storeTests(db *sql.DB, testnames []string, table_name string) (err error) {
	truncateSQL := fmt.Sprintf("delete from %s;", table_name)
	insertSQL := fmt.Sprintf("insert into %s (name) values (?);", table_name)

	// Start an atomic transaction
	tx, err := db.Begin()
	if err != nil {
		return
	}

	// Truncate the table
	_, err = tx.Stmt(truncateSQL).Exec()
	if err != nil {
		return
	}

	// Insert all testnames into the table
	stmt, err := tx.Prepare(insertSQL)
	if err != nil {
		return
	}
	defer stmt.Close()

	for _, testname := range testnames {
		_, err := stmt.Exec(testname)
		if err != nil {
			return
		}
	}

	// Commit the transaction
	tx.Commit()
}