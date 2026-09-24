package jobs

// Target defines on which database role and how often a job runs.
type Target struct {
	Role         string `yaml:"role"`
	Distribution string `yaml:"distribution"`
	Repeat       int    `yaml:"repeat"`
	Delay        int    `yaml:"delay"`
}
