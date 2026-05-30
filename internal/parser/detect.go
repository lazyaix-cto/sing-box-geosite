package parser

import (
	"bufio"
	"bytes"
	"net/netip"
	"strings"
)

// Detect picks the most likely format for content fetched from url. Used when a
// source leaves format empty or "auto". The chosen format is logged by the
// generator so the decision is never opaque.
func Detect(content []byte, url string) string {
	lower := strings.ToLower(url)
	sample := leadingSample(content, 4096)
	trimmed := strings.TrimLeft(sample, " \t\r\n")

	switch {
	case strings.HasPrefix(trimmed, "{") && strings.Contains(sample, "\"rules\""):
		return "singbox"
	case strings.Contains(sample, "payload:"),
		strings.HasSuffix(lower, ".yaml"), strings.HasSuffix(lower, ".yml"):
		return "clash"
	case firstRuleLineIsClassical(content):
		return "quantumultx"
	case looksLikeAdGuard(sample):
		return "adguard"
	case looksLikeHosts(content):
		return "hosts"
	default:
		return "domainlist"
	}
}

func looksLikeAdGuard(sample string) bool {
	if strings.Contains(sample, "[Adblock") {
		return true
	}
	for _, line := range strings.Split(sample, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "||") || strings.HasPrefix(line, "@@||") {
			return true
		}
	}
	return false
}

func looksLikeHosts(content []byte) bool {
	line := firstMeaningfulLine(content)
	if strings.HasPrefix(line, "address=/") || strings.HasPrefix(line, "server=/") {
		return true
	}
	fields := strings.Fields(line)
	if len(fields) >= 2 {
		if _, err := netip.ParseAddr(fields[0]); err == nil {
			return true
		}
	}
	return false
}

func firstMeaningfulLine(content []byte) string {
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") ||
			strings.HasPrefix(line, "!") || strings.HasPrefix(line, ";") {
			continue
		}
		return line
	}
	return ""
}

func leadingSample(content []byte, n int) string {
	if len(content) > n {
		content = content[:n]
	}
	return string(content)
}

// firstRuleLineIsClassical reports whether the first meaningful line looks like
// `TYPE,value` with a known classical rule type.
func firstRuleLineIsClassical(content []byte) bool {
	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		idx := strings.IndexByte(line, ',')
		if idx < 0 {
			return false
		}
		return classicalTypes[strings.ToUpper(strings.TrimSpace(line[:idx]))]
	}
	return false
}
