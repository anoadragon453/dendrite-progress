package main

import (
	"fmt"
	"database/sql"
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
	_, err = db.Exec(sql)

	return
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
	_, err = db.Exec(sql)

	return
}

// getTests is a function that retrieves tests from the database when supplied
// with a table name
func getTests(db *sql.DB, tableName string) (testnames []string, err error) {
	sql := fmt.Sprintf("select * from %s;", tableName)

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

	return
}

// storeTests is a function that inserts all given test names into the database.
// It will truncate the table beforehand
func storeTests(db *sql.DB, testnames []string, tableName string) (err error) {
	truncateSQL := fmt.Sprintf("delete from %s;", tableName)
	insertSQL := fmt.Sprintf("insert into %s (name) values (?);", tableName)

	// Start an atomic transaction
	tx, err := db.Begin()
	if err != nil {
		return
	}

	stmt, err := tx.Prepare(truncateSQL)
	if err != nil {
		return
	}

	// Truncate the table
	_, err = tx.Stmt(stmt).Exec()
	if err != nil {
		return
	}

	// Insert all testnames into the table
	stmt, err = tx.Prepare(insertSQL)
	if err != nil {
		return
	}
	defer stmt.Close()

	for _, testname := range testnames {
		_, err = stmt.Exec(testname)
		if err != nil {
			return
		}
	}

	// Commit the transaction
	tx.Commit()
	return
}
