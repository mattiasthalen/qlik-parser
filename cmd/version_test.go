package cmd_test

import (
	"bytes"
	"testing"

	"github.com/mattiasthalen/qlik-script-extractor/cmd"
)

func TestVersionCmd(t *testing.T) {
	buf := &bytes.Buffer{}
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("qlik-script-extractor")) {
		t.Errorf("version output missing binary name, got: %s", out)
	}
}
