package pg

import (
	"fmt"
	"strings"
)

// Dsn holds PostgreSQL connection parameters.
type Dsn map[string]string

// String returns the connection parameters as a string, with the password masked.
func (d Dsn) String(masked bool) string {
	var parts []string
	for k, v := range d {
		if k == "password" {
			v = "*****"
		}
		parts = append(parts, fmt.Sprintf("%s=\"%s\"", k, strings.ReplaceAll(v, "\"", "\"\"")))
	}
	return strings.Join(parts, " ")
}
