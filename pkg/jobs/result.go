package jobs

import (
	"fmt"
	"regexp"
	"strings"
)

// ResultLine is one line of output.
type ResultLine string

// Result holds the lines of output of a command or query.
type Result []ResultLine

// NewResult returns a Result with the given lines.
func NewResult(lines []string) (result Result) {
	for _, line := range lines {
		result = append(result, ResultLine(line))
	}
	return result
}

// NewResultFromString splits a string on newlines and returns it as a Result.
func NewResultFromString(lines string) (result Result) {
	lines = strings.TrimSuffix(lines, "\n")
	for _, line := range strings.Split(lines, "\n") {
		result = append(result, ResultLine(line))
	}
	return result
}

// Append returns the result with the additional lines appended.
func (r Result) Append(additional Result) Result {
	return append(r, additional...)
}

func (r Result) String() string {
	var lines []string
	for _, line := range r {
		lines = append(lines, fmt.Sprintf("'%s'", strings.Replace(string(line), "'", "''", -1)))
	}
	return fmt.Sprintf("[ %s ]", strings.Join(lines, ", "))
}

// Contains reports whether any line contains part.
func (r Result) Contains(part string) bool {
	for _, l := range r {
		if l.Contains(part) {
			if debug() {
				log.Debugf("%s contains %s", r.String(), part)
			}
			return true
		}
	}
	if debug() {
		log.Debugf("%s does not contain %s", r.String(), part)
	}
	return false
}

// ContainsLine reports whether any line is equal to line.
func (r Result) ContainsLine(line string) bool {
	for _, l := range r {
		if string(l) == line {
			if debug() {
				log.Debugf("%s contains line %s", r.String(), line)
			}
			return true
		}
	}
	if debug() {
		log.Debugf("%s does not contain line %s", r.String(), line)
	}
	return false
}

// RegExpContains reports whether any line matches the regular expression exp.
func (r Result) RegExpContains(exp string) bool {
	re, err := regexp.Compile(exp)
	if err != nil {
		log.Errorf("Could not compile Regular Expression %s: %e", exp, err)
		return false
	}
	for _, l := range r {
		if re.Match([]byte(l)) {
			if debug() {
				log.Debugf("%s contains regexp %s", r.String(), exp)
			}
			return true
		}
	}
	if debug() {
		log.Debugf("%s does not contain regexp %s", r.String(), exp)
	}
	return false
}

// Contains reports whether the line contains part.
func (r ResultLine) Contains(part string) bool {
	return strings.Contains(string(r), part)
}
