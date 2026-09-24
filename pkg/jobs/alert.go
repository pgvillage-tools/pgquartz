package jobs

// Alert holds info on a PgQuartz alert
type Alert struct {
	AlertType string `yaml:"type"`
	Command   string `yaml:"command"`
}

// Alerts is a list of Alert objects
type Alerts []Alert
