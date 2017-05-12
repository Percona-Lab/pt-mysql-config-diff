package main

import (
	"bytes"
	"encoding/json"
	"fmt"
)

type outputFormatter interface {
	Format(map[string][]interface{}) (string, error)
}

type jsonOutput struct {
	prettyStyle bool
}

func (o *jsonOutput) Format(diff map[string][]interface{}) (string, error) {
	output, err := json.Marshal(diff)
	if o.prettyStyle {
		output, err = json.MarshalIndent(diff, "", "\t")
	}
	if err != nil {
		return "", err
	}

	return string(output), nil
}

type plainOutput struct {
}

func (o *plainOutput) Format(diff map[string][]interface{}) (string, error) {
	var buffer bytes.Buffer
	for key, val := range diff {
		buffer.WriteString(fmt.Sprintf("%35s: %40s : %40s\n", key, val[0], val[1]))
	}

	return string(buffer.String()), nil
}
