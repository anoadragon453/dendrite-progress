/*
	Dendrite Progress
	Show development progress of the matrix homeserver, Dendrite.

	Andrew Morgan 2019
*/

package main

import (
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v5/github"

	// Only need to import once to load the driver hooks
	_ "github.com/mattn/go-sqlite3"
)

var (
	// TODO: Set up a config file
	HTTP_PORT             = 8765
	DENDRITE_TESTFILE_URL = "https://raw.githubusercontent.com/matrix-org/dendrite/master/testfile"
	SYTEST_GIT_URL        = "https://github.com/matrix-org/sytest"
	SYTEST_GIT_DIR, _     = filepath.Abs("sytest")
	WEBHOOK_SECRET        = "xxx"
	DATABASE_PATH         = "stats.db"
	LOG_LEVEL             = log.DebugLevel

	hook, _ = github.New(github.Options.Secret(WEBHOOK_SECRET))
	db      *sql.DB
)

// getPassingTests downloads
func getPassingTests() (testnames []string, err error) {
	log.Debug("Getting passing tests")

	// Download the latest iteration of the testfile
	resp, err := http.Get(DENDRITE_TESTFILE_URL)
	if err != nil {
		return
	}

	// Retrieve the response body
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}

	// Get each individual test name
	testnames = strings.Split(string(body), "\n")
	log.Debugf("Got passing count: %d", len(testnames))
	return
}

// getAllTests fetches SyTest's source code then scans through it, enumerating
// all of the test names
func getAllTests() (testnames []string, err error) {
	log.Debug("Getting all tests")

	// Check if the sytest checkout exists already
	_, err = os.Stat(SYTEST_GIT_DIR)
	if os.IsNotExist(err) {
		// Checkout the source
		log.Debug("Cloning sytest...")
		err = cloneSytest(SYTEST_GIT_DIR, SYTEST_GIT_URL)
		if err != nil {
			return
		}
	} else {
		// Make sure the checkout is up-to-date
		log.Debug("Updating sytest checkout...")
		err = pullSytest(SYTEST_GIT_DIR)
		if err != nil {
			return
		}
	}

	log.Debug("Sytest checkout updated.")

	// Read through all test files and check for test names
	testfilePaths := []string{}
	err = filepath.Walk(SYTEST_GIT_DIR+"/tests", func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			testfilePaths = append(testfilePaths, path)
		}
		return nil
	})
	if err != nil {
		return
	}

	testnames = make([]string, 1000)
	for _, testfilePath := range testfilePaths {
		testfile, err := os.Open(testfilePath)
		if err != nil {
			return make([]string, 0, 0), err
		}
		defer testfile.Close()

		testfileContent, err := ioutil.ReadAll(testfile)
		if err != nil {
			return make([]string, 0, 0), err
		}
		testfileLines := strings.Split(string(testfileContent), "\n")

		for _, line := range testfileLines {
			if strings.HasPrefix(line, "test \"") {
				testname := line[6 : len(line)-1]
				testnames = append(testnames, testname)
			}
		}
	}
	log.Debugf("Got total test count: %d", len(testnames))

	return
}

// refreshPassingTests is a function that retrieves the number of tests that
// Dendrite passes and saves the count to the database and prometheus metrics
func refreshPassingTests() (err error) {
	// Save passing tests
	passingTests, err := getPassingTests()
	if err != nil {
		return
	}
	err = storeTests(db, passingTests, "passing_tests")
	if err != nil {
		return
	}
	setPassingTests(len(passingTests))

	return
}

// refreshTotalTests is a function that retrieves the total number of tests and
// saves the count to the database and prometheus metrics
func refreshTotalTests() (err error) {
	// Save all tests
	allTests, err := getAllTests()
	if err != nil {
		return
	}
	err = storeTests(db, allTests, "all_tests")
	if err != nil {
		return
	}
	setTotalTests(len(allTests))

	return
}

// handleDendriteWebhook is a http.Handler function that listens for webhook events sent
// from Dendrite every time a commit occurs
func handleDendriteWebhook(w http.ResponseWriter, req *http.Request) {
	// Ensure this is an authenticated webhook request
	payload, err := hook.Parse(req, github.PushEvent)
	if err != nil {
		log.Error("[dendrite webhook handler] %s", err)
		return
	}

	// Act according to the payload type
	switch payload.(type) {
	case github.PushPayload:
		// Refresh data on every push event
		refreshPassingTests()
	default:
		log.Debug("[dendrite webhook handler] Unhandled webhook request type: %s", payload)
	}
}

// handleSytestWebhook is a http.Handler function that listens for webhook events sent
// from Sytest every time a commit occurs
func handleSytestWebhook(w http.ResponseWriter, req *http.Request) {
	// Ensure this is an authenticated webhook request
	payload, err := hook.Parse(req, github.PushEvent)
	if err != nil {
		log.Error("[sytest webhook handler] %s", err)
		return
	}

	// Act according to the payload type
	switch payload.(type) {
	case github.PushPayload:
		// Refresh data on every push event
		refreshTotalTests()
	default:
		log.Debug("[sytest webhook handler] Unhandled webhook request type: %s", payload)
	}
}

// setupDB is a function that opens a connection to the database and ensures the
// correct tables exist
func setupDB() {
	var err error
	db, err = sql.Open("sqlite3", DATABASE_PATH)
	if err != nil {
		log.Fatalf("Unable to open database file %s: %q\n", DATABASE_PATH, err)
	}

	err = createTableAllTests(db)
	if err != nil {
		log.Fatalf("Issue creating all tests database table: %q", err)
	}

	err = createTablePassingTests(db)
	if err != nil {
		log.Fatalf("Issue creating passing tests database table: %q", err)
	}

	// Pull latest changes from the db
	log.Debug("Retrieving latest changes...")
	refreshPassingTests()
	refreshTotalTests()
	log.Debug("Done retrieving latest changes.")
}

func main() {
	log.SetLevel(LOG_LEVEL)

	// Create database connection and tables
	setupDB()

	// Listen for webhook requests
	http.HandleFunc("/dendrite-webhook", handleDendriteWebhook)
	http.HandleFunc("/sytest-webhook", handleSytestWebhook)

	// Listen for prometheus metrics request
	http.Handle("/metrics", serveMetrics())

	// Start the HTTP server
	port := fmt.Sprintf(":%d", HTTP_PORT)
	log.Fatal(http.ListenAndServe(port, nil))
}
