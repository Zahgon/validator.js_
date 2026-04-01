// Copyright (c) 2018 Chris O'Hara <cohara87@gmail.com>
//
// Ported from validator.js, which is distributed under the MIT licence; see LICENSE.

package validator

import (
	"github.com/validatorjs/validator/jsre"
	"github.com/validatorjs/validator/jsstr"
	"github.com/validatorjs/validator/jsval"
)

var fqdnDefaults = jsval.ObjectOf(
	jsval.Of("require_tld"), true,
	jsval.Of("allow_underscores"), false,
	jsval.Of("allow_trailing_dot"), false,
	jsval.Of("allow_numeric_tld"), false,
	jsval.Of("allow_wildcard"), false,
	jsval.Of("ignore_max_length"), false,
)

var (
	reFQDNTLD  = re("^([a-z\\u00A1-\\u00A8\\u00AA-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFEF]{2,}|xn[a-z0-9-]{2,})$", "i")
	reFQDNWS   = re("\\s", "")
	reDigits   = re("^\\d+$", "")
	reFQDNPart = re("^[a-z_\\u00a1-\\uffff0-9-]+$", "i")
	reFullWide = re("[\\uff01-\\uff5e]", "")
	reEdgeDash = re("^-|-$", "")
	reUnderbar = re("_", "")
)

// IsFQDN reports whether the string is a fully qualified domain name.
func IsFQDN(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, fqdnDefaults)
	if jsval.Truthy(opts.Get("allow_trailing_dot")) && len(s) > 0 && s[len(s)-1] == '.' {
		s = s.Substring(0, len(s)-1)
	}
	if jsval.StrictEquals(opts.Get("allow_wildcard"), true) && s.Index("*.") == 0 {
		s = s.Substring(2, len(s))
	}
	parts := s.Split(".")
	tld := parts[len(parts)-1]
	if jsval.Truthy(opts.Get("require_tld")) {
		if len(parts) < 2 {
			return false
		}
		if !jsval.Truthy(opts.Get("allow_numeric_tld")) && !reFQDNTLD.Test(tld) {
			return false
		}
		if reFQDNWS.Test(tld) {
			return false
		}
	}
	if !jsval.Truthy(opts.Get("allow_numeric_tld")) && reDigits.Test(tld) {
		return false
	}
	for _, part := range parts {
		if len(part) > 63 && !jsval.Truthy(opts.Get("ignore_max_length")) {
			return false
		}
		if !reFQDNPart.Test(part) {
			return false
		}
		if reFullWide.Test(part) {
			return false
		}
		if reEdgeDash.Test(part) {
			return false
		}
		if !jsval.Truthy(opts.Get("allow_underscores")) && reUnderbar.Test(part) {
			return false
		}
	}
	return true
}

const (
	ipv4Segment = "(?:[0-9]|[1-9][0-9]|1[0-9][0-9]|2[0-4][0-9]|25[0-5])"
	ipv4Address = "(" + ipv4Segment + "[.]){3}" + ipv4Segment
	ipv6Segment = "(?:[0-9a-fA-F]{1,4})"
)

var (
	reIPv4 = re("^"+ipv4Address+"$", "")
	reIPv6 = re("^("+
		"(?:"+ipv6Segment+":){7}(?:"+ipv6Segment+"|:)|"+
		"(?:"+ipv6Segment+":){6}(?:"+ipv4Address+"|:"+ipv6Segment+"|:)|"+
		"(?:"+ipv6Segment+":){5}(?::"+ipv4Address+"|(:"+ipv6Segment+"){1,2}|:)|"+
		"(?:"+ipv6Segment+":){4}(?:(:"+ipv6Segment+"){0,1}:"+ipv4Address+"|(:"+ipv6Segment+"){1,3}|:)|"+
		"(?:"+ipv6Segment+":){3}(?:(:"+ipv6Segment+"){0,2}:"+ipv4Address+"|(:"+ipv6Segment+"){1,4}|:)|"+
		"(?:"+ipv6Segment+":){2}(?:(:"+ipv6Segment+"){0,3}:"+ipv4Address+"|(:"+ipv6Segment+"){1,5}|:)|"+
		"(?:"+ipv6Segment+":){1}(?:(:"+ipv6Segment+"){0,4}:"+ipv4Address+"|(:"+ipv6Segment+"){1,6}|:)|"+
		"(?::((?::"+ipv6Segment+"){0,5}:"+ipv4Address+"|(?::"+ipv6Segment+"){1,7}|:))"+
		")(%[0-9a-zA-Z.]{1,})?$", "")
)

