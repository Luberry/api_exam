// vim: set ts=2 sw=2 :
package main

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	flag "github.com/namsral/flag"
	"github.com/sirupsen/logrus"
)

const (
	// ID is the constant for the internal id field
	ID = "INTERNAL_ID"
	// FName is the constant for the first name field
	FName = "FIRST_NAME"
	// MName is the constant for the middle name field
	MName = "MIDDLE_NAME"
	// LName is the constant for the last name field
	LName = "LAST_NAME"
	// Phone is the constant for the phone number field
	Phone = "PHONE_NUM"
)

// Name provides fields describing a name for our output schema
type Name struct {
	// First is the first name
	First string `json:"first"`
	// Middle is the middle name which can be empty
	Middle string `json:"middle,omitempty"`
	// Last is the last name
	Last string `json:"last"`
}

// Record provides the fields describing the record we wish to output
type Record struct {
	// InternalID is the 8 digit positive id number for our record
	InternalID int `json:"id"`
	// Name is the person's name
	Name Name `json:"name"`
	// Phone is the person's phone number
	Phone string `json:"phone"`
}

var (
	inputDir    string
	outputDir   string
	errorDir    string
	logLevel    string
	phoneRegexp *regexp.Regexp = regexp.MustCompile("^\\d{3}-\\d{3}-\\d{4}$")
)

// init sets up our flags and initializes our logger
func init() {
	flag.StringVar(&inputDir, "input-directory", "./input", "directory to watch for new `csv` files.")
	flag.StringVar(&outputDir, "output-directory", "./output", "directory to output json files to")
	flag.StringVar(&errorDir, "error-directory", "./errors", "directory to output error files to")
	flag.StringVar(&logLevel, "log-level", "info", "log level can be one of (panic,error,warn,info,debug,trace)")
	flag.Parse()
	logrus.SetFormatter(&logrus.JSONFormatter{})
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.WithError(err).Fatal()
	}
	logrus.SetLevel(level)
}

func main() {
	// create a new file watcher using fsnotify to watch for new files
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.WithError(err).Fatal("could not initialize watcher")
	}
	defer watcher.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				logrus.WithField("event", event).Trace("received file event")
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
					// handle write or create file events and send them to our file processor
					logrus.WithField("file", event.Name).Info("processing csv file")
					err = processFile(event.Name)
					if err != nil {
						logrus.WithField("file", event.Name).WithError(err).Error()
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				logrus.WithError(err).Error()
			}
		}
	}()
	// process initial files since fsnotify only looks for new files, and there may be files already
	// in the directory
	files, err := filepath.Glob(filepath.Join(inputDir, "*.csv"))
	if err != nil {
		logrus.WithError(err).Fatal()
	}
	for _, file := range files {
		logrus.WithField("file", file).Info("processing csv file")
		err = processFile(file)
		if err != nil {
			logrus.WithField("file", file).WithError(err).Error()
		}
	}

	// set up file watcher to watch our input dir for any new files
	err = watcher.Add(inputDir)
	if err != nil {
		logrus.WithError(err).WithField("input-directory", inputDir).Fatal("could not watch directory")
	}

	wg.Wait()
}

