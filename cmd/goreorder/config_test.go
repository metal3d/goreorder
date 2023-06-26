package main

import (
	"bytes"
	"testing"

	"github.com/metal3d/goreorder/ordering"
	"gopkg.in/yaml.v3"
)

func TestNoConfigFile(t *testing.T) {
	writer := bytes.NewBuffer([]byte{})
	printConfigFile(&ReorderConfig{
		FormatToolName: "gofmt",
		Write:          false,
		Verbose:        false,
		ReorderTypes:   false,
		MakeDiff:       false,
		DefOrder:       ordering.DefaultOrder,
	}, writer)

	conf := make(map[string]interface{})
	err := yaml.Unmarshal(writer.Bytes(), &conf)
	if err != nil {
		t.Error(err)
	}
	if conf["format"] != "gofmt" {
		t.Error("format should be gofmt")
	}
	if conf["write"] != false {
		t.Error("write should be false")
	}
	if conf["verbose"] != false {
		t.Error("verbose should be false")
	}
	if conf["reorder-types"] != false {
		t.Error("reorder-types should be false")
	}
	if conf["diff"] != false {
		t.Error("diff should be false", conf["diff"])
	}
	if v, ok := conf["order"]; ok {
		switch v := v.(type) {
		case nil:
			t.Error("order should not be nil")
		case []interface{}:
			// bind to string
			order := make([]string, len(v))
			for i, val := range v {
				order[i] = val.(string)
			}
			if len(order) != len(ordering.DefaultOrder) {
				t.Error("order should have default length")
			}
		}
	} else {
		t.Error("order should be default")
	}
}
