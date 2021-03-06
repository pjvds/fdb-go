// FoundationDB Go options translator
// Copyright (c) 2013 FoundationDB, LLC

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"encoding/xml"
	"io/ioutil"
	"fmt"
	"log"
	"strings"
	"os"
	"unicode"
	"unicode/utf8"
	"go/doc"
)

type Option struct {
	Name string `xml:"name,attr"`
	Code int `xml:"code,attr"`
	ParamType string `xml:"paramType,attr"`
	ParamDesc string `xml:"paramDescription,attr"`
	Description string `xml:"description,attr"`
}
type Scope struct {
	Name string `xml:"name,attr"`
	Option []Option
}
type Options struct {
	Scope []Scope
}

func writeOptString(receiver string, function string, opt Option) {
	fmt.Printf(`func (o %s) %s(param string) error {
	return o.setOpt(%d, []byte(param))
}
`, receiver, function, opt.Code)
}

func writeOptBytes(receiver string, function string, opt Option) {
	fmt.Printf(`func (o %s) %s(param []byte) error {
	return o.setOpt(%d, param)
}
`, receiver, function, opt.Code)
}

func writeOptInt(receiver string, function string, opt Option) {
	fmt.Printf(`func (o %s) %s(param int64) error {
	b, e := int64ToBytes(param)
	if e != nil {
		return e
	}
	return o.setOpt(%d, b)
}
`, receiver, function, opt.Code)
}

func writeOptNone(receiver string, function string, opt Option) {
	fmt.Printf(`func (o %s) %s() error {
	return o.setOpt(%d, nil)
}
`, receiver, function, opt.Code)
}

func writeOpt(receiver string, opt Option) {
	function := "Set" + translateName(opt.Name)

	fmt.Println()

	if opt.Description != "" {
		fmt.Printf("// %s\n", opt.Description)
		if opt.ParamDesc != "" {
			fmt.Printf("//\n// Parameter: %s\n", opt.ParamDesc)
		}
	} else {
		fmt.Printf("// Not yet implemented.\n")
	}

	switch opt.ParamType {
	case "String":
		writeOptString(receiver, function, opt)
	case "Bytes":
		writeOptBytes(receiver, function, opt)
	case "Int":
		writeOptInt(receiver, function, opt)
	case "":
		writeOptNone(receiver, function, opt)
	default:
		log.Fatalf("Totally unexpected ParamType %s", opt.ParamType)
	}
}

func translateName(old string) string {
	return strings.Replace(strings.Title(strings.Replace(old, "_", " ", -1)), " ", "", -1)
}

func lowerFirst (s string) string {
        if s == "" {
                return ""
        }
        r, n := utf8.DecodeRuneInString(s)
        return string(unicode.ToLower(r)) + s[n:]
}

func writeMutation(opt Option) {
	desc := lowerFirst(opt.Description)
	tname := translateName(opt.Name)
	fmt.Printf(`
// %s %s
func (t Transaction) %s(key KeyConvertible, param []byte) {
	t.atomicOp(key.FDBKey(), param, %d)
}
`, tname, desc, tname, opt.Code)
}

func writeEnum(scope Scope, opt Option, delta int) {
	fmt.Println()
	if opt.Description != "" {
		doc.ToText(os.Stdout, opt.Description, "    // ", "", 73)
		// fmt.Printf("	// %s\n", opt.Description)
	}
	fmt.Printf("	%s %s = %d\n", scope.Name + translateName(opt.Name), scope.Name, opt.Code + delta)
}

func main() {
	var err error

	v := Options{}

	data, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatal(err)
	}

	err = xml.Unmarshal(data, &v)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Print(`// DO NOT EDIT THIS FILE BY HAND. This file was generated using
// translate_fdb_options.go, part of the fdb-go repository, and a copy of the
// fdb.options file (installed as part of the FoundationDB client, typically
// found as /usr/include/foundationdb/fdb.options).

// To regenerate this file, from the top level of an fdb-go repository checkout,
// run:
// $ go run _util/translate_fdb_options.go < /usr/include/foundationdb/fdb.options > fdb/generated.go

package fdb

import (
	"bytes"
	"encoding/binary"
)

func int64ToBytes(i int64) ([]byte, error) {
	buf := new(bytes.Buffer)
	if e := binary.Write(buf, binary.LittleEndian, i); e != nil {
		return nil, e
	}
	return buf.Bytes(), nil
}
`)

	for _, scope := range(v.Scope) {
		if strings.HasSuffix(scope.Name, "Option") {
			receiver := scope.Name + "s"

			for _, opt := range(scope.Option) {
				if opt.Description != "Deprecated" { // Eww
					writeOpt(receiver, opt)
				}
			}
			continue
		}

		if scope.Name == "MutationType" {
			for _, opt := range(scope.Option) {
				if opt.Description != "Deprecated" { // Eww
					writeMutation(opt)
				}
			}
			continue
		}

		// We really need the default StreamingMode (0) to be ITERATOR
		var d int
		if scope.Name == "StreamingMode" {
			d = 1
		}

		// ConflictRangeType shouldn't be exported
		if scope.Name == "ConflictRangeType" {
			scope.Name = "conflictRangeType"
		}

		fmt.Printf(`
type %s int
const (
`, scope.Name)
		for _, opt := range(scope.Option) {
			writeEnum(scope, opt, d)
		}
		fmt.Println(")")
	}
}
