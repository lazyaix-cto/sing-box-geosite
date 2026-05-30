package parser

import (
	"strconv"
	"strings"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

// applyClassical maps a single classical-style rule (`TYPE,value`) into the IR.
// Shared by the QuantumultX parser and the Clash classical-behavior path.
// Returns false when the type is unsupported so the caller can record it in
// Skipped instead of dropping it silently.
func applyClassical(rs *model.RuleSet, typ, val string) bool {
	switch strings.ToUpper(typ) {
	case "HOST", "DOMAIN":
		rs.Domain = append(rs.Domain, val)
	case "HOST-SUFFIX", "DOMAIN-SUFFIX":
		rs.DomainSuffix = append(rs.DomainSuffix, val)
	case "HOST-KEYWORD", "DOMAIN-KEYWORD":
		rs.DomainKeyword = append(rs.DomainKeyword, val)
	case "DOMAIN-REGEX":
		rs.DomainRegex = append(rs.DomainRegex, val)
	case "IP-CIDR", "IP-CIDR6", "IP6-CIDR":
		rs.IPCIDR = append(rs.IPCIDR, val)
	case "SRC-IP-CIDR", "SOURCE-IP-CIDR":
		rs.SourceIPCIDR = append(rs.SourceIPCIDR, val)
	case "DST-PORT", "PORT":
		p, err := strconv.ParseUint(val, 10, 16)
		if err != nil {
			return false
		}
		rs.Port = append(rs.Port, uint16(p))
	default:
		// IP-ASN, GEOIP, USER-AGENT, URL-REGEX, AND/OR, PROCESS-NAME, ...
		return false
	}
	return true
}

// classicalTypes is the set of rule-type tokens that mark a line as
// classical-format (QuantumultX / Clash classical), used by Detect. It
// intentionally includes unsupported types — their presence still identifies
// the format.
var classicalTypes = map[string]bool{
	"HOST": true, "DOMAIN": true,
	"HOST-SUFFIX": true, "DOMAIN-SUFFIX": true,
	"HOST-KEYWORD": true, "DOMAIN-KEYWORD": true,
	"DOMAIN-REGEX": true, "URL-REGEX": true,
	"IP-CIDR": true, "IP-CIDR6": true, "IP6-CIDR": true,
	"SRC-IP-CIDR": true, "SOURCE-IP-CIDR": true,
	"GEOIP": true, "IP-ASN": true,
	"DST-PORT": true, "SRC-PORT": true, "PORT": true,
	"USER-AGENT": true, "PROCESS-NAME": true,
	"AND": true, "OR": true, "NOT": true,
}
