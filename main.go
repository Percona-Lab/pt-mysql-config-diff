package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/Percona-Lab/pt-mysql-config-diff/internal/confreader"
	"github.com/Percona-Lab/pt-mysql-config-diff/ptdsn"
	_ "github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	re = regexp.MustCompile("(?i)(\\d+)([kmg])")

	app          = kingpin.New("pt-config-diff", "pt-config-diff")
	cnfs         = app.Arg("cnf", "Config file or DNS in the form h=host,P=port,u=user,p=pass").Strings()
	outputFormat = app.Flag("format", "Output format: text or json.").Default("text").String()
	version      = app.Flag("version", "Show version and exit").Bool()

	Version   = "0.0.0."
	Commit    = "<sha1>"
	Branch    = "branch-name"
	Build     = "2017-01-01"
	GoVersion = "1.9.2"
)

func main() {

	app.Parse(os.Args[1:])

	if *version {
		fmt.Printf("Version   : %s\n", Version)
		fmt.Printf("Commit    : %s\n", Commit)
		fmt.Printf("Branch    : %s\n", Branch)
		fmt.Printf("Build     : %s\n", Build)
		fmt.Printf("Go version: %s\n", GoVersion)
		return
	}

	if *outputFormat != "text" && *outputFormat != "json" {
		*outputFormat = "text"
	}

	// To make testing easier because we can pass a function that returns a mock db connection
	dbConnector := func(dsn string) (*sql.DB, error) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		if db.Ping() != nil {
			return nil, errors.Wrapf(err, "Cannot connect to MySQL at %q", dsn)
		}
		return db, nil
	}

	configs, err := getConfigs(*cnfs, dbConnector)
	if err != nil {
		log.Printf("Cannot get configs: %s", err.Error())
		os.Exit(1)
	}

	diffs := compare(configs)

	switch *outputFormat {
	case "text":
		printTextDiff(diffs)
	case "json":
		printJsonDiff(diffs)
	}

}

func printTextDiff(diffs map[string][]interface{}) {
	var keyLen, rightLen, leftLen int

	for key, val := range diffs {
		if l := len(fmt.Sprintf("%v", key)); l > keyLen {
			keyLen = l
		}
		if l := len(fmt.Sprintf("%v", val[0])); l > leftLen && l < 40 {
			leftLen = l
		}
		if l := len(fmt.Sprintf("%v", val[1])); l > rightLen && l < 40 {
			rightLen = l
		}
	}
	format := fmt.Sprintf("%%%ds: %%%dv <-> %%%dv\n", keyLen, leftLen, rightLen)

	for key, val := range diffs {
		fmt.Printf(format, key, val[0], val[1])
	}
}

func printJsonDiff(diffs map[string][]interface{}) {
	b, _ := json.MarshalIndent(diffs, "", "  ")
	fmt.Println(string(b))
}

/*
   We need to compare cfg1 vs cfg2 and cfg2 vs cfg1.
   Configs can be:

    cfg1      | cfg2
   -----------+----------
    leftkey1 = A  | key1 = A
    leftkey2 = B  | key2 = C
    leftkey3 = D  |
                  | key4 = E

	So we need 2 inner loops: first through cfg1 leftkeys and then through
	cfg2 leftkeys to be able to compare the keys that exist in cfg2 but are
	missing in cfg1.

	MySQL SHOW VARIABLES will return ALL variables but we must skip variables
	in MySQL config that are missing in the cnf.
	In the example above, if cfg2 is "cnf" type, leftkey4 must be included in
	the diff but, if cfg2 type is "mysql", it must be excluded from the diff.

*/
func compare(configs []confreader.ConfigReader) map[string][]interface{} {
	diffs := make(map[string][]interface{})

	if len(configs) < 2 {
		return nil
	}

	for i := 1; i < len(configs); i++ {
		canSkipMissingLeftKey := (configs[0].Type() == "cnf" && configs[i].Type() != "cnf") || configs[0].Type() == "defaults"

		canSkipMissingRightKey := (configs[i].Type() == "cnf" && configs[0].Type() != "cnf") || configs[i].Type() == "defaults"

		for leftkey, leftval := range configs[0].Entries() {
			rightval, ok := configs[i].Get(leftkey)
			if !ok {
				if !canSkipMissingRightKey {
					addDiff(diffs, leftkey, leftval, "<Missing>")
				}
				continue
			}

			leftval = adjustValue(leftval)
			rightval = adjustValue(rightval)

			if leftval != rightval {
				addDiff(diffs, leftkey, leftval, rightval)
			}
		}

		if canSkipMissingLeftKey {
			continue
		}

		for rightkey, rightval := range configs[i].Entries() {
			if _, ok := configs[0].Get(rightkey); !ok {
				addDiff(diffs, rightkey, "<Missing>", rightval)
			}
		}
	}
	return diffs
}

