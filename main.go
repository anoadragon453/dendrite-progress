/* 
	Dendrite Progress
	Show development progress of the matrix homeserver, Dendrite.

	Andrew Morgan 2019
*/

package main

import (
	"database/sql"
	"net/http"
	"path/filepath"
	"io"
	"io/ioutil"
	"os/exec"
	"os"
	"fmt"
	"strings"
	
	log "github.com/sirupsen/logrus"
	"gopkg.in/go-playground/webhooks.v5/github"

	// Only need to import once to load the driver hooks
	_ "github.com/mattn/go-sqlite3"
)

var (
	// TODO: Set up a config file
	HTTP_PORT = 8765
	DENDRITE_TESTFILE_URL = "https://raw.githubusercontent.com/matrix-org/dendrite/master/testfile"
	SYTEST_GIT_URL = "https://github.com/matrix-org/sytest"
	SYTEST_GIT_DIR, _ = filepath.Abs("sytest")
	WEBHOOK_SECRET = "xxx"
	DATABASE_PATH = "stats.db"

	hook, _ = github.New(github.Options.Secret(WEBHOOK_SECRET))
	db *sql.DB
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
		log.Debug("Cloning sytest...")

		// Checkout the source
		cmd := exec.Command(
			fmt.Sprintf("git clone %s %s", SYTEST_GIT_URL, SYTEST_GIT_DIR),
		)
		err = cmd.Run()
		if err != nil {
			return
		}
	} else {
		log.Debug("Updating sytest checkout...")

		// Make sure the checkout is up-to-date
		cmd := exec.Command(
			fmt.Sprintf(`git -C "%s" pull`, SYTEST_GIT_DIR), os.Getenv("PATH"),
		)
		err = cmd.Run()
		if err != nil {
			return
		}
	}

	log.Debug("Sytest checkout updated...")

	// Read through all test files and check for test names
	testfilePaths, err := ioutil.ReadDir(SYTEST_GIT_DIR + "/tests")
	if err != nil {
		return
	}
	log.Debug("Got testfilePaths: %s", testfilePaths)

	testnames = make([]string, 1000)
	for _, testfilePathInfo := range testfilePaths {
		testfile, err := os.Open(testfilePathInfo.Name())
		if err != nil {
			return make([]string, 0, 0), err
		}
		defer testfile.Close()

		testfileContent, err := ioutil.ReadAll(testfile)
		if err != nil {
			return make([]string, 0, 0), err
		}
		testfileLines:= strings.Split(string(testfileContent), "\n")

		for _, line := range testfileLines {
			if strings.HasPrefix(line, "test \"") {
				testname := line[6:len(line)-1]
				testnames = append(testnames, testname)
			}
		}
	}
	log.Debugf("Got all count: %d", len(testnames))

	return
}

// refreshProgressData kicks off a refresh off all statistical data sources
func refreshProgressData() (err error) {
	// Save passing tests
	passingTests, err := getPassingTests()
	if err != nil {
		return
	}
	err = storeTests(db, passingTests, "all_tests")
	if err != nil {
		return
	}

	// Save all tests
	allTests, err := getAllTests()
	if err != nil {
		return
	}
	err = storeTests(db, allTests, "all_tests")
	if err != nil {
		return
	}

	return
}

// handleWebhook is a http.Handler function that listens for webhook events sent
// from Github every time a commit occurs
func handleWebhook(w http.ResponseWriter, req *http.Request) {
	// Ensure this is an authenticated webhook request
	payload, err := hook.Parse(req, github.PushEvent)
	if err != nil {
		log.Error("[webhook handler] %s", err)
		return
	}

	// Act according to the payload type
	switch payload.(type) {
	case github.PushPayload:
		// Refresh data on every push event
		refreshProgressData()
	default:
		log.Info("[webhook handler] Unhandled webhook request type: %s", payload)
	}
}

// serveStats is a http.Handler function that serves a templated webpage of
// statistics
func serveStats(w http.ResponseWriter, req *http.Request) {
	// Retrieve data from DB
	allTests, err := getTests(db, "all_tests")
	if err != nil {
		log.Fatalf("Error retrieving all tests: %q", err)
	}
	passingTests, err := getTests(db, "passing_tests")
	if err != nil {
		log.Fatalf("Error retrieving passing tests: %q", err)
	}

	io.WriteString(w, fmt.Sprintf("%d/%d", len(allTests), len(passingTests)))
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
}

func boop(w http.ResponseWriter, req *http.Request) {
	err := refreshProgressData()
	if err != nil {
		log.Errorf("Error updating progress data: %q", err)
	}
}

func main() {
	log.SetLevel(log.DebugLevel)
	// Create database connection and tables
	setupDB()

	// Serve statistics at root
	http.HandleFunc("/", serveStats)

	http.HandleFunc("/boop", boop)

	// Listen for webhook requests
	http.HandleFunc("/webhook", handleWebhook)

	// Start the HTTP server
	port := fmt.Sprintf(":%d", HTTP_PORT)
	log.Fatal(http.ListenAndServe(port, nil))
}
