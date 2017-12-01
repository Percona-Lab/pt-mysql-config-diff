package confreader

import (
	"testing"

	tu "github.com/Percona-Lab/pt-mysql-config-diff/testutils"
)

func TestDefaultsReader(t *testing.T) {
	cnf, err := NewDefaultsParser("testdata/defaults.txt")
	tu.IsNil(t, err)
	var want *Config
	tu.LoadJson(t, "want_defaults.json", &want)
	tu.Equals(t, cnf, want)
}
