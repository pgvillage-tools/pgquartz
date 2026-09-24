package jobs

// Runners is a list of Runner pointers.
type Runners []*Runner

// Runner runs the step instances it receives from its Handler.
type Runner struct {
	index  int
	config Config
	Steps  Steps
	parent *Handler
	done   bool
}

// Done reports whether all runners are done.
func (rs Runners) Done() bool {
	for _, r := range rs {
		if !r.done {
			return false
		}
	}
	return true
}

// NewRunner returns a new Runner for the given Handler.
func NewRunner(h *Handler, index int) *Runner {
	return &Runner{
		index:  index,
		parent: h,
		config: h.Config,
	}
}

// Run runs work from the ToDo channel of the Handler until it is closed, and reports it on the Done channel.
func (r *Runner) Run() {
	for work := range r.parent.ToDo {
		step, sExists := r.parent.Steps[work.Step]
		if !sExists {
			log.Panicf("Runner %d: Trying to run a step %s that does not exist?", r.index, work.Step)
		}
		instance, iExists := step.Instances[work.ArgKey]
		if !iExists {
			log.Panicf("Runner %d: Trying to run an instance [%s].[%s] that does not exist?",
				r.index, work.Step, work.ArgKey)
		}
		log.Debugf("Runner %d: Running step [%s].[%s]", r.index, work.Step, work.ArgKey)
		if err := instance.commands.Run(r.config.Conns, instance.args); err != nil {
			log.Errorf("Runner %d: Error occurred while running step instance [%s].[%s]: %e",
				r.index, work.Step, work.ArgKey, err)
		}
		r.parent.Done <- work
	}
	log.Debugf("Runner %d: Done", r.index)
	r.done = true
}
