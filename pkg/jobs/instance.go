package jobs

// Instances maps instance names to step instances.
type Instances map[string]*Instance

// Clone returns a copy of all instances.
func (is Instances) Clone() (clone Instances) {
	clone = make(Instances)
	for _, i := range is {
		clone[i.name] = i.Clone()
	}
	return clone
}

// Done reports whether all instances are done.
func (is Instances) Done() bool {
	for _, i := range is {
		if !i.done {
			return false
		}
	}
	return true
}

// StdOut returns the combined stdout of all instances.
func (is Instances) StdOut() (r Result) {
	for _, i := range is {
		r = append(r, i.StdOut()...)
	}
	return r
}

// StdErr returns the combined stderr of all instances.
func (is Instances) StdErr() (r Result) {
	for _, i := range is {
		r = append(r, i.StdErr()...)
	}
	return r
}

// Rc returns the sum of the return codes of all instances.
func (is Instances) Rc() (rc int) {
	for _, instance := range is {
		rc += instance.commands.Rc()
	}
	return rc
}

// Instance is one run of the commands of a step, with one set of matrix arguments.
type Instance struct {
	name     string
	args     InstanceArguments
	commands Commands
	done     bool
}

// NewInstance returns a new Instance for the given arguments and commands.
func NewInstance(args InstanceArguments, commands Commands) *Instance {
	return &Instance{
		args:     args,
		name:     args.String(),
		commands: commands,
	}
}

// Clone returns a copy of the instance.
func (i Instance) Clone() (clone *Instance) {
	return &Instance{
		args:     i.args.Clone(),
		commands: i.commands.Clone(),
	}
}

// StdOut returns the stdout of the commands of this instance.
func (i Instance) StdOut() (r Result) {
	return i.commands.StdOut()
}

// StdErr returns the stderr of the commands of this instance.
func (i Instance) StdErr() (r Result) {
	return i.commands.StdErr()
}

// Name returns the name of the instance, which is derived from its arguments.
func (i Instance) Name() string {
	return i.name
}
