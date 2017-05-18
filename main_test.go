package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"testing"

	sqlmock "gopkg.in/DATA-DOG/go-sqlmock.v1"
)

func TestCompareCNFs(t *testing.T) {

	mockConfig1 := &config{
		configType: "cnf",
		entries: map[string]interface{}{
			"key1": "value1",
			"key2": 2,
			"key3": true,
		},
	}

	mockConfig2 := &config{
		configType: "cnf",
		entries: map[string]interface{}{
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

	got := compare([]configReader{mockConfig1, mockConfig2})

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got:\n%#v\nWant:\n %#v\n", got, want)
	}

}

func TestCompareCNFvsMySQL(t *testing.T) {

	mockConfig1 := &config{
		configType: "cnf",
		entries: map[string]interface{}{
			"key1": "value1",
			"key2": 2,
			"key3": true,
		},
	}

	// MySQL SHOW VARIABLES will return ALL variables but we must skip variables in
	// MySQL config that are missing in the cnf.
	// In this particular case, key4 should not be included in the diff
	mockConfig2 := &config{
		configType: "mysql",
		entries: map[string]interface{}{
			"key1": "value1",
			"key2": 3,
			"key4": true,
		},
	}

	want := map[string][]interface{}{
		"key2": []interface{}{2, 3},
		"key3": []interface{}{true, "<Missing>"},
	}

	got := compare([]configReader{mockConfig1, mockConfig2})

	if !reflect.DeepEqual(got, want) {
		t.Errorf("Got:\n%#v\nWant:\n %#v\n", got, want)
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

	cnf, err := newCNFReader("some_fake_file")
	if err == nil {
		t.Error("Should return error on invalid files")
	}

	want := &config{
		configType: "cnf",
		entries: map[string]interface{}{
			"sql_mode":                       "IGNORE_SPACE,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_AUTO_CREATE_USER,NO_ENGINE_SUBSTITUTION",
			"innodb_buffer_pool_size":        "512M",
			"log_slow_rate_limit":            "100.1234",
			"log_slow_verbosity":             "full",
			"basedir":                        "/usr",
			"innodb_flush_log_at_trx_commit": "2",
			"log_slow_rate_type":             "query",
			"log_slow_admin_statements":      "ON",
			"pid-file":                       "/var/run/mysqld/mysqld.pid",
			"socket":                         "/var/run/mysqld/mysqld.sock",
			"bind-address":                   "127.0.0.1",
			"slow_query_log":                 "OFF",
			"user":                           "mysql",
			"log_slow_slave_statements":         "ON",
			"datadir":                           "/var/lib/mysql",
			"local-infile":                      "1",
			"explicit_defaults_for_timestamp":   "true",
			"secure-file-priv":                  "\"\"",
			"log-error":                         "/var/log/mysql/error.log",
			"log_output":                        "file",
			"slow_query_log_use_global_control": "all",
			"tmpdir":                           "/tmp",
			"lc-messages-dir":                  "/usr/share/mysql",
			"long_query_time":                  "0",
			"port":                             "3306",
			"max_allowed_packet":               "128M",
			"symbolic-links":                   "0",
			"key_buffer_size":                  "1024M",
			"slow_query_log_file":              "/var/log/mysql/slow.log",
			"slow_query_log_always_write_time": "1",
		},
	}

	cnf, err = newCNFReader("./test/mysqld.cnf")
	if err != nil {
		t.Errorf("Shouldn't return error on existent file: %s", err.Error())
	}

	if !reflect.DeepEqual(cnf, want) {
		fmt.Printf("Got:\n%#v\nWant: %#v\n", cnf, want)
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

	want := &config{
		configType: "mysql",
		entries: map[string]interface{}{
			"innodb_buffer_pool_size": "512M",
			"log_slow_rate_limit":     "100.1234",
			"log_slow_verbosity":      "full",
		},
	}

	cnf, err := newMySQLReader(db)
	if err != nil {
		t.Errorf("Shouldn't return error on mock up db: %s", err.Error())
	}

	if !reflect.DeepEqual(cnf, want) {
		fmt.Printf("Got:\n%#v\nWant: %#v\n", cnf, want)
	}

}

func TestGetConfigs(t *testing.T) {

	opts := &options{
		CNFs: []string{"./test/mysqld.cnf"},
		DSNs: []string{"mock:pass@tcp(127.1:3306)/"},
	}

	mockDBConnector := func(dns string) (*sql.DB, error) {
		db, mock, err := sqlmock.New()
		if err != nil {
			t.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
		}

		columns := []string{"Variable_name", "Value"}

		mock.ExpectQuery("SHOW VARIABLES").WillReturnRows(sqlmock.NewRows(columns).
			AddRow("innodb_buffer_pool_size", "512M").
			AddRow("log_slow_rate_limit", "100.1234").
			AddRow("log_slow_verbosity", "full"))

		return db, nil
	}

	configs, err := getConfigs(opts, mockDBConnector)
	if err != nil {
		t.Error(err)
	}

	if len(configs) != 2 {
		t.Errorf("There must be 2 configs, got %d", len(configs))
	}

	if configs[0].Type() != "cnf" {
		t.Errorf("First config should be a cnf file. Got: %s", configs[0].Type())
	}

	if configs[1].Type() != "mysql" {
		t.Errorf("Second config should be 'mysql'. Got: %s", configs[1].Type())
	}

}

func TestJsonOutput(t *testing.T) {

	mockConfig1 := &config{
		configType: "cnf",
		entries: map[string]interface{}{
			"key1": "value1",
			"key2": 2,
			"key3": true,
		},
	}

	mockConfig2 := &config{
		configType: "cnf",
		entries: map[string]interface{}{
			"key1": "value1",
			"key2": 3,
			"key4": true,
		},
	}

	want := `{"key2":[2,3],"key3":[true,"\u003cMissing\u003e"],"key4":["\u003cMissing\u003e",true]}`

	diff := compare([]configReader{mockConfig1, mockConfig2})
	jsonFormatter := &jsonOutput{}

	got, _ := jsonFormatter.Format(diff)

	if got != want {
		t.Errorf("Got:\n%#v\nWant:\n %#v\n", got, diff)
	}

}

func TestProcessParams(t *testing.T) {
	args := []string{"--dsn=h=127.1,P=12345,u=user1,p=pass,D=db,t=table", "--cnf=mysqld.conf"}
	opts, err := processParams(args)
	if err != nil {
		t.Errorf("Cannot parse params")
	}
	if opts.compareBase != "dsn" {
		t.Errorf("Compare base must be dsn. Got %s", opts.compareBase)
	}

	args = []string{"--cnf=mysqld.conf", "--dsn=h=127.1,P=12345,u=user1,p=pass,D=db,t=table"}
	opts, err = processParams(args)
	if opts.compareBase != "cnf" {
		t.Errorf("Compare base must be cnf. Got %s", opts.compareBase)
	}
}
