package main

import (
	"bytes"
	"crypto/tls"
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

type ClientCert struct {
	ClientCertFile string
	PrivateKeyFile string
}

// Request
type Request struct {
	Scheme           string
	Host             string
	IPAddress        string
	TLS              bool
	Port             int
	URI              string
	Timeout          int
	Verbose          bool
	SSLNoVerify      bool
	Authentication   Authentication
	FollowRedirects  bool
	WarningTimeout   int
	CriticalTimeout  int
	NoSNI            bool
	ClientCert       ClientCert
	TLSRenegotiation bool
}

// Check params
type Expected struct {
	StatusCodes []int
	BodyText    string
	SSLCheck    SSLCheck
}

// Lookup map for auth type names
var authLookup = map[int]string{
	AUTH_NONE:  "none",
	AUTH_BASIC: "basic auth",
	AUTH_NTLM:  "NTLM auth",
}

// URL getter
func (r Request) GetURL() string {
	var host string
	if len(r.IPAddress) > 0 {
		host = r.IPAddress
	} else {
		host = r.Host
	}
	return fmt.Sprintf("%s://%s:%s%s", r.Scheme, host, strconv.Itoa(r.Port), r.URI)
}

// Use timeout interval
func (r Request) UseTimoutInterval() bool {
	return r.WarningTimeout > 0 && r.CriticalTimeout > 0 && r.WarningTimeout < r.CriticalTimeout
}

// Status code check helper
func checkStatusCode(code int, e *Expected) bool {
	for _, expectedCode := range e.StatusCodes {
		if expectedCode == code {
			return true
		}
	}
	return false
}

// Certificate check helper
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
				return fmt.Sprintf("WARNING - SSL cert expires in %f days", float32(expiresIn)/24), EXIT_WARNING
			}
		}
	}
	return "", EXIT_OK
}

// TLS config factory
func getTLSConfig(r *Request) (*tls.Config, error) {
	TLSConfig := &tls.Config{}

	// InsecureSkipVerify
	if r.SSLNoVerify {
		TLSConfig.InsecureSkipVerify = true
	}

	// SNI
	if !r.NoSNI && len(r.Host) > 0 {
		TLSConfig.ServerName = r.Host
	}

	// Client cert
	if r.ClientCert.ClientCertFile != "" && r.ClientCert.PrivateKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(r.ClientCert.ClientCertFile, r.ClientCert.PrivateKeyFile)
		if err != nil {
			return nil, err
		}
		TLSConfig.Certificates = []tls.Certificate{cert}
	}

	// Renegotiation
	if r.TLSRenegotiation {
		TLSConfig.Renegotiation = tls.RenegotiateOnceAsClient
	}

	return TLSConfig, nil
}

// HTTP client factory
func initHTTPClient(r *Request) (*http.Client, error) {
	// Get TLS config
	TLSConfig, err := getTLSConfig(r)
	if err != nil {
		return nil, err
	}

	http.DefaultTransport.(*http.Transport).TLSClientConfig = TLSConfig

	// Setup timeout
	var timeout time.Duration
	if r.UseTimoutInterval() {
		timeout = time.Duration(time.Duration(r.CriticalTimeout) * time.Second)
	} else {
		timeout = time.Duration(time.Duration(r.Timeout) * time.Second)
	}

	// Init client
	client := &http.Client{
		Timeout: timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if r.FollowRedirects {
				return nil
			} else {
				return http.ErrUseLastResponse
			}
		},
	}

	return client, nil
}

// Adds custom User-Agent header
func setUserAgent(request *http.Request) {
	request.Header.Set("User-Agent", fmt.Sprintf("icinga-http-check/%s Go-http-client/%s", appVersion, goVersion))
}

// Main check function
func Check(r *Request, e *Expected) (string, int, error) {
	if len(r.Host) == 0 && len(r.IPAddress) == 0 {
		return "UNKNOWN - No host or IP address given", EXIT_UNKNOWN, nil
	}

	client, err := initHTTPClient(r)
	if err != nil {
		return "CRITICAL", EXIT_CRITICAL, err
	}

	url := r.GetURL()

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

	// User agent
	setUserAgent(request)

	// Authentication
	if r.Authentication.Type == AUTH_BASIC {
		request.SetBasicAuth(r.Authentication.User, r.Authentication.Password)
	}

	// TODO - test
	if r.Authentication.Type == AUTH_NTLM {
		// Get TLS config
		TLSConfig, err := getTLSConfig(r)
		if err != nil {
			return "CRITICAL", EXIT_CRITICAL, err
		}
		transport := ntlmssp.Negotiator{
			RoundTripper: &http.Transport{
				TLSClientConfig: TLSConfig,
			},
		}
		client.Transport = transport
		request.SetBasicAuth(r.Authentication.User, r.Authentication.Password)
	}

	// SNI
	if !r.NoSNI && len(r.Host) > 0 {
		request.Host = r.Host
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
			var timeout int
			if r.UseTimoutInterval() {
				timeout = r.CriticalTimeout
			} else {
				timeout = r.Timeout
			}
			return fmt.Sprintf("CRITICAL - Timeout - No response recieved in %d seconds|%s", timeout, timeInfo()), EXIT_CRITICAL, nil
		}
		return fmt.Sprintf("CRITICAL - %s|%s", err.Error(), timeInfo()), EXIT_CRITICAL, nil
	}

	defer res.Body.Close()

	// Timeout interval
	if r.UseTimoutInterval() {
		delta := float32(time.Now().UnixNano()-start.UnixNano()) / float32(1000000000)
		if delta >= float32(r.WarningTimeout) {
			return fmt.Sprintf("WARNING - Timeout - No response recieved in %d seconds|%s", r.WarningTimeout, timeInfo()), EXIT_WARNING, nil
		}
	}

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

// Detects auth type
func DetectAuthType(r *Request) int {
	client, err := initHTTPClient(r)
	if err != nil {
		// `Check` should handle all errors
		return AUTH_NONE
	}

	url := r.GetURL()

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		// `Check` should handle all errors
		return AUTH_NONE
	}

	// User agent
	setUserAgent(request)

	// SNI
	if !r.NoSNI && len(r.Host) > 0 {
		request.Host = r.Host
	}

	res, err := client.Do(request)
	if err != nil {
		// `Check` should handle all errors
		return AUTH_NONE
	}
	defer res.Body.Close()

	authHeaders, ok := res.Header["Www-Authenticate"]
	if ok {
		authHeader := strings.ToLower(authHeaders[0])
		if strings.HasPrefix(authHeader, "ntlm") || strings.HasPrefix(authHeader, "negotiate") {
			return AUTH_NTLM
		}
		if strings.HasPrefix(authHeader, "basic") {
			return AUTH_BASIC
		}
	}

	return AUTH_NONE
}
