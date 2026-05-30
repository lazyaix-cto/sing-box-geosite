package parser

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/lazyaix/sing-box-geosite/internal/model"
)

func init() { register(adGuard{}) }

// adGuard parses the DNS-filtering subset of AdGuard/uBlock syntax:
//   - `||domain^[$modifiers]` -> domain_suffix (or ip_cidr for an IP)
//   - `@@...` exceptions, cosmetic (`##` etc.), `*` wildcards and URL/path
//     patterns are counted as skipped (not representable in a headless rule set)
//   - a bare domain line -> domain_suffix
type adGuard struct{}

func (adGuard) Name() string { return "adguard" }

func (adGuard) Parse(content []byte) (*Result, error) {
	rs := &model.RuleSet{}
	skipped := map[string]int{}

	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "!") {
			continue
		}
		switch {
		case isCosmetic(line):
			// Element-hiding / scriptlet rules — not representable; checked
			// before the '#' comment skip since cosmetic rules start with '#'.
			skipped["cosmetic"]++
		case strings.HasPrefix(line, "#"):
			continue // a '#' comment that isn't a cosmetic rule
		case strings.HasPrefix(line, "@@"):
			skipped["exception"]++
		case strings.HasPrefix(line, "||"):
			addAdGuardToken(rs, skipped, cutAtAny(line[2:], "^$/|"))
		case strings.HasPrefix(line, "|"), strings.ContainsAny(line, "*/$"):
			skipped["pattern"]++
		default:
			addAdGuardToken(rs, skipped, line)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return &Result{RuleSet: rs, Skipped: skipped}, nil
}

func addAdGuardToken(rs *model.RuleSet, skipped map[string]int, token string) {
	token = strings.TrimSpace(token)
	switch {
	case token == "":
		skipped["empty"]++
	case strings.ContainsAny(token, "*"):
		skipped["wildcard"]++
	default:
		if cidr, ok := toCIDR(token); ok {
			rs.IPCIDR = append(rs.IPCIDR, cidr)
		} else if looksLikeDomain(token) {
			rs.DomainSuffix = append(rs.DomainSuffix, token)
		} else {
			skipped["other"]++
		}
	}
}

func isCosmetic(line string) bool {
	return strings.Contains(line, "##") || strings.Contains(line, "#@#") ||
		strings.Contains(line, "#%#") || strings.Contains(line, "#$#") ||
		strings.Contains(line, "#?#")
}

func cutAtAny(s, chars string) string {
	if i := strings.IndexAny(s, chars); i >= 0 {
		return s[:i]
	}
	return s
}
