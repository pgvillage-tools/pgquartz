package pg

import (
	"fmt"
	"strings"
)

// Row holds the values of one result row.
type Row []string

// Result holds the header and rows returned by a query.
type Result struct {
	header Row
	rows   []Row
}

// AsMapArray returns every row as a map from column name to value.
func (r Result) AsMapArray() (arraysOfMaps []map[string]string) {
	for _, row := range r.rows {
		m := make(map[string]string)
		for i, c := range row {
			m[r.header[i]] = c
		}
		arraysOfMaps = append(arraysOfMaps, m)
	}
	return arraysOfMaps
}

// AsStringArray returns every row as a string of {column}={value} pairs.
// The optional parameter sets the delimiter, which defaults to ", ".
func (r Result) AsStringArray(params ...string) (arraysOfStrings []string) {
	var delimiter = ", "
	if len(params) > 0 {
		delimiter = params[0]
	}
	for _, row := range r.rows {
		var cols []string
		for i, col := range row {
			cols = append(cols, fmt.Sprintf("{%s}={%s}", r.header[i],
				strings.ReplaceAll(col, "'", "''")))
		}
		arraysOfStrings = append(arraysOfStrings, strings.Join(cols, delimiter))
	}
	return arraysOfStrings
}
