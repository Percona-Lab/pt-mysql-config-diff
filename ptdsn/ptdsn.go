package ptdsn

import (
	"fmt"
	"strconv"
	"strings"
)

type PTDSN struct {
	Database string
	Host     string
	Password string
	Port     int
	Table    string
	User     string
	Protocol string
}

func NewPTDSN(value string) *PTDSN {
	return parse(value)
}

func (d *PTDSN) Set(value string) error {
	d = parse(value)
	return nil
}

func (d *PTDSN) String() string {
	return fmt.Sprintf("%v:%v@%v(%v:%v)/%v", d.User, d.Password, d.Protocol, d.Host, d.Port, d.Database)
}

type PTDSNs []*PTDSN

func (d PTDSNs) Set(value string) error {
	v := parse(value)
	d = append(d, v)
	return nil
}

func (d PTDSNs) String() string {
	return ""
}

func parse(value string) *PTDSN {
	d := &PTDSN{}
	parts := strings.Split(value, ",")

	for _, part := range parts {
		m := strings.Split(part, "=")
		key := m[0]
		value := ""
		if len(m) > 1 {
			value = m[1]
		}
		switch key {
		case "D":
			d.Database = value
		case "h":
			d.Host = value
			if d.Host == "localhost" {
				d.Protocol = "unix"
			} else {
				d.Protocol = "tcp"
			}
		case "p":
			d.Password = value
		case "P":
			if port, err := strconv.ParseInt(value, 10, 64); err == nil {
				d.Port = int(port)
			}
		case "t":
			d.Table = value
		case "u":
			d.User = value
		}
	}

	if d.Protocol == "tcp" && d.Port == 0 {
		d.Port = 3306
	}

	return d
}
