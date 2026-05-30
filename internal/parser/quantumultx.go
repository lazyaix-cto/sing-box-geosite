package parser

import (
	"bufio"
	"bytes"
	"strings"

	"github.com/lazyaix-cto/sing-box-geosite/internal/model"
)

func init() { register(quantumultX{}) }

// quantumultX parses blackmatrix7-style QuantumultX / Clash-classical rule
// lists: one `TYPE,value[,policy|options]` per line, `#` or `;` for comments.
type quantumultX struct{}

func (quantumultX) Name() string { return "quantumultx" }

func (quantumultX) Parse(content []byte) (*Result, error) {
	rs := &model.RuleSet{}
	skipped := map[string]int{}

	sc := bufio.NewScanner(bytes.NewReader(content))
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			continue
		}
		typ := strings.ToUpper(strings.TrimSpace(fields[0]))
		val := strings.TrimSpace(fields[1])
		if val == "" {
			continue
		}
		if !applyClassical(rs, typ, val) {
			skipped[typ]++
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return &Result{RuleSet: rs, Skipped: skipped}, nil
}
