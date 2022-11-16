package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/jkstack/jkframe/utils"
	"gopkg.in/yaml.v3"
)

type itemType string

const (
	typeString   itemType = "string"
	typeCsv      itemType = "csv"
	typeInt      itemType = "int"
	typeUint     itemType = "uint"
	typeFloat    itemType = "float"
	typeBool     itemType = "bool"
	typeNetAddr  itemType = "naddr"
	typePath     itemType = "path"
	typeBytes    itemType = "bytes"
	typeDuration itemType = "duration"
)

func (t *itemType) UnmarshalYAML(value *yaml.Node) error {
	switch value.Value {
	case string(typeString):
		*t = typeString
	case string(typeCsv):
		*t = typeCsv
	case string(typeInt):
		*t = typeInt
	case string(typeUint):
		*t = typeUint
	case string(typeFloat):
		*t = typeFloat
	case string(typeBool):
		*t = typeBool
	case string(typeNetAddr):
		*t = typeNetAddr
	case string(typePath):
		*t = typePath
	case string(typeBytes):
		*t = typeBytes
	case string(typeDuration):
		*t = typeDuration
	default:
		return fmt.Errorf("unsupported type: %s", value.Value)
	}
	return nil
}

func (t itemType) String() string {
	return string(t)
}

type item struct {
	Key  string `yaml:"key"`
	Name struct {
		Zh string `yaml:"zh"`
	} `yaml:"name"`
	Desc struct {
		Zh string `yaml:"zh"`
	} `yaml:"desc"`
	Type          itemType    `yaml:"type"`
	Default       interface{} `yaml:"default"`
	CsvValid      []string    `yaml:"csv_valid"`
	StrValid      string      `yaml:"str_valid"`
	Min           interface{} `yaml:"min"`
	Max           interface{} `yaml:"max"`
	Length        uint        `yaml:"len"`
	AllowRelative bool        `yaml:"allow_relative"`
	Enabled       struct {
		When struct {
			Target  string      `yaml:"target"`
			Contain string      `yaml:"contain"`
			Equal   interface{} `yaml:"equal"`
		} `yaml:"when"`
	} `yaml:"enabled"`
}

func main() {
	if len(os.Args) <= 1 {
		fmt.Println("missing manifest.yaml file dir")
		os.Exit(1)
	}
	var items []item
	utils.Assert(decode(os.Args[1], &items))
	requiredLint(items)
	defaultValueLint(items)
	csvLint(items)
	strLint(items)
	minMaxValueLint(items)
	lengthLint(items)
	allowRelativeLint(items)
	enabledLint(items)
}

func decode(dir string, items *[]item) error {
	log.Println("decoding...")
	f, err := os.Open(dir)
	utils.Assert(err)
	defer f.Close()
	return yaml.NewDecoder(f).Decode(items)
}

func requiredLint(items []item) {
	log.Println("check required fields...")
	for i, item := range items {
		if len(item.Key) == 0 {
			panic(fmt.Sprintf("missing key on item %d", i+1))
		}
		if len(item.Name.Zh) == 0 {
			panic(fmt.Sprintf("missing name.zh on item %s", item.Key))
		}
		if len(item.Desc.Zh) == 0 {
			panic(fmt.Sprintf("missing desc.zh on item %s", item.Key))
		}
		if len(item.Type) == 0 {
			panic(fmt.Sprintf("missing type on item %s", item.Key))
		}
	}
}

func defaultValueLint(items []item) {
	log.Println("check default value types...")
	for _, item := range items {
		if item.Default == nil {
			continue
		}
		onPanic := func(err ...error) {
			if len(err) == 0 {
				panic(fmt.Sprintf("invalid default value type on item %s", item.Key))
			}
			if err[0] != nil {
				panic(fmt.Sprintf("invalid default value type on item %s: %v", item.Key, err[0]))
			}
		}
		switch item.Type {
		case typeString, typePath:
			if _, ok := item.Default.(string); !ok {
				onPanic()
			}
		case typeCsv:
			if _, ok := item.Default.([]interface{}); !ok {
				onPanic()
			}
		case typeInt, typeUint:
			if _, ok := item.Default.(int); !ok {
				onPanic()
			}
		case typeFloat:
			if _, ok := item.Default.(float64); !ok {
				if _, ok := item.Default.(int); !ok {
					onPanic()
				}
			}
		case typeBool:
			if _, ok := item.Default.(bool); !ok {
				onPanic()
			}
		case typeNetAddr:
			if _, ok := item.Default.(string); !ok {
				onPanic()
			}
			_, port, err := net.SplitHostPort(item.Default.(string))
			onPanic(err)
			_, err = strconv.ParseUint(port, 10, 16)
			onPanic(err)
		case typeBytes:
			if _, ok := item.Default.(string); !ok {
				onPanic()
			}
			_, err := humanize.ParseBytes(item.Default.(string))
			onPanic(err)
		case typeDuration:
			if _, ok := item.Default.(string); !ok {
				onPanic()
			}
			_, err := time.ParseDuration(item.Default.(string))
			onPanic(err)
		}
	}
}

