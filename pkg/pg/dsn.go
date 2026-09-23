package pg

import (
	"fmt"
	"strings"
)

type Dsn map[string]string

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
