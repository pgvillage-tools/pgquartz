package jobs

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

type stepState int

const (
	stepStateWaiting stepState = iota
	stepStateSkipped
	stepStateReady
	stepStateScheduled
	stepStateRunning
	stepStateDone
	stepStateUnknown
)

var (
	stepStateStrings = map[stepState]string{
		stepStateWaiting:   "Waiting",
		stepStateSkipped:   "Skipped",
		stepStateReady:     "Ready",
		stepStateScheduled: "Scheduled",
		stepStateRunning:   "Running",
		stepStateDone:      "Done",
		stepStateUnknown:   "Unknown",
	}
)

func (sst stepState) String() string {
	stepStateString, exists := stepStateStrings[sst]
	if !exists {
		log.Panicf("Unknown stepStateStrings")
	}
	return stepStateString
}

// Steps maps step names to steps.
type Steps map[string]*Step

// InstanceFinished marks the step instance referenced by w as done.
func (ss *Steps) InstanceFinished(w Work) {
	s, sExists := (*ss)[w.Step]
	if !sExists {
		log.Panicf("Unknown step %s is finished", w.Step)
	}
	i, iExists := s.Instances[w.ArgKey]
	if !iExists {
		log.Panicf("Unknown instance %s for step %s is finished", w.ArgKey, w.Step)
	}
	i.done = true
	log.Debugf("Set instance [%s].[%s] to done", w.Step, w.ArgKey)
}

// Verify checks all steps and their dependencies and returns the problems found.
func (ss Steps) Verify(conns Connections) (errs []error) {
	for stepName, step := range ss {
		errs = append(errs, step.Commands.Verify(stepName, conns)...)
		for _, dependency := range step.Depends {
			if _, exists := ss[dependency]; !exists {
				errs = append(errs, fmt.Errorf("step %s depends on unknown step %s", stepName, dependency))
			}
		}
	}
	return errs
}

// Initialize sets up the instances of all steps.
func (ss *Steps) Initialize() {
	for _, step := range *ss {
		step.Initialize()
	}
}

// GetNumInstances returns the total number of instances of all steps.
func (ss Steps) GetNumInstances() int {
	var count int
	for name, step := range ss {
		step.SetInstances()
		num := len(step.GetInstances())
		log.Debugf("step %s has %d instances", name, num)
		count += num
	}
	log.Debugf("counting %d instances", count)
	return count
}

// Clone returns an initialized copy of all steps.
func (ss Steps) Clone() Steps {
	clone := make(Steps)
	for name, step := range ss {
		newStep := step.Clone()
		newStep.Initialize()
		clone[name] = newStep
	}
	return clone
}

func (ss Steps) setStepState(stepName string, newState stepState) {
	step, exists := ss[stepName]
	if !exists {
		log.Panicf("Looking for a step %s that does not exist???", stepName)
	}
	if err := step.setState(newState); err != nil {
		log.Panicf("Error while changing state for step %s: %e", stepName, err)
	}
}

// GetReadySteps returns the names of the steps that are ready, or waiting with all dependencies done or skipped.
func (ss Steps) GetReadySteps() (ready []string) {
	var isReady bool
	for stepName, step := range ss {
		if step.Ready() {
			ready = append(ready, stepName)
		}
		if !step.Waiting() {
			continue
		}
		isReady = true
		for _, dependency := range step.Depends {
			if subStep, exists := ss[dependency]; !exists {
				log.Panicf("step %s depends on unknows step %s", stepName, dependency)
			} else if !subStep.Done() && subStep.state != stepStateSkipped {
				isReady = false
				break
			}
		}
		if isReady {
			ready = append(ready, stepName)
		}
	}
	return ready
}

// NumWaiting returns the number of steps that are waiting.
func (ss Steps) NumWaiting() (numWaiting int) {
	for _, step := range ss {
		if !step.Waiting() {
			continue
		}
		numWaiting++
	}
	return numWaiting
}