func adjustValue(val interface{}) interface{} {
	units := map[string]int64{
		"k": 1024,
		"m": 1024 * 1024,
		"g": 1024 * 1024 * 1024,
		"t": 1024 * 1024 * 1024 * 1024,
	}

	switch val.(type) {
	case string:
		val := fmt.Sprintf("%v", val)
		// var re = regexp.MustCompile("(?i)(\\d+)([kmg])")
		if strings.ToLower(val) == "yes" || strings.ToLower(val) == "on" || strings.ToLower(val) == "true" {
			return 1
		}

		if strings.ToLower(val) == "no" || strings.ToLower(val) == "off" || strings.ToLower(val) == "false" {
			return 0
		}

		if m := re.FindStringSubmatch(val); len(m) == 3 {
			number, _ := strconv.ParseInt(m[1], 10, 64)
			multiplier := units[strings.ToLower(m[2])]
			return fmt.Sprintf("%.0f", float64(number*multiplier))
		}

		if f, err := strconv.ParseFloat(fmt.Sprintf("%s", val), 64); err == nil {
			return fmt.Sprintf("%.0f", f)
		}
	}
	return val
}

func addDiff(diffs map[string][]interface{}, leftkey string, leftval, rightval interface{}) {
	if _, ok := diffs[leftkey]; !ok {
		diffs[leftkey] = append(diffs[leftkey], leftval)
	}
	diffs[leftkey] = append(diffs[leftkey], rightval)
}

func getConfigs(cnfs []string, dbConnector func(string) (*sql.DB, error)) ([]confreader.ConfigReader, error) {
	var configs []confreader.ConfigReader

	for _, spec := range cnfs {
		if _, err := os.Stat(spec); err == nil {
			if cnf, err := getCNF(spec); err == nil {
				configs = append(configs, cnf)
			} else {
				fmt.Println(err.Error())
			}
			continue
		}
		if cnf, err := getMySQL(spec, dbConnector); err == nil {
			configs = append(configs, cnf)
		} else {
			fmt.Println(err.Error())
		}
	}

	return configs, nil
}

func getCNF(filename string) (confreader.ConfigReader, error) {
	cfg, err := confreader.NewCNFReader(filename)
	if err != nil {
		return nil, errors.Wrapf(err, "Cannot read %s", filename)
	}

	if len(cfg.Entries()) == 0 {
		cfg, err = confreader.NewDefaultsParser(filename)
		if err != nil {
			return nil, errors.Wrapf(err, "Cannot read %s", filename)
		}
	}

	return cfg, nil
}

func getMySQL(dsns string, dbConnector func(string) (*sql.DB, error)) (confreader.ConfigReader, error) {
	dsn := ptdsn.NewPTDSN(dsns)

	db, err := dbConnector(dsn.String())
	if err != nil {
		return nil, fmt.Errorf("Cannot connect to the db %s", err.Error())
	}
	if db == nil {
		return nil, fmt.Errorf("Cannot connect to the database on %q", dsns)
	}
	defer db.Close()

	cfg, err := confreader.NewMySQLReader(db)
	if err != nil {
		return nil, fmt.Errorf("Cannot read the config variables: %s", err.Error())
	}

	return cfg, nil
}
