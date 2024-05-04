package sip

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type uriFSM func(uri *Uri, s string) (uriFSM, string, error)

// ParseUri converts a string representation of a URI into a Uri object.
// Following https://datatracker.ietf.org/doc/html/rfc3261#section-19.1.1
// sip:user:password@host:port;uri-parameters?headers
func ParseUri(uriStr string, uri *Uri) (err error) {
	if len(uriStr) == 0 {
		return errors.New("Empty URI")
	}
	state := uriStateSIP
	str := uriStr
	for state != nil {
		state, str, err = state(uri, str)
		if err != nil {
			return
		}
	}
	return
}

func uriStateSIP(uri *Uri, s string) (uriFSM, string, error) {
	if len(s) >= 4 && strings.EqualFold(s[:4], "sip:") {
		return uriStateScheme, s[4:], nil
	}

	if len(s) >= 5 && strings.EqualFold(s[:5], "sips:") {
		uri.Encrypted = true
		return uriStateScheme, s[5:], nil
	}

	if s == "*" {
		// Normally this goes under url path, but we set on host
		uri.Host = "*"
		uri.Wildcard = true
		return nil, "", nil
	}

	// return nil, "", errors.New("missing protocol scheme")
	return uriStateUser, s, nil
}

func uriStateScheme(_ *Uri, s string) (uriFSM, string, error) {
	// Check does uri contain slashes
	// They are valid in uri but normally we cut them
	s, _ = strings.CutPrefix(s, "//")
	return uriStateUser, s, nil
}

func uriStateUser(uri *Uri, s string) (uriFSM, string, error) {
	var userend int = 0
	for i, c := range s {
		if c == ':' {
			userend = i
		}

		if c == '@' {
			if userend > 0 {
				uri.User = s[:userend]
				uri.Password = s[userend+1 : i]
			} else {
				uri.User = s[:i]
			}
			return uriStateHost, s[i+1:], nil
		}
	}

	return uriStateHost, s, nil
}

func uriStatePassword(uri *Uri, s string) (uriFSM, string, error) {
	for i, c := range s {
		if c == '@' {
			uri.Password = s[:i]
			return uriStateHost, s[i+1:], nil
		}
	}

	return nil, "", fmt.Errorf("missing @")
}

func uriStateHost(uri *Uri, s string) (uriFSM, string, error) {
	for i, c := range s {
		if c == ':' {
			uri.Host = s[:i]
			return uriStatePort, s[i+1:], nil
		}

		if c == ';' {
			uri.Host = s[:i]
			return uriStateUriParams, s[i+1:], nil
		}

		if c == '?' {
			uri.Host = s[:i]
			return uriStateHeaders, s[i+1:], nil
		}
	}
	// If no special chars found, it means we are at end
	uri.Host = s
	// Check is this wildcard
	uri.Wildcard = s == "*"
	return uriStateUriParams, "", nil
}

func uriStatePort(uri *Uri, s string) (uriFSM, string, error) {
	var err error
	for i, c := range s {
		if c == ';' {
			uri.Port, err = strconv.Atoi(s[:i])
			return uriStateUriParams, s[i+1:], err
		}

		if c == '?' {
			uri.Port, err = strconv.Atoi(s[:i])
			return uriStateHeaders, s[i+1:], err
		}
	}

	uri.Port, err = strconv.Atoi(s)
	return nil, s, err
}

func uriStateUriParams(uri *Uri, s string) (uriFSM, string, error) {
	var n int
	var err error
	if len(s) == 0 {
		uri.UriParams = NewParams()
		uri.Headers = NewParams()
		return nil, s, nil
	}
	uri.UriParams = NewParams()
	// uri.UriParams, n, err = ParseParams(s, 0, ';', '?', true, true)
	n, err = UnmarshalParams(s, ';', '?', uri.UriParams)
	if err != nil {
		return nil, s, err
	}

	if n == len(s) {
		n = n - 1
	}

	if s[n] != '?' {
		return nil, s, nil
	}

	return uriStateHeaders, s[n+1:], nil
}

func uriStateHeaders(uri *Uri, s string) (uriFSM, string, error) {
	var err error
	// uri.Headers, _, err = ParseParams(s, 0, '&', 0, true, false)
	uri.Headers = NewParams()
	_, err = UnmarshalParams(s, '&', 0, uri.Headers)
	return nil, s, err
}
