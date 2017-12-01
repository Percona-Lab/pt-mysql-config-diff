package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	"github.com/Percona-Lab/pt-mysql-config-diff/internal/confreader"
	tu "github.com/Percona-Lab/pt-mysql-config-diff/testutils"
	"github.com/kr/pretty"
	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCompareCNFs(t *testing.T) {

	mockConfig1 := &confreader.Config{
		ConfigType: "cnf",
		EntriesMap: map[string]interface{}{
			"key1": "value1",
			"key2": 2,
			"key3": true,
		},
	}

	mockConfig2 := &confreader.Config{
		ConfigType: "cnf",
		EntriesMap: map[string]interface{}{
			"key1": "value1",
			"key2": 3,
			"key4": true,
		},
	}

	want := map[string][]interface{}{
		"key2": []interface{}{2, 3},
		"key3": []interface{}{true, "<Missing>"},
		"key4": []interface{}{"<Missing>", true},
	}

	got := compare([]confreader.ConfigReader{mockConfig1, mockConfig2})

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got:\n%#v\nWant:\n%#v\n", got, want)
	}

}

func TestCompareCNFvsMySQL(t *testing.T) {

	mockConfig1 := &confreader.Config{
		ConfigType: "cnf",
		EntriesMap: map[string]interface{}{
			"key1": "value1",
			"key2": 2,
			"key3": true,
		},
	}

	// MySQL SHOW VARIABLES will return ALL variables but we must skip variables in
	// MySQL config that are missing in the cnf.
	// In this particular case, key4 should not be included in the diff
	mockConfig2 := &confreader.Config{
		ConfigType: "mysql",
		EntriesMap: map[string]interface{}{
			"key1": "value1",
			"key2": 3,
			"key4": true,
		},
	}

	want := map[string][]interface{}{
		"key2": []interface{}{2, 3},
		"key3": []interface{}{true, "<Missing>"},
	}

	got := compare([]confreader.ConfigReader{mockConfig1, mockConfig2})

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got:\n%#v\nWant:\n%#v\n", got, want)
	}

}

func TestAddDiff(t *testing.T) {

	diffs := make(map[string][]interface{})

	want1 := map[string][]interface{}{"key1": []interface{}{"value1", "value2"}}
	addDiff(diffs, "key1", "value1", "value2")
	if !reflect.DeepEqual(diffs, want1) {
		t.Errorf("Error adding key/val: Got\n%#v, want\n%#v\n", diffs, want1)
	}

	want2 := map[string][]interface{}{"key1": []interface{}{"value1", "value2", "value3"}}
	addDiff(diffs, "key1", "value1", "value3")
	if !reflect.DeepEqual(diffs, want2) {
		t.Errorf("Error adding key/val: Got\n%#v, want\n%#v\n", diffs, want2)
	}

}

func TestReadCNFs(t *testing.T) {

	cnf, err := confreader.NewCNFReader("some_fake_file")
	if err == nil {
		t.Error("Should return error on invalid files")
	}

	want := &confreader.Config{ConfigType: "cnf",
		EntriesMap: map[string]interface{}{
			"basedir":                           "/usr",
			"bind-address":                      "127.0.0.1",
			"datadir":                           "/var/lib/mysql",
			"explicit_defaults_for_timestamp":   "true",
			"innodb_buffer_pool_size":           "512M",
			"innodb_flush_log_at_trx_commit":    "2",
			"key_buffer_size":                   "512M",
			"lc-messages-dir":                   "/usr/share/mysql",
			"local-infile":                      "1",
			"log-error":                         "/var/log/mysql/error.log",
			"log_output":                        "file",
			"log_slow_admin_statements":         "ON",
			"log_slow_rate_limit":               "100.1234",
			"log_slow_rate_type":                "query",
			"log_slow_slave_statements":         "ON",
			"log_slow_verbosity":                "full",
			"long_query_time":                   "0",
			"max_allowed_packet":                "128M",
			"pid-file":                          "/var/run/mysqld/mysqld.pid",
			"port":                              "3306",
			"secure-file-priv":                  "",
			"slow_query_log":                    "OFF",
			"slow_query_log_always_write_time":  "1",
			"slow_query_log_file":               "/var/log/mysql/slow.log",
			"slow_query_log_use_global_control": "all",
			"socket":         "/var/run/mysqld/mysqld.sock",
			"sql_mode":       "IGNORE_SPACE,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION",
			"symbolic-links": "0",
			"tmpdir":         "/tmp",
			"user":           "mysql",
		},
	}
	cnf, err = confreader.NewCNFReader("./test/mysqld.cnf")
	if err != nil {
		t.Errorf("Shouldn't return error on existent file: %s", err.Error())
	}

	if !reflect.DeepEqual(cnf.Entries(), want.Entries()) {
		println(pretty.Diff(cnf.Entries(), want.Entries()))
		t.Errorf("Got:\n%#v\nWant: %#v\n", cnf, want)
	}

}

