package main

import (
	"fmt"
	"log"
	"net"
	"os"
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
	defaultTypeLint(items)
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

func defaultTypeLint(items []item) {
	log.Println("check default value types...")
	for _, item := range items {
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
