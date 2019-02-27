package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const (
	// Authentication types
	AUTH_NONE = 0
	AUTH_BASE = 1
	AUTH_NTML = 2

	// Exit codes
	EXIT_OK       = 0
	EXIT_WARNING  = 1
	EXIT_CRITICAL = 2
	EXIT_UNKNOWN  = 3
)

// Authentication
type Authentication struct {
	Type     int
	User     string
	Password string
}

type SSLCheck struct {
	Run          bool
	DaysWarning  int
	DaysCritical int
}

// Request
type Request struct {
	Scheme         string
	Host           string
	IPAddress      string
	TLS            bool
	Port           int
	URI            string
	Timeout        int
	Verbose        bool
	Authentication Authentication
	SSLCheck       SSLCheck
}

// Check params
type Expected struct {
	StatusCodes []int
	BodyText    string
}

func checkStatusCode(code int, e *Expected) bool {
	for _, expectdCode := range e.StatusCodes {
		if expectdCode == code {
			return true
		}
	}
	return false
}

func Check(r *Request, e *Expected) (string, int, error) {
	// Setup timeout
	timeout := time.Duration(time.Duration(r.Timeout) * time.Second)

	// Init client
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	url := fmt.Sprintf("%s://%s:%s%s", r.Scheme, r.Host, strconv.Itoa(r.Port), r.URI)

	if r.Verbose {
		fmt.Println(">> URL: " + url)
	}

	res, err := client.Get(url)
	if err != nil {
		if r.Verbose {
			fmt.Println(fmt.Sprintf(">> client.GET error: %v", err))
		}
		return "CRITICAL", EXIT_CRITICAL, err
	}

	if r.Verbose {
		fmt.Println(fmt.Sprintf(">> Response status: %s", res.Status))
	}

	if !checkStatusCode(res.StatusCode, e) {
		return "CRITICAL", EXIT_CRITICAL, nil
	}

	return "OK", EXIT_OK, nil
}
