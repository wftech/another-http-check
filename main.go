package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/jessevdk/go-flags"
)

type Options struct {
	Host          string `short:"H" description:"Host ex. google.com" default:""`
	IPAdress      string `short:"I" description:"IPv4 address ex. 8.8.4.4" default:""`
	URI           string `short:"u" long:"uri" description:"URI to check" default:"/"`
	Port          int    `short:"p" description:"Port ex. 80 for HTTP 443 for HTTPS" default:"80"`
	SSL           bool   `short:"S" long:"tls" description:"Use TLS"`
	Timeout       int    `short:"t" long:"timeout" description:"Timeout" default:"30"`
	AuthBasic     bool   `long:"auth-basic" description:"Use bacis auth"`
	AuthNtlm      bool   `long:"auth-ntlm" description:"Use NTLM auth"`
	Auth          string `short:"a" long:"authorisation" description:"ex. user:passwrod" default:""`
	ExpectedCode  string `short:"e" long:"expect" description:"Expected HTTP code" default:"200"`
	BodyText      string `short:"s" long:"string" description:"Search for given string in response body" default:""`
	SSLExpiration string `short:"C" description:"Check SSL cert expiration" default:""`
	Verbose       bool   `short:"v" long:"verbose" description:"Verbose mode"`
}

var options Options
var parser = flags.NewParser(&options, flags.Default)

func main() {
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	}

	var scheme string
	if options.Port == 443 {
		scheme = "https"
	} else {
		scheme = "http"
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
		if len(authParts) != 1 {
			// TODO
		}
		authUser = authParts[0]
		authPassword = authParts[1]
	}

	if authType == AUTH_NONE && len(authUser) > 0 && len(authPassword) > 0 {
		authType = AUTH_BASIC
	}

	r := &Request{
		Host:      options.Host,
		IPAddress: options.IPAdress,
		URI:       options.URI,
		Port:      options.Port,
		Scheme:    scheme,
		Timeout:   options.Timeout,
		Authentication: Authentication{
			Type:     authType,
			User:     authUser,
			Password: authPassword,
		},
		Verbose: options.Verbose,
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
			// TODO
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