// IsIP reports whether the string is an internet address of the given version, of either version when none is
// given. The version may be given as the options or, for the sake of older callers, as the second argument.
func IsIP(args []jsval.Value) jsval.Value {
	input := arg(args, 0)
	s := str(input)
	options := arg(args, 1)
	if jsval.IsUndefined(options) {
		options = jsval.NewObject()
	}
	var version jsval.Value
	if jsval.TypeOf(options) == "object" {
		version = prop(options, "version")
	} else {
		version = arg(args, 1)
	}
	version = or(version, jsval.Of(""))
	if !jsval.Truthy(version) {
		return jsval.Truthy(IsIP([]jsval.Value{input, jsval.ObjectOf(jsval.Of("version"), float64(4))})) ||
			jsval.Truthy(IsIP([]jsval.Value{input, jsval.ObjectOf(jsval.Of("version"), float64(6))}))
	}
	switch jsval.ToString(version).Go() {
	case "4":
		return reIPv4.Test(s)
	case "6":
		return reIPv6.Test(s)
	}
	return false
}

var reSubnetMaybe = re("^\\d{1,3}$", "")

// IsIPRange reports whether the string is an internet address with a prefix length.
func IsIPRange(input, version jsval.Value) jsval.Value {
	s := str(input)
	if jsval.IsUndefined(version) {
		version = jsval.Of("")
	}
	parts := s.Split("/")
	if len(parts) != 2 {
		return false
	}
	if !reSubnetMaybe.Test(parts[1]) {
		return false
	}
	if len(parts[1]) > 1 && parts[1].HasPrefix("0") {
		return false
	}
	if !jsval.Truthy(IsIP([]jsval.Value{parts[0], version})) {
		return false
	}
	var expected float64
	switch jsval.ToString(version).Go() {
	case "4":
		expected = 32
	case "6":
		expected = 128
	default:
		expected = 32
		if jsval.Truthy(IsIP([]jsval.Value{parts[0], jsval.Of("6")})) {
			expected = 128
		}
	}
	return jsval.Compare(parts[1], expected, "<=") && jsval.Compare(parts[1], float64(0), ">=")
}

var emailDefaults = jsval.ObjectOf(
	jsval.Of("allow_display_name"), false,
	jsval.Of("allow_underscores"), false,
	jsval.Of("require_display_name"), false,
	jsval.Of("allow_utf8_local_part"), true,
	jsval.Of("require_tld"), true,
	jsval.Of("blacklisted_chars"), jsval.Of(""),
	jsval.Of("ignore_max_length"), false,
	jsval.Of("host_blacklist"), []jsval.Value{},
	jsval.Of("host_whitelist"), []jsval.Value{},
)

var (
	reSplitNameAddress = re("^([^\\x00-\\x1F\\x7F-\\x9F\\cX]+)<", "i")
	reEmailUserPart    = re("^[a-z\\d!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]+$", "i")
	reGmailUserPart    = re("^[a-z\\d]+$", "")
	reQuotedEmailUser  = re("^([\\s\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f\\x21\\x23-\\x5b\\x5d-\\x7e]|"+
		"(\\\\[\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]))*$", "i")
	reEmailUserUtf8Part = re("^[a-z\\d!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~"+
		"\\u00A1-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFEF]+$", "i")
	reQuotedEmailUserUtf8 = re("^([\\s\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f\\x21\\x23-\\x5b\\x5d-\\x7e"+
		"\\u00A0-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFEF]|"+
		"(\\\\[\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f\\u00A0-\\uD7FF\\uF900-\\uFDCF\\uFDF0-\\uFFEF]))*$", "i")
	reQuotedDisplayName = re("^\"(.+)\"$", "")
	reIllegalInName     = re("[\\.\";<>]", "")
	reAngleBrackets     = re("(^<|>$)", "g")
	reDots              = re("\\.", "g")
)

