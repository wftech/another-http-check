package main

import (
	"bytes"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
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
	for _, expectdCode := range e.StatusCodes {
		if expectdCode == code {
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
			if e.SSLCheck.DaysCritical*24 >= expiresIn {
				fmt.Println(fmt.Sprintf("%d %d", e.SSLCheck.DaysCritical*24, expiresIn))
				return "CRITICAL", EXIT_CRITICAL
			}
			if e.SSLCheck.DaysWarning*24 >= expiresIn {
				return "WARNING", EXIT_WARNING
			}
		}
	}
	return "", EXIT_OK
}

func Check(r *Request, e *Expected) (string, int, error) {
	if len(r.Host) == 0 && len(r.IPAddress) == 0 {
		return "UNKNOWN", EXIT_UNKNOWN, nil
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

	res, err := client.Do(request)
	if err != nil {
		if r.Verbose {
			fmt.Println(fmt.Sprintf(">> client.GET error: %v", err))
		}
		return "CRITICAL", EXIT_CRITICAL, err
	}

	defer res.Body.Close()

	if r.Verbose {
		fmt.Println(fmt.Sprintf(">> Response status: %s", res.Status))
	}

	// Check status code
	if !checkStatusCode(res.StatusCode, e) {
		return "CRITICAL", EXIT_CRITICAL, nil
	}

	// Check body text
	if len(e.BodyText) > 0 {
		expectedText := []byte(e.BodyText)
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return "CRITICAL", EXIT_CRITICAL, nil
		}
		if !bytes.Contains(bodyBytes, expectedText) {
			return "CRITICAL", EXIT_CRITICAL, nil
		}
	}

	// Check SSL cert
	if e.SSLCheck.Run {
		SSLMsg, SSLExit := checkCerts(res.TLS.VerifiedChains, e)
		if SSLExit != EXIT_OK {
			return SSLMsg, SSLExit, nil
		}
	}

	return "OK", EXIT_OK, nil
}
