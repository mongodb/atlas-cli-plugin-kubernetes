// Copyright 2025 MongoDB Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/mongodb/atlas-cli-core/config"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/jsonpathwriter"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/jsonwriter"
	"github.com/mongodb/atlas-cli-plugin-kubernetes/internal/templatewriter"
)

const (
	jsonFormat     = "json"
	jsonPath       = "json-path"
	goTemplate     = "go-template"
	goTemplateFile = "go-template-file"
)

var templateFormats = []string{goTemplate, goTemplateFile, jsonPath}

type OutputOpts struct {
	Template  string
	OutWriter io.Writer
	Output    string
}

// ConfigOutput returns the output format.
// If the format is empty, it caches it after querying config.
func (opts *OutputOpts) ConfigOutput() string {
	if opts.Output != "" {
		return opts.Output
	}
	opts.Output = config.Output()
	return opts.Output
}

// ConfigWriter returns the io.Writer.
// If the writer is nil, it defaults to os.Stdout and caches it.
func (opts *OutputOpts) ConfigWriter() io.Writer {
	if opts.OutWriter != nil {
		return opts.OutWriter
	}
	opts.OutWriter = os.Stdout
	return opts.OutWriter
}

func isNil(o any) bool {
	if o == nil {
		return true
	}
	ot := reflect.TypeOf(o)
	otk := ot.Kind()
	switch otk { //nolint:exhaustive // clearer code
	case reflect.Array, reflect.Slice, reflect.Map, reflect.Chan, reflect.Pointer, reflect.UnsafePointer, reflect.Interface:
		return reflect.ValueOf(o).IsNil()
	default:
		return false
	}
}

func isOrPtrToSliceOrArray(o any) bool {
	ot := reflect.TypeOf(o)
	if ot == nil {
		return false
	}
	otk := ot.Kind()
	switch otk { //nolint:exhaustive // clearer code
	case reflect.Array, reflect.Slice:
		return true
	case reflect.Pointer:
		opt := reflect.PointerTo(ot)
		optk := opt.Kind()
		switch optk { //nolint:exhaustive // clearer code
		case reflect.Array, reflect.Slice:
			return true
		}
	}
	return false
}

// Print will evaluate the defined format and try to parse it accordingly outputting to the set writer.
func (opts *OutputOpts) Print(o any) error {
	if opts.ConfigOutput() == jsonFormat {
		return jsonwriter.Print(opts.ConfigWriter(), o)
	}

	outputType, val := opts.outputTypeAndValue()
	if outputType == jsonPath {
		return jsonpathwriter.Print(opts.ConfigWriter(), val, o)
	}

	t, err := template(outputType, val)
	if err != nil {
		return err
	}

	if t != "" {
		if isNil(o) {
			if isOrPtrToSliceOrArray(o) {
				o = []map[string]any{}
			} else {
				o = map[string]any{}
			}
		}
		return templatewriter.Print(opts.ConfigWriter(), t, o)
	}
	_, err = fmt.Fprintln(opts.ConfigWriter(), o)
	return err
}

// outputTypeAndValue returns the output type and the associated value
// Current available output types are:
// "go-template=Template string",
// "go-template-file=path/to/template",
// and "json-path=path".
func (opts *OutputOpts) outputTypeAndValue() (outputType, v string) {
	v = opts.Template
	for _, format := range templateFormats {
		format += "="
		if strings.HasPrefix(opts.ConfigOutput(), format) {
			v = opts.ConfigOutput()[len(format):]
			outputType = format[:len(format)-1]
			break
		}
	}

	return
}

var errTemplate = errors.New("error loading template")

// template returns the correct template from the output type.
func template(outputType, value string) (string, error) {
	if outputType == goTemplateFile {
		data, err := os.ReadFile(value)
		if err != nil {
			return "", fmt.Errorf("%w: %s, %w", errTemplate, value, err)
		}

		value = string(data)
	}
	return value, nil
}
