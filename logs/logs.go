package logs

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/icza/backscanner"
)

type NotFoundError struct {
	Message string
}

func (e *NotFoundError) Error() string {
	return e.Message
}

func GetTimeOfTheLastRequestFromLogs(userKey string) (time.Time, error) {
	last, err := findLastRequestLogRecord(userKey)
	if _, ok := err.(*NotFoundError); ok {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("getTimeOfTheLastRequestFromLogs: error finding last request time: %v", err)
	}
	timeStr, err := substringUntilFirstSub(last, " [")
	if err != nil {
		return time.Time{}, fmt.Errorf("getTimeOfTheLastRequestFromLogs: error parsing last request time: %v", err)
	}
	reqTime, err := parseTime(timeStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("checkLastRequestTime: error parsing last request time: %v", err)
	}
	return reqTime, nil
}

func findLastRequestLogRecord(userKey string) (string, error) {
	f, err := os.Open(fmt.Sprintf("./users/%v/frps.log", userKey))
	if err != nil {
		return "", fmt.Errorf("findLastRequestLogRecord: error opening file: %v", err)
	}
	fi, err := f.Stat()
	if err != nil {
		return "", fmt.Errorf("findLastRequestLogRecord: error getting file info: %v", err)
	}
	defer f.Close()

	scanner := backscanner.New(f, int(fi.Size()))
	what := []byte("get new HTTP request host")
	for {
		line, _, err := scanner.LineBytes()
		if err != nil {
			return "", &NotFoundError{Message: fmt.Sprintf("findLastRequestLogRecord: error opening file: %v", err)}
		}
		if bytes.Contains(line, what) {
			//log.Printf("	Found %q at line position: %d, line: %q\n", what, pos, line)
			return string(line), nil
		}
	}
}

func substringUntilFirstSub(input string, sub string) (string, error) {
	subIndex := strings.Index(input, sub)

	if subIndex == -1 {
		return "", errors.New("substringUntilFirstDot: dot character not found")
	}

	substring := input[:subIndex]

	return substring, nil
}

func parseTime(timeString string) (time.Time, error) {
	layout := "2006/01/02 15:04:05"
	parsedTime, err := time.Parse(layout, timeString)
	if err != nil {
		return time.Time{}, fmt.Errorf("parseTime: error parsing time: %v", err)
	}
	return parsedTime, nil
}

// func getUserKeyFromTheLastRequest() (string, error) {
// 	last, err := findLastRequestLogRecord()
// 	if err != nil {
// 		return "", fmt.Errorf("getUserKeyFromTheLastRequest: error finding last request time: %v", err)
// 	}
// 	address, err := between(last, "host [", "] path")
// 	if err != nil {
// 		return "", fmt.Errorf("getUserKeyFromTheLastRequest: error parsing last request time: %v", err)
// 	}
// 	key, err := substringUntilFirstSub(address, ".")
// 	if err != nil {
// 		return "", fmt.Errorf("getUserKeyFromTheLastRequest: error parsing address: %v", err)
// 	}
// 	return key, nil
// }
