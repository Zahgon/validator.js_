// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

var urlDefaults = jsval.ObjectOf(
	jsval.Of("protocols"), []jsval.Value{jsval.Of("http"), jsval.Of("https"), jsval.Of("ftp")},
	jsval.Of("require_tld"), true,
	jsval.Of("require_protocol"), false,
	jsval.Of("require_host"), true,
	jsval.Of("require_port"), false,
	jsval.Of("require_valid_protocol"), true,
	jsval.Of("allow_underscores"), false,
	jsval.Of("allow_trailing_dot"), false,
	jsval.Of("allow_protocol_relative_urls"), false,
	jsval.Of("allow_fragments"), true,
	jsval.Of("allow_query_components"), true,
	jsval.Of("validate_length"), true,
	jsval.Of("max_allowed_length"), float64(2084),
)

var (
	reWrappedIPv6   = re("^\\[([^\\]]+)\\](?::([0-9]+))?$", "")
	reURLBadChars   = re("[\\s<>]", "")
	reURLProtocol   = re("^([a-z][a-z0-9+\\-.]*):", "i")
	reValidAuth     = re("^[a-zA-Z0-9\\-_.%:]*$", "")
	reEncodedInAuth = re("%[0-9a-fA-F]{2}", "")
	reLeadingDigit  = re("^[0-9]", "")
	reAllDigits     = re("^[0-9]+$", "")
)

// IsURL reports whether the string is a web address, with the protocols, the hosts, the port and the parts the
// options allow.
func IsURL(input, options jsval.Value) jsval.Value {
	url := str(input)
	if len(url) == 0 || reURLBadChars.Test(url) {
		return false
	}
	if url.Index("mailto:") == 0 {
		return false
	}
	opts := jsval.Merge(options, urlDefaults)
	if jsval.Truthy(opts.Get("validate_length")) &&
		jsval.Compare(float64(len(url)), opts.Get("max_allowed_length"), ">") {
		return false
	}
	if !jsval.Truthy(opts.Get("allow_fragments")) && url.Contains("#") {
		return false
	}
	if !jsval.Truthy(opts.Get("allow_query_components")) && (url.Contains("?") || url.Contains("&")) {
		return false
	}
	url = url.Split("#")[0]
	url = url.Split("?")[0]

	_, protocolMatch := jsre.Matches(url, reURLProtocol)
	hadExplicitProtocol := false
	var protocol jsstr.String
	// cleanUpProtocol takes the protocol off the front, answering false when it is not one the options allow.
	cleanUpProtocol := func(potential jsstr.String) (jsstr.String, bool) {
		hadExplicitProtocol = true
		protocol = potential.ToLower()
		if jsval.Truthy(opts.Get("require_valid_protocol")) && indexOfValue(opts.Get("protocols"), protocol) == -1 {
			return nil, false
		}
		whole, _ := protocolMatch.Group(0)
		return url.Substring(len(whole), len(url)), true
	}
	if protocolMatch != nil {
		potential, _ := protocolMatch.Group(1)
		whole, _ := protocolMatch.Group(0)
		afterColon := url.Substring(len(whole), len(url))
		if !afterColon.Slice(0, 2).Is("//") {
			firstSlash := afterColon.Index("/")
			beforeSlash := afterColon
			if firstSlash != -1 {
				beforeSlash = afterColon.Substring(0, firstSlash)
			}
			at := beforeSlash.Index("@")
			if at != -1 {
				beforeAt := beforeSlash.Substring(0, at)
				if reValidAuth.Test(beforeAt) && !reEncodedInAuth.Test(beforeAt) {
					if jsval.Truthy(opts.Get("require_protocol")) {
						return false
					}
				} else {
					next, ok := cleanUpProtocol(potential)
					if !ok {
						return false
					}
					url = next
				}
			} else if reLeadingDigit.Test(afterColon) {
				if jsval.Truthy(opts.Get("require_protocol")) {
					return false
				}
			} else {
				next, ok := cleanUpProtocol(potential)
				if !ok {
					return false
				}
				url = next
			}
		} else {
			next, ok := cleanUpProtocol(potential)
			if !ok {
				return false
			}
			url = next
		}
	} else if jsval.Truthy(opts.Get("require_protocol")) {
		return false
	}
	if url.Slice(0, 2).Is("//") {
		if !hadExplicitProtocol && !jsval.Truthy(opts.Get("allow_protocol_relative_urls")) {
			return false
		}
		url = url.From(2)
	}
	if len(url) == 0 {
		return false
	}
	split := url.Split("/")
	url = split[0]
	if len(url) == 0 && !jsval.Truthy(opts.Get("require_host")) {
		return true
	}
	split = url.Split("@")
	if len(split) > 1 {
		if jsval.Truthy(opts.Get("disallow_auth")) {
			return false
		}
		if len(split[0]) == 0 {
			return false
		}
		auth := split[0]
		split = split[1:]
		if auth.Index(":") >= 0 && len(auth.Split(":")) > 2 {
			return false
		}
		credentials := auth.Split(":")
		user := credentials[0]
		password := jsstr.String(nil)
		if len(credentials) > 1 {
			password = credentials[1]
		}
		if len(user) == 0 && password != nil && len(password) == 0 {
			return false
		}
	}
	hostname := jsstr.Join(split, "@")
	var portStr jsstr.String
	havePort := false
	var ipv6 jsstr.String
	haveIPv6 := false
	var host jsstr.String
	if _, m := jsre.Matches(hostname, reWrappedIPv6); m != nil {
		host = jsstr.String{}
		ipv6, haveIPv6 = m.Group(1)
		if text, ok := m.Group(2); ok && len(text) > 0 {
			portStr, havePort = text, true
		}
	} else {
		parts := hostname.Split(":")
		host = parts[0]
		if len(parts) > 1 {
			portStr, havePort = jsstr.Join(parts[1:], ":"), true
		}
	}
	if havePort && len(portStr) > 0 {
		port := jsstr.ParseInt(portStr, 10)
		if !reAllDigits.Test(portStr) || port <= 0 || port > 65535 {
			return false
		}
	} else if jsval.Truthy(opts.Get("require_port")) {
		return false
	}
	if jsval.Truthy(opts.Get("host_whitelist")) {
		return checkHost(host, opts.Get("host_whitelist"))
	}
	if len(host) == 0 && !jsval.Truthy(opts.Get("require_host")) {
		return true
	}
	if !jsval.Truthy(IsIP([]jsval.Value{host})) && !jsval.Truthy(IsFQDN(host, opts)) &&
		(!haveIPv6 || !jsval.Truthy(IsIP([]jsval.Value{ipv6, float64(6)}))) {
		return false
	}
	if len(host) == 0 && haveIPv6 {
		host = ipv6
	}
	if jsval.Truthy(opts.Get("host_blacklist")) && checkHost(host, opts.Get("host_blacklist")) {
		return false
	}
	return true
}
