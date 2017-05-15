package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	flag "github.com/spf13/pflag"
	ini "gopkg.in/ini.v1"
)

type options struct {
	CNFs        []string
	DSNs        dsnFlags
	OutputFmt   string
	Help        bool
	compareBase string // First CNF or first MySQL used as comparisson base
}

type dsnFlag struct {
	Host     string
	Port     int
	User     string
	Password string
	Database string
	Table    string
	protocol string
}

type dsnFlags []dsnFlag

func (d dsnFlags) String() string {
	parts := []string{}

	//if d.Host != "" {
	//	parts = append(parts, "h="+d.Host)
	//}
	//if d.Port != 0 {
	//	parts = append(parts, fmt.Sprintf("P=%d", d.Port))
	//}
	//if d.User != "" {
	//	parts = append(parts, "u="+d.User)
	//}
	//if d.Password != "" {
	//	parts = append(parts, "p="+d.Password)
	//}
	//if d.Database != "" {
	//	parts = append(parts, "D="+d.Database)
	//}
	//if d.Table != "" {
	//	parts = append(parts, "t="+d.Table)
	//}

	return strings.Join(parts, ",")
}

func (d dsnFlags) Set(value string) error {
	parts := strings.Split(value, ",")

	var dsn dsnFlag
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		key := string(part[0])
		value := string(part[2:])
		switch key {
		case "D":
			dsn.Database = value
		case "h":
			dsn.Host = value
		case "p":
			dsn.Password = value
		case "P":
			port, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				dsn.Port = int(port)
			}
		case "t":
			dsn.Table = value
		case "u":
			dsn.User = value
		}
	}

	if dsn.Host == "localhost" {
		dsn.protocol = "unix"
	} else {
		dsn.protocol = "tcp"
	}
	d = append(d, dsn)
	return nil
}

func (d dsnFlags) Type() string {
	return "dsn"
}

func main() {
	opts, err := processParams(os.Args[1:])
	if err != nil {
		os.Exit(1)
	}

	// Make a func to connect to the db, so it can be mocked on tests
	dbConnector := func(dsn string) (*sql.DB, error) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		return db, nil
	}

	configs, err := getConfigs(opts, dbConnector)
	if err != nil {
		log.Printf("Cannot get configs: %s", err.Error())
		os.Exit(1)
	}

	formatter, err := getFormatter(opts.OutputFmt)
	if err != nil {
		log.Printf("Cannot get output formatter: %s", err.Error())
		os.Exit(1)
	}

	diffs := compare(configs)

	formattedOutput, err := formatter.Format(diffs)
	if err != nil {
		log.Printf("There was an error formatting differences: %s", err.Error())
		os.Exit(1)
	}

	fmt.Print(formattedOutput)
}

func getFormatter(formatter string) (outputFormatter, error) {
	switch formatter {
	case "json":
		return &jsonOutput{}, nil
	case "prettyJson":
		return &jsonOutput{prettyStyle: true}, nil
	case "plain":
		return &plainOutput{}, nil
	default:
		return nil, errors.New("The specified output format doesn't exist")
	}
}

func newCNFReader(filename string) (configReader, error) {
	cfg, err := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true}, filename)
	if err != nil {
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("Invalid file: %s", filename)
	}

	cnf := &config{configType: "cnf", entries: make(map[string]interface{})}

	for _, key := range cfg.Section("mysqld").Keys() {
		cnf.entries[key.Name()] = key.Value()
	}

	return cnf, nil
}

func newMySQLReader(db *sql.DB) (configReader, error) {
	// Since the MySQL driver uses a lazy connection, check if we really can
	// connect to the db
	if err := db.Ping(); err != nil {
		return nil, err
	}

	rows, err := db.Query("SHOW VARIABLES")
	if err != nil {
		return nil, err
	}

	ini := &config{configType: "mysql", entries: make(map[string]interface{})}

	for rows.Next() {
		var key string
		var val interface{}
		err := rows.Scan(&key, &val)
		if err != nil {
			continue
		}

		ini.entries[key] = val
	}
	return ini, nil
}

