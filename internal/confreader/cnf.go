package confreader

import (
	"fmt"

	ini "gopkg.in/ini.v1"
)

func NewCNFReader(filename string) (ConfigReader, error) {
	filename = cleanFilename(filename)
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, filename)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("Invalid file: %s", filename)
	}

	cnf := &Config{ConfigType: "cnf", EntriesMap: make(map[string]interface{})}

	for _, key := range cfg.Section("mysqld").Keys() {
		cnf.EntriesMap[key.Name()] = key.Value()
	}

	return cnf, nil
}
