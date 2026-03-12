package cmd_test

import (
	"bytes"
	"testing"

	"github.com/mattiasthalen/qlik-parser/cmd"
)

func TestRootHelp(t *testing.T) {
	buf := &bytes.Buffer{}
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"--help"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("qlik-parser")) {
		t.Errorf("expected help output to contain binary name, got: %s", buf.String())
	}
}