/*
   We need to compare cfg1 vs cfg2 and cfg2 vs cfg1.
   Configs can be:

    cfg1      | cfg2
   -----------+----------
    key1 = A  | key1 = A
    key2 = B  | key2 = C
    key3 = D  |
              | key4 = E

	So we need 2 inner loops: first through cfg1 keys and then through
	cfg2 keys to be able to compare the keys that exist in cfg2 but are
	missing in cfg1.

	MySQL SHOW VARIABLES will return ALL variables but we must skip variables
	in MySQL config that are missing in the cnf.
	In the example above, if cfg2 is "cnf" type, key4 must be included in
	the diff but, if cfg2 type is "mysql", it must be excluded from the diff.

*/
func compare(configs []configReader) map[string][]interface{} {
	diffs := make(map[string][]interface{})

	if len(configs) < 2 {
		return nil
	}
	for i := 1; i < len(configs); i++ {

		for key, value1 := range configs[0].Entries() {
			value2, ok := configs[i].Get(key)
			if !ok && (configs[0].Type() != "mysql" || configs[0].Type() == configs[1].Type()) {
				addDiff(diffs, key, value1, "<Missing>")
				continue
			}

			// Adjust numbers truncating unnecessary 0s so 10.0 (as string) == 10
			float1, err := strconv.ParseFloat(fmt.Sprintf("%s", value1), 64)
			if err == nil {
				value1 = fmt.Sprintf("%.0f", float1)
			}

			float2, err := strconv.ParseFloat(fmt.Sprintf("%s", value2), 64)
			if err == nil {
				value2 = fmt.Sprintf("%.0f", float2)
			}

			if fmt.Sprintf("%s", value1) != fmt.Sprintf("%s", value2) {
				addDiff(diffs, key, value1, value2)
				continue
			}
		}

		for key, value1 := range configs[i].Entries() {
			_, ok := configs[0].Get(key)
			if !ok && (configs[i].Type() != "mysql" || configs[0].Type() == configs[i].Type()) {
				addDiff(diffs, key, "<Missing>", value1)
			}
		}
	}

	return diffs
}

func addDiff(diffs map[string][]interface{}, key string, value1, value2 interface{}) {
	if _, ok := diffs[key]; !ok {
		diffs[key] = append(diffs[key], value1)
	}
	diffs[key] = append(diffs[key], value2)
}

func processParams(arguments []string) (*options, error) {
	opts := &options{}

	fs := flag.NewFlagSet("default", flag.ContinueOnError)
	fs.StringArrayVarP(&opts.CNFs, "cnf", "c", nil, "cnf file name")
	fs.VarP(opts.DSNs, "dsn", "d", "full db dsn. Example: user:pass@tcp(127.1:3306)")
	fs.StringVarP(&opts.OutputFmt, "output", "o", "plain", "Output formatting. Could be json, prettyJson or plain.")

	err := fs.Parse(arguments)

	if err != nil {
		return nil, err
	}

	fs.SortFlags = false
	fs.Visit(func(f *flag.Flag) {
		if opts.compareBase != "" {
			return
		}
		switch f.Name {
		case "cnf":
			opts.compareBase = "cnf"
		case "dsn":
			opts.compareBase = "dsn"
		}
	})

	return opts, nil
}

func getConfigs(opts *options, dbConnector func(string) (*sql.DB, error)) ([]configReader, error) {
	var configs []configReader

	cnfs, err := getCNFs(opts.CNFs)
	if err != nil {
		return nil, err
	}

	mysqls, err := getMySQLs(opts.DSNs, dbConnector)
	if err != nil {
		return nil, err
	}

	if opts.compareBase == "mysql" {
		configs = append(mysqls, cnfs...)
	} else {
		configs = append(cnfs, mysqls...)
	}

	return configs, nil
}

func getCNFs(filenames []string) ([]configReader, error) {
	var configs []configReader

	for _, filename := range filenames {
		cfg, err := newCNFReader(filename)
		if err != nil {
			return nil, fmt.Errorf("Cannot read %s: %s", filename, err.Error())
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}

func getMySQLs(dsns dsnFlags, dbConnector func(string) (*sql.DB, error)) ([]configReader, error) {
	var configs []configReader

	for _, dsn := range dsns {
		db, err := dbConnector(dsn))
		if err != nil {
			return nil, fmt.Errorf("Cannot connect to the db %s", err.Error())
		}
		cfg, err := newMySQLReader(db)
		if err != nil {
			return nil, fmt.Errorf("Cannot read the config variables: %s", err.Error())
		}
		configs = append(configs, cfg)
	}

	return configs, nil
}
