package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Host            string `short:"H" description:"Host ex. google.com" default:""`
	IPAddress       string `short:"I" description:"IPv4 address ex. 8.8.4.4" default:""`
	URI             string `short:"u" long:"uri" description:"URI to check" default:"/"`
	Port            int    `short:"p" description:"Port ex. 80 for HTTP 443 for HTTPS" default:"80"`
	SSL             bool   `short:"S" long:"tls" description:"Use HTTPS"`
	Timeout         int    `short:"t" long:"timeout" description:"Timeout" default:"30"`
	AuthBasic       bool   `long:"auth-basic" description:"Use bacis auth"`
	AuthNtlm        bool   `long:"auth-ntlm" description:"Use NTLM auth"`
	Auth            string `short:"a" long:"auth" description:"ex. user:password" default:""`
	ExpectedCode    string `short:"e" long:"expect" description:"Expected HTTP code" default:"200"`
	BodyText        string `short:"s" long:"string" description:"Search for given string in response body" default:""`
	SSLExpiration   string `short:"C" description:"Check SSL cert expiration" default:""`
	SSLNoVerify     bool   `short:"k" long:"insecure" description:"Controls whether a client verifies the server's certificate chain and host name"`
	Verbose         bool   `short:"v" long:"verbose" description:"Verbose mode"`
	GuessAuth       bool   `long:"guess-auth" description:"Guess auth type"`
	FollowRedirects bool   `long:"follow-redirects" description:"Follow redirects"`
	WarningTimeout  int    `short:"w" description:"Warning timeout" default:"0"`
	CriticalTimeout int    `short:"c" description:"Critical timeout" default:"0"`
	NoSNI           bool   `long:"no-sni" description:"Do not use SNI"`
	ClientCertFile  string `short:"J" long:"client-cert" description:"Name of file containing the client certificate (PEM format) to be used in establishing the SSL session"`
	PrivateKeyFile  string `short:"K" long:"private-key" description:"Name of file containing the private key (PEM format) matching the client certificate"`
}

var options Options
var parser = flags.NewParser(&options, flags.Default)
var appVersion string
var goVersion string

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			os.Exit(1)
		}
	}

	var scheme string
	if options.Port == 443 || options.SSL {
		scheme = "https"
	} else {
		scheme = "http"
	}

	port := options.Port
	if scheme == "https" && port == 80 {
		port = 443
	}

	authType := AUTH_NONE
	if options.AuthBasic {
		authType = AUTH_BASIC
	}
	if options.AuthNtlm {
		authType = AUTH_NTLM
	}

	var authUser string
	var authPassword string

	if strings.Contains(options.Auth, ":") {
		authParts := strings.Split(options.Auth, ":")
		if len(authParts) != 2 {
			fmt.Println("UNKNOWN - Username and password not given: provide -a|--auth username:password")
			os.Exit(EXIT_UNKNOWN)
		}
		authUser = authParts[0]
		authPassword = authParts[1]
	}

	if authType == AUTH_NONE && len(authUser) > 0 && len(authPassword) > 0 {
		authType = AUTH_BASIC
	}

	if len(options.Auth) > 0 && len(authUser) == 0 && len(authPassword) == 0 {
		fmt.Println("UNKNOWN - Username and password not given: provide -a|--auth username:password")
		os.Exit(EXIT_UNKNOWN)
	}

	r := &Request{
		Host:      options.Host,
		IPAddress: options.IPAddress,
		URI:       options.URI,
		Port:      port,
		Scheme:    scheme,
		Timeout:   options.Timeout,
		Authentication: Authentication{
			Type:     authType,
			User:     authUser,
			Password: authPassword,
		},
		SSLNoVerify:     options.SSLNoVerify,
		Verbose:         options.Verbose,
		FollowRedirects: options.FollowRedirects,
		WarningTimeout:  options.WarningTimeout,
		CriticalTimeout: options.CriticalTimeout,
		NoSNI:           options.NoSNI,
		ClientCert: ClientCert{
			ClientCertFile: options.ClientCertFile,
			PrivateKeyFile: options.PrivateKeyFile,
		},
	}

	if options.GuessAuth {
		authType = DetectAuthType(r)
		if r.Verbose {
			fmt.Println(fmt.Sprintf(">> Detected auth: %s", authLookup[authType]))
		}
		r.Authentication.Type = authType
	}

	var statusCodes []int
	if strings.Contains(options.ExpectedCode, ",") {
		for _, code := range strings.Split(options.ExpectedCode, ",") {
			codeInt, _ := strconv.Atoi(code)
			statusCodes = append(statusCodes, codeInt)
		}
	} else {
		codeInt, _ := strconv.Atoi(options.ExpectedCode)
		statusCodes = append(statusCodes, codeInt)
	}

	var SSLWarning int
	var SSLCritical int
	if strings.Contains(options.SSLExpiration, ",") {
		SSLParts := strings.Split(options.SSLExpiration, ",")
		if len(SSLParts) != 2 {
			fmt.Println("UNKNOWN - SSL check has invalid parameters: provide e.g. -C 14,7")
			os.Exit(EXIT_UNKNOWN)
		}
		SSLWarning, _ = strconv.Atoi(SSLParts[0])
		SSLCritical, _ = strconv.Atoi(SSLParts[1])
	} else {
		SSLWarning, _ = strconv.Atoi(options.SSLExpiration)
		SSLCritical = 0
	}

	e := &Expected{
		StatusCodes: statusCodes,
		BodyText:    options.BodyText,
		SSLCheck: SSLCheck{
			Run:          options.SSL,
			DaysWarning:  SSLWarning,
			DaysCritical: SSLCritical,
		},
	}

	msg, code, err := Check(r, e)

	if err != nil {
		fmt.Println(fmt.Sprintf("UNKNOWN, %s", err.Error()))
		os.Exit(EXIT_UNKNOWN)
	}

	fmt.Println(msg)
	os.Exit(code)
}
