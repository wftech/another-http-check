package main

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-ntlmssp"
)

const (
	// Authentication types
	AUTH_NONE  = 0
	AUTH_BASIC = 1
	AUTH_NTLM  = 2

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
}

// Check params
type Expected struct {
	StatusCodes []int
	BodyText    string
	SSLCheck    SSLCheck
}

func checkStatusCode(code int, e *Expected) bool {
	for _, expectedCode := range e.StatusCodes {
		if expectedCode == code {
			return true
		}
	}
	return false
}

func checkCerts(certs [][]*x509.Certificate, e *Expected) (string, int) {
	timeNow := time.Now()
	checkedCerts := make(map[string]bool)
	for _, chain := range certs {
		for _, cert := range chain {
			if _, checked := checkedCerts[string(cert.Signature)]; checked {
				continue
			}
			checkedCerts[string(cert.Signature)] = true
			expiresIn := int(cert.NotAfter.Sub(timeNow).Hours())
			if e.SSLCheck.DaysCritical > 0 && e.SSLCheck.DaysCritical*24 >= expiresIn {
				return fmt.Sprintf("CRITICAL - SSL cert expires in %f days", float32(expiresIn)/24), EXIT_CRITICAL
			}
			if e.SSLCheck.DaysWarning > 0 && e.SSLCheck.DaysWarning*24 >= expiresIn {
				return fmt.Sprintf("WARNING - SSL cert expires in %f dasy", float32(expiresIn)/24), EXIT_WARNING
			}
		}
	}
	return "", EXIT_OK
}

func Check(r *Request, e *Expected) (string, int, error) {
	if len(r.Host) == 0 && len(r.IPAddress) == 0 {
		return "UNKNOWN - No host or IP address given", EXIT_UNKNOWN, nil
	}

	var host string
	if len(r.Host) == 0 {
		host = r.IPAddress
	} else {
		host = r.Host
	}

	// Setup timeout
	timeout := time.Duration(time.Duration(r.Timeout) * time.Second)

	// Init client
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	url := fmt.Sprintf("%s://%s:%s%s", r.Scheme, host, strconv.Itoa(r.Port), r.URI)

	if r.Verbose {
		fmt.Println(">> URL: " + url)
	}

	// Prepare request
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		if r.Verbose {
			fmt.Println(fmt.Sprintf(">> http.NewRequest error: %v", err))

		}
		return "UNKNOWN", EXIT_UNKNOWN, err
	}

	// Authentication
	if r.Authentication.Type == AUTH_BASIC {
		request.SetBasicAuth(r.Authentication.User, r.Authentication.Password)
	}

	// TODO - test
	if r.Authentication.Type == AUTH_NTLM {
		transport := ntlmssp.Negotiator{
			RoundTripper: &http.Transport{},
		}
		client.Transport = transport
		request.SetBasicAuth(r.Authentication.User, r.Authentication.Password)
	}

	start := time.Now()
	timeInfo := func() string {
		return fmt.Sprintf("time=%fs", float32(time.Now().UnixNano()-start.UnixNano())/float32(1000000000))
	}
	res, err := client.Do(request)
	if err != nil {
		if r.Verbose {
			fmt.Println(fmt.Sprintf(">> client.GET error: %v", err))
		}
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return fmt.Sprintf("CRITICAL - Timeout - No response recieved in %d seconds|%s", r.Timeout, timeInfo()), EXIT_CRITICAL, nil
		}
		return fmt.Sprintf("CRITICAL - %s|%s", err.Error(), timeInfo()), EXIT_CRITICAL, nil
	}

	defer res.Body.Close()

	if r.Verbose {
		fmt.Println(fmt.Sprintf(">> Response status: %s", res.Status))
	}

	// Check status code
	if !checkStatusCode(res.StatusCode, e) {
		var expectedStatusCodes []string
		for _, code := range e.StatusCodes {
			expectedStatusCodes = append(expectedStatusCodes, strconv.Itoa(code))
		}
		return fmt.Sprintf("CRITICAL - Got  response HTTP/1.1 %s, expected %s|%s", strconv.Itoa(res.StatusCode), strings.Join(expectedStatusCodes, ", "), timeInfo()), EXIT_CRITICAL, nil
	}

	// Check body text
	if len(e.BodyText) > 0 {
		expectedText := []byte(e.BodyText)
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "UNKNOWN", EXIT_UNKNOWN, err
		}
		if !bytes.Contains(bodyBytes, expectedText) {
			return fmt.Sprintf("CRITICAL - String '%s' not found in body|%s", e.BodyText, timeInfo()), EXIT_CRITICAL, nil
		}
	}

	// Check SSL cert
	if e.SSLCheck.Run {
		SSLMsg, SSLExit := checkCerts(res.TLS.VerifiedChains, e)
		if SSLExit != EXIT_OK {
			return fmt.Sprintf("%s|%s", SSLMsg, timeInfo()), SSLExit, nil
		}
	}

	return fmt.Sprintf("OK - Got response HTTP/1.1 %s|%s", strconv.Itoa(res.StatusCode), timeInfo()), EXIT_OK, nil
}