const defaultMaxEmailLength = 254

// validateDisplayName checks the name in front of an address: it may hold the characters that need quoting only
// when it is quoted, and the quotes inside it must be escaped.
func validateDisplayName(name jsstr.String) bool {
	withoutQuotes := jsre.Replace(name, reQuotedDisplayName, jsstr.Of("$1"))
	if len(withoutQuotes.Trim()) == 0 {
		return false
	}
	if reIllegalInName.Test(withoutQuotes) {
		if withoutQuotes.Equal(name) {
			return false
		}
		if len(withoutQuotes.Split("\"")) != len(withoutQuotes.Split("\\\"")) {
			return false
		}
	}
	return true
}

// checkHost reports whether the host is one of the hosts, each of which is either the host itself or a pattern
// that matches it.
func checkHost(host jsstr.String, matches jsval.Value) bool {
	list, ok := jsval.Array(matches)
	if !ok {
		return false
	}
	for _, match := range list {
		if text, ok := match.(jsstr.String); ok && host.Equal(text) {
			return true
		}
		if pattern, ok := match.(*jsre.Regexp); ok && pattern.Test(host) {
			return true
		}
	}
	return false
}

// IsEmail reports whether the string is an email address, with the parts and the hosts the options allow.
func IsEmail(input, options jsval.Value) jsval.Value {
	s := str(input)
	opts := jsval.Merge(options, emailDefaults)
	if jsval.Truthy(opts.Get("require_display_name")) || jsval.Truthy(opts.Get("allow_display_name")) {
		_, m := jsre.Matches(s, reSplitNameAddress)
		if m != nil {
			name, _ := m.Group(1)
			s = jsre.Replace(s.ReplaceText(name, jsstr.String{}), reAngleBrackets, jsstr.String{})
			if name.HasSuffix(" ") {
				name = name.Slice(0, -1)
			}
			if !validateDisplayName(name) {
				return false
			}
		} else if jsval.Truthy(opts.Get("require_display_name")) {
			return false
		}
	}
	if !jsval.Truthy(opts.Get("ignore_max_length")) && len(s) > defaultMaxEmailLength {
		return false
	}
	parts := s.Split("@")
	domain := parts[len(parts)-1]
	parts = parts[:len(parts)-1]
	lowerDomain := domain.ToLower()
	if blacklist, ok := jsval.Array(opts.Get("host_blacklist")); ok && len(blacklist) > 0 &&
		checkHost(lowerDomain, opts.Get("host_blacklist")) {
		return false
	}
	if whitelist, ok := jsval.Array(opts.Get("host_whitelist")); ok && len(whitelist) > 0 &&
		!checkHost(lowerDomain, opts.Get("host_whitelist")) {
		return false
	}
	user := jsstr.Join(parts, "@")
	if jsval.Truthy(opts.Get("domain_specific_validation")) &&
		(lowerDomain.Is("gmail.com") || lowerDomain.Is("googlemail.com")) {
		user = user.ToLower()
		username := user.Split("+")[0]
		withoutDots := jsre.Replace(username, reDots, jsstr.String{})
		if !jsval.Truthy(IsByteLength([]jsval.Value{withoutDots,
			jsval.ObjectOf(jsval.Of("min"), float64(6), jsval.Of("max"), float64(30))})) {
			return false
		}
		for _, part := range username.Split(".") {
			if !reGmailUserPart.Test(part) {
				return false
			}
		}
	}
	if jsval.StrictEquals(opts.Get("ignore_max_length"), false) &&
		(!jsval.Truthy(IsByteLength([]jsval.Value{user, jsval.ObjectOf(jsval.Of("max"), float64(64))})) ||
			!jsval.Truthy(IsByteLength([]jsval.Value{domain, jsval.ObjectOf(jsval.Of("max"), float64(254))}))) {
		return false
	}
	if !jsval.Truthy(IsFQDN(domain, jsval.ObjectOf(
		jsval.Of("require_tld"), opts.Get("require_tld"),
		jsval.Of("ignore_max_length"), opts.Get("ignore_max_length"),
		jsval.Of("allow_underscores"), opts.Get("allow_underscores"),
	))) {
		if !jsval.Truthy(opts.Get("allow_ip_domain")) {
			return false
		}
		if !jsval.Truthy(IsIP([]jsval.Value{domain})) {
			if !domain.HasPrefix("[") || !domain.HasSuffix("]") {
				return false
			}
			noBracket := domain.Slice(1, -1)
			if len(noBracket) == 0 || !jsval.Truthy(IsIP([]jsval.Value{noBracket})) {
				return false
			}
		}
	}
	if jsval.Truthy(opts.Get("blacklisted_chars")) {
		pattern := newRegExp(jsval.Of("["+jsval.ToString(opts.Get("blacklisted_chars")).Go()+"]+"), jsval.Of("g"))
		if jsre.Search(user, pattern) != -1 {
			return false
		}
	}
	if len(user) > 0 && user[0] == '"' && user[len(user)-1] == '"' {
		user = user.Slice(1, len(user)-1)
		if jsval.Truthy(opts.Get("allow_utf8_local_part")) {
			return reQuotedEmailUserUtf8.Test(user)
		}
		return reQuotedEmailUser.Test(user)
	}
	pattern := reEmailUserPart
	if jsval.Truthy(opts.Get("allow_utf8_local_part")) {
		pattern = reEmailUserUtf8Part
	}
	for _, part := range user.Split(".") {
		if !pattern.Test(part) {
			return false
		}
	}
	return true
}

