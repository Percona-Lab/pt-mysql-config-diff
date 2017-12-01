package confreader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/pkg/errors"
)

// How to get defaults:
// touch /tmp/my.cnf
// mysqld --defaults-file=/tmp/my.cnf --verbose --help > /tmp/defaultvals

func NewDefaultsParser(filename string) (ConfigReader, error) {
	f, err := os.Open(cleanFilename(filename))
	if err != nil {
		return nil, errors.Wrap(err, "cannot read defaults file")
	}
	defer f.Close()

	return parseFile(f)
}

func parseFile(r io.Reader) (ConfigReader, error) {
	cnf := &Config{ConfigType: "defaults", EntriesMap: make(map[string]interface{})}
	s := bufio.NewScanner(r)

	inHeader := true
	for s.Scan() {
		t := s.Text()
		if inHeader {
			if strings.HasPrefix(t, "-----") {
				inHeader = false
			}
			continue
		}
		if strings.TrimSpace(t) == "" {
			break
		}

		parts := strings.SplitN(t, " ", 2)
		key := strings.Replace(strings.TrimSpace(parts[0]), "-", "_", -1)
		val := ""
		if len(parts) == 2 {
			val = strings.TrimSpace(parts[1])
		}
		if val == "(No default value)" {
			val = ""
		}
		cnf.EntriesMap[key] = val
	}

	if len(cnf.EntriesMap) == 0 {
		return nil, fmt.Errorf("Invalid defaults file. There are no entries to parse")
	}

	return cnf, nil
}
