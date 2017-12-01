package confreader

import (
	"database/sql"
)

func NewMySQLReader(db *sql.DB) (ConfigReader, error) {
	rows, err := db.Query("SHOW GLOBAL VARIABLES")
	if err != nil {
		return nil, err
	}

	ini := &Config{ConfigType: "mysql", EntriesMap: make(map[string]interface{})}

	for rows.Next() {
		var key, val string
		err := rows.Scan(&key, &val)
		if err != nil {
			continue
		}
		ini.EntriesMap[key] = val
	}
	return ini, nil
}