// IsMailtoURI reports whether the string is a mailto link whose addresses are all valid.
func IsMailtoURI(input, options jsval.Value) jsval.Value {
	s := str(input)
	if s.Index("mailto:") != 0 {
		return false
	}
	rest := s.ReplaceText(jsstr.Of("mailto:"), jsstr.String{}).Split("?")
	to := rest[0]
	queryString := jsstr.String{}
	if len(rest) > 1 {
		queryString = rest[1]
	}
	if len(to) == 0 && len(queryString) == 0 {
		return true
	}
	cc, bcc, ok := parseMailtoQueryString(queryString)
	if !ok {
		return false
	}
	all := jsstr.Concat(to, jsstr.Of(","), cc, jsstr.Of(","), bcc)
	for _, email := range all.Split(",") {
		trimmed := jsval.Str(Trim(email, jsval.Of(" ")))
		if len(trimmed) > 0 && !jsval.Truthy(IsEmail(trimmed, options)) {
			return false
		}
	}
	return true
}

// parseMailtoQueryString reads the query of a mailto link, which may name a subject, a body and copied
// addresses, each of them at most once.
func parseMailtoQueryString(queryString jsstr.String) (cc, bcc jsstr.String, ok bool) {
	allowed := map[string]bool{"subject": true, "body": true, "cc": true, "bcc": true}
	cc, bcc = jsstr.String{}, jsstr.String{}
	params := queryString.Split("&")
	if len(params) > 4 {
		return nil, nil, false
	}
	for _, q := range params {
		pair := q.Split("=")
		key := pair[0]
		value := jsstr.String{}
		if len(pair) > 1 {
			value = pair[1]
		}
		if len(key) > 0 && !allowed[key.Go()] {
			return nil, nil, false
		}
		if len(value) > 0 && (key.Is("cc") || key.Is("bcc")) {
			if key.Is("cc") {
				cc = value
			} else {
				bcc = value
			}
		}
		if len(key) > 0 {
			delete(allowed, key.Go())
		}
	}
	return cc, bcc, true
}