// processFile will process a file at the given path and return an error if it cannot process the file
func processFile(path string) error {
	var records []Record
	// prepare the error list for export to csv if needed
	errs := [][]string{
		{
			"LINE_NUM",
			"ERROR_MSG",
		},
	}

	// create a small helper function to save on repeated code when handling errors
	errFunc := func(line int, err error) {
		logrus.WithError(err).WithField("lineNumber", line).Error()
		errs = append(errs, []string{fmt.Sprintf("%d", line), err.Error()})
	}

	ext := filepath.Ext(path)

	if strings.ToLower(ext) == ".csv" {
		// open the csv file for reading
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		r := csv.NewReader(f)

		var header map[string]int
		// loop through the csv file line by line and process each line
		for line := 1; ; line++ {
			record, err := r.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				errFunc(line, err)
				continue
			}
			// create a map to store the table column positions using the data in the first row
			if header == nil {
				header = make(map[string]int)
				for i, val := range record {
					header[val] = i
				}
				continue
			}
			// make sure the record has the same amount of fields as the header
			if len(record) != len(header) {
				errFunc(line, fmt.Errorf("err: number of fields: %d does not match header: %d", len(record), len(header)))
				continue
			}
			rec := Record{}
			id, exists := header[ID]
			if !exists {
				errFunc(line, fmt.Errorf("err: missing: %q error field in header", ID))
				continue
			}
			if len(record[id]) != 8 {
				errFunc(line, fmt.Errorf("err: id field: %q is not an 8 digit integer", record[id]))
				continue
			}
			rec.InternalID, err = strconv.Atoi(record[id])
			if err != nil {
				errFunc(line, fmt.Errorf("err: id field: %q either empty, or an invalid integer, %v", record[id], err))
				continue
			}
			if rec.InternalID < 0 {
				errFunc(line, fmt.Errorf("err: id: %d should not be negative", rec.InternalID))
				continue
			}

			id, exists = header[FName]
			if !exists {
				errFunc(line, fmt.Errorf("err: missing: %q error field in header", FName))
				continue
			}
			if len(record[id]) > 15 {
				errFunc(line, fmt.Errorf("err: first name field: %q should not exceed 15 characters", record[id]))
				continue
			}
			rec.Name.First = record[id]
			if rec.Name.First == "" {
				errFunc(line, errors.New("err: first name field should not be empty"))
				continue
			}
			id, exists = header[MName]
			if !exists {
				errFunc(line, fmt.Errorf("err: missing: %q error field in header", MName))
				continue
			}
			if len(record[id]) > 15 {
				errFunc(line, fmt.Errorf("err: middle name field: %q should not exceed 15 characters", record[id]))
				continue
			}
			rec.Name.Middle = record[id]
			id, exists = header[LName]
			if !exists {
				errFunc(line, fmt.Errorf("err: missing: %q error field in header", LName))
				continue
			}
			if len(record[id]) > 15 {
				errFunc(line, fmt.Errorf("err: last name field: %q should not exceed 15 characters", record[id]))
				continue
			}
			rec.Name.Last = record[id]
			if rec.Name.Last == "" {
				errFunc(line, errors.New("err: last name field should not be empty"))
				continue
			}
			id, exists = header[Phone]
			if !exists {
				errFunc(line, fmt.Errorf("err: missing: %q error field in header", Phone))
				continue
			}
			rec.Phone = record[id]
			if !phoneRegexp.MatchString(rec.Phone) {
				errFunc(line, fmt.Errorf("err: phone field: %q either empty, or an invalid phone number", rec.Phone))
				continue
			}
			// add the processed record to our output list
			records = append(records, rec)
		}
		// close out the file
		f.Close()

		// marshal our records to json
		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return err
		}
		// write our json data to the output directory
		err = ioutil.WriteFile(
			filepath.Join(
				outputDir,
				strings.Join([]string{strings.TrimSuffix(filepath.Base(path), ext), "json"}, "."),
			), data, 0755)
		if err != nil {
			return err
		}
		// if our errors list has errors in it write the error csv to the errors dir
		if len(errs) > 1 {
			errFile, err := os.OpenFile(
				filepath.Join(errorDir, filepath.Base(path)),
				os.O_RDWR|os.O_CREATE|os.O_TRUNC,
				0755,
			)
			if err != nil {
				logrus.WithError(err).Fatal()
			}
			defer errFile.Close()
			w := csv.NewWriter(errFile)
			err = w.WriteAll(errs)
			if err != nil {
				logrus.WithError(err).Fatal()
			}
		}
		// remove the processed file
		err = os.Remove(path)
		if err != nil {
			return err
		}
	}
	return nil
}
