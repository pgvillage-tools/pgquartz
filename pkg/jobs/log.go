package jobs

// Log defines where the output of a job is logged.
type Log struct {
	LogType string `yaml:"type"`
	Command string `yaml:"command"`
}

// Logs is a list of Log definitions.
type Logs []Log
