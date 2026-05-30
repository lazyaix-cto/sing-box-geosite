package compile

import (
	"bytes"
	"fmt"
	"os"

	"github.com/sagernet/sing-box/common/srs"
	"github.com/sagernet/sing-box/option"
	"github.com/sagernet/sing/common/json"
)

// SRS compiles source-format JSON bytes into a binary .srs file at path, using
// sing-box's own writer in-process — no external sing-box binary and no
// shell-out (the key improvement over the original project, which installed a
// pinned sing-box .deb and called `os.system`).
//
// generateVersion selects the binary format version (1/2/3); 0 falls back to
// the version declared in the JSON. Errors from parsing (bad regex/CIDR) or
// writing surface here instead of being lost in a shell exit code.
func SRS(jsonBytes []byte, path string, generateVersion uint8) error {
	compat, err := json.UnmarshalExtended[option.PlainRuleSetCompat](jsonBytes)
	if err != nil {
		return fmt.Errorf("parse source json: %w", err)
	}
	version := generateVersion
	if version == 0 {
		version = compat.Version
	}

	var buf bytes.Buffer
	if err := srs.Write(&buf, compat.Options, version); err != nil {
		return fmt.Errorf("write srs: %w", err)
	}
	if err := verify(buf.Bytes()); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// verify round-trips the binary back through srs.Read — our in-process proof
// that a real sing-box client can load the file, without shipping the CLI.
func verify(data []byte) error {
	if _, err := srs.Read(bytes.NewReader(data), false); err != nil {
		return fmt.Errorf("srs read-back verification failed: %w", err)
	}
	return nil
}