func TestReadMySQL(t *testing.T) {

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	columns := []string{"Variable_name", "Value"}

	mock.ExpectQuery("SHOW VARIABLES").WillReturnRows(sqlmock.NewRows(columns).
		AddRow("innodb_buffer_pool_size", "512M").
		AddRow("log_slow_rate_limit", "100.1234").
		AddRow("log_slow_verbosity", "full"))

	want := &confreader.Config{
		ConfigType: "mysql",
		EntriesMap: map[string]interface{}{
			"innodb_buffer_pool_size": "512M",
			"log_slow_rate_limit":     "100.1234",
			"log_slow_verbosity":      "full",
		},
	}

	cnf, err := confreader.NewMySQLReader(db)
	if err != nil {
		t.Errorf("Shouldn't return error on mock up db: %s", err.Error())
	}

	if !reflect.DeepEqual(cnf, want) {
		fmt.Printf("Got:\n%#v\nWant: %#v\n", cnf, want)
	}

}

func TestCnfVsMySQLIntegration(t *testing.T) {
	db, err := sql.Open("mysql", "root:@tcp(127.1:3306)/")
	if err != nil {
		t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}
	defer db.Close()

	cnf := &confreader.Config{
		ConfigType: "cnf",
		EntriesMap: map[string]interface{}{
			"innodb_buffer_pool_size": "128M", // default in MySQL: 134217728 = 128M
			"log_slow_rate_limit":     "100.1234",
			"log_slow_verbosity":      "full",
		},
	}

	myvars, err := confreader.NewMySQLReader(db)
	if err != nil {
		t.Errorf("Shouldn't return error on mock up db: %s", err.Error())
	}

	want := map[string][]interface{}{
		//"innodb_buffer_pool_size": []interface{}{"112M", "134217728"},
		"log_slow_rate_limit": []interface{}{"100.1234", "<Missing>"},
		"log_slow_verbosity":  []interface{}{"full", "<Missing>"},
	}

	diff := compare([]confreader.ConfigReader{cnf, myvars})
	tu.Equals(t, diff, want)
}

func TestShowNonDefaults(t *testing.T) {
	cnf := &confreader.Config{
		ConfigType: "cnf",
		EntriesMap: map[string]interface{}{
			"innodb_buffer_pool_size": "512M", // default in MySQL: 134217728 = 128M
			"log_slow_rate_limit":     "100.1234",
			"log_slow_verbosity":      "full",
			"auto-increment-offset":   2,
		},
	}

	defaults, err := confreader.NewDefaultsParser("internal/confreader/testdata/defaults.txt")
	tu.IsNil(t, err)

	want := map[string][]interface{}{
		"log_slow_verbosity":      []interface{}{"full", "<Missing>"},
		"auto-increment-offset":   []interface{}{2, "<Missing>"},
		"innodb_buffer_pool_size": []interface{}{"536870912", "134217728"},
		"log_slow_rate_limit":     []interface{}{"100.1234", "<Missing>"},
	}

	diff := compare([]confreader.ConfigReader{cnf, defaults})
	tu.Equals(t, diff, want)
}