// CheckWhen evaluates the 'when' templates of a step and reports whether all of them return True.
func (ss Steps) CheckWhen(all Handler, stepName string) (bool, error) {
	step, exists := ss[stepName]
	if !exists {
		return false, fmt.Errorf("checking a 'when' on an undefined step %s", stepName)
	}
	numChecks := len(step.When)
	for _, whenCheck := range step.When {
		if !strings.Contains(whenCheck, "{{") || !strings.Contains(whenCheck, "}}") {
			whenCheck = fmt.Sprintf("{{if %s }}True{{end}}", whenCheck)
		}
		t, err := template.New("when").Parse(whenCheck)
		if err != nil {
			return false, err
		}
		log.Debugf("Processing WhenCheck '%s' for step %s", whenCheck, stepName)
		var parsed bytes.Buffer
		err = t.Execute(&parsed, all)
		log.Debugf("WhenCheck '%s' returned %s for step %s", whenCheck, parsed.String(), stepName)
		if err != nil {
			return false, err
		} else if parsed.String() != "True" {
			return false, nil
		}
	}
	log.Debugf("All %d WhenChecks for step %s are OK", numChecks, stepName)
	return true, nil
}

// Step is a set of commands that is run for every combination of its matrix arguments.
type Step struct {
	Commands  Commands `yaml:"commands"`
	Depends   []string `yaml:"depends,omitempty"`
	state     stepState
	When      []string   `yaml:"when,omitempty"`
	Matrix    MatrixArgs `yaml:"matrix,omitempty"`
	Instances Instances  `yaml:"-"`
}

// Waiting reports whether the step is waiting to be scheduled.
func (s Step) Waiting() bool {
	return s.state == stepStateWaiting
}

// Ready reports whether the step is ready to be scheduled.
func (s Step) Ready() bool {
	return s.state == stepStateReady
}

// Done reports whether the step is done, and marks it done when all its instances are done.
func (s *Step) Done() bool {
	if s.state == stepStateDone {
		return true
	}
	if s.Instances.Done() {
		if err := s.setState(stepStateDone); err != nil {
			log.Panicf("Could not issue done state for step %e", err)
		}
		return true
	}
	return false
}

// InstanceFinished marks an instance as done and reports whether the step is done.
func (s *Step) InstanceFinished(instance string) bool {
	if s.Done() {
		log.Fatalf("calling instanceFinished on a step that is already finished")
	}
	i, exists := s.Instances[instance]
	if !exists {
		log.Fatalf("calling instanceFinished on an instance that does not exist")
	}
	log.Debugf("instance done: %s", i.Name())
	i.done = true
	return s.Done()
}

func (s *Step) setState(newState stepState) error {
	if s.state > newState {
		return fmt.Errorf("invalid step transition from %s to %s", s.state.String(), newState.String())
	}
	s.state = newState
	return nil
}

// Clone returns a copy of the step in the waiting state.
func (s Step) Clone() *Step {
	return &Step{
		Commands:  s.Commands.Clone(),
		Instances: s.Instances.Clone(),
		Depends:   s.Depends,
		state:     stepStateWaiting,
		When:      s.When,
		Matrix:    s.Matrix,
	}
}

// StdOut returns the combined stdout of all instances of the step.
func (s Step) StdOut() Result {
	return s.Instances.StdOut()
}

// StdErr returns the combined stderr of all instances of the step.
func (s Step) StdErr() Result {
	return s.Instances.StdErr()
}

// Rc returns the sum of the return codes of all instances of the step.
func (s Step) Rc() int {
	return s.Instances.Rc()
}

// Initialize sets up the instances of the step.
func (s *Step) Initialize() {
	s.SetInstances()
}

// SetInstances creates an instance for every combination of matrix arguments, unless instances already exist.
func (s *Step) SetInstances() {
	if len(s.Instances) > 0 {
		return
	}
	s.Instances = make(Instances)
	for _, args := range s.Matrix.Instances() {
		s.Instances[args.String()] = NewInstance(args, s.Commands.Clone())
	}
}

// GetInstances returns the instances of the step.
func (s Step) GetInstances() Instances {
	// log.Debugf("Instances: %s", s.Instances.String())
	return s.Instances
}