func csvLint(items []item) {
	log.Println("check csv lint...")
	for _, item := range items {
		if len(item.CsvValid) == 0 {
			continue
		}
		if item.Type != typeCsv {
			panic(fmt.Sprintf("unsupported csv_valid for item %s: type %s", item.Key, item.Type))
		}
		if item.Default == nil {
			continue
		}
		for _, it := range item.Default.([]interface{}) {
			found := false
			for _, valid := range item.CsvValid {
				if fmt.Sprintf("%v", it) == valid {
					found = true
					break
				}
			}
			if !found {
				panic(fmt.Sprintf("unsupported default value for item %s: value %v valid %v",
					item.Key, it, item.CsvValid))
			}
		}
	}
}

func strLint(items []item) {
	log.Println("check string lint...")
	for _, item := range items {
		if len(item.StrValid) == 0 {
			continue
		}
		_, err := regexp.Compile(item.StrValid)
		if err != nil {
			panic(fmt.Sprintf("invalid str_valid setting on item %s: %v", item.Key, err))
		}
	}
}

func minMaxValueLint(items []item) {
	log.Println("check min/max value types...")
	for _, item := range items {
		if item.Min == nil && item.Max == nil {
			continue
		}
		onPanic := func(name string, err ...error) {
			if len(err) == 0 {
				panic(fmt.Sprintf("invalid %s value type on item %s", name, item.Key))
			}
			if err[0] != nil {
				panic(fmt.Sprintf("invalid %s value type on item %s: %v", name, item.Key, err[0]))
			}
		}
		switch item.Type {
		case typeInt, typeUint:
			if _, ok := item.Min.(int); item.Min != nil && !ok {
				onPanic("min")
			}
			if _, ok := item.Max.(int); item.Max != nil && !ok {
				onPanic("max")
			}
		case typeFloat:
			if item.Min != nil {
				if _, ok := item.Min.(float64); !ok {
					if _, ok := item.Min.(int); !ok {
						onPanic("min")
					}
				}
			}
			if item.Max != nil {
				if _, ok := item.Max.(float64); !ok {
					if _, ok := item.Max.(int); !ok {
						onPanic("max")
					}
				}
			}
		case typeBytes:
			if item.Min != nil {
				if _, ok := item.Min.(string); !ok {
					onPanic("min")
				}
				_, err := humanize.ParseBytes(item.Min.(string))
				onPanic("min", err)
			}
			if item.Max != nil {
				if _, ok := item.Max.(string); !ok {
					onPanic("max")
				}
				_, err := humanize.ParseBytes(item.Max.(string))
				onPanic("max", err)
			}
		case typeDuration:
			if item.Min != nil {
				if _, ok := item.Min.(string); !ok {
					onPanic("min")
				}
				_, err := time.ParseDuration(item.Min.(string))
				onPanic("min", err)
			}
			if item.Max != nil {
				if _, ok := item.Max.(string); !ok {
					onPanic("max")
				}
				_, err := time.ParseDuration(item.Max.(string))
				onPanic("max", err)
			}
		default:
			if item.Min != nil {
				onPanic("min", fmt.Errorf("unsupported for type %s", item.Type.String()))
			}
			if item.Max != nil {
				onPanic("max", fmt.Errorf("unsupported for type %s", item.Type.String()))
			}

		}
	}
}

func lengthLint(items []item) {
	log.Println("check length limit...")
	for _, item := range items {
		if item.Length == 0 {
			continue
		}
		if item.Type != typeString &&
			item.Type != typeCsv &&
			item.Type != typePath {
			panic(fmt.Sprintf("unsupported length for item %s: type %s", item.Key, item.Type))
		}
	}
}

func allowRelativeLint(items []item) {
	log.Println("check allow_relative limit...")
	for _, item := range items {
		if !item.AllowRelative {
			continue
		}
		if item.Type != typePath {
			panic(fmt.Sprintf("unsupported allow_relative for item %s: type %s", item.Key, item.Type))
		}
	}
}

func enabledLint(items []item) {
	log.Println("check enabled lint...")
	m := make(map[string]item)
	for _, item := range items {
		m[item.Key] = item
	}
	for _, item := range items {
		if len(item.Enabled.When.Target) == 0 {
			continue
		}
		target, ok := m[item.Enabled.When.Target]
		if !ok {
			panic(fmt.Sprintf("missing enabled.when.target on item %s", item.Key))
		}
		if len(item.Enabled.When.Contain) > 0 &&
			item.Enabled.When.Equal != nil {
			panic(fmt.Sprintf("multi condition on item %s", item.Key))
		}
		if len(item.Enabled.When.Contain) > 0 {
			if target.Type != typeCsv {
				panic(fmt.Sprintf("target is not csv type on item %s", item.Key))
			}
			if len(target.CsvValid) > 0 {
				found := false
				for _, it := range target.CsvValid {
					if item.Enabled.When.Contain == it {
						found = true
						break
					}
				}
				if !found {
					panic(fmt.Sprintf("contain value is not in csv_valid from target on item %s", item.Key))
				}
			}
		}
		if item.Enabled.When.Equal != nil {
			// TODO: 暂时只支持bool类型
			if target.Type != typeBool {
				panic(fmt.Sprintf("target is not bool type on item %s", item.Key))
			}
		}

	}
}
