package run

import (
	"time"

	"github.com/form3tech-oss/f1/v2/internal/metrics"
	"github.com/form3tech-oss/f1/v2/internal/progress"
	"github.com/form3tech-oss/f1/v2/pkg/f1/scenarios"
	"github.com/form3tech-oss/f1/v2/pkg/f1/testing"
)

type iterationState struct {
	teardown func()
	t        *testing.T
}

func newIterationState(scenario string) *iterationState {
	state := &iterationState{}
	state.t, state.teardown = testing.NewT("", scenario)

	return state
}

type ActiveScenario struct {
	scenario *scenarios.Scenario
	m        *metrics.Metrics
	progress *progress.Stats
	t        *testing.T
	Teardown func()
}

func NewActiveScenario(
	scenario *scenarios.Scenario,
	metricsInstance *metrics.Metrics,
	stats *progress.Stats,
) *ActiveScenario {
	t, teardown := testing.NewT("setup", scenario.Name)

	s := &ActiveScenario{
		scenario: scenario,
		m:        metricsInstance,
		t:        t,
		Teardown: teardown,
		progress: stats,
	}

	start := time.Now()
	func() {
		defer testing.CheckResults(t, nil)

		s.scenario.RunFn = s.scenario.ScenarioFn(t)
	}()

	// wait for completion
	s.m.RecordSetupResult(scenario.Name, metrics.Result(t.Failed()), time.Since(start).Nanoseconds())
	return s
}

// Run performs a single iteration of the test. It returns `true` if the test was successful, `false` otherwise.
func (s *ActiveScenario) Run(state *iterationState) bool {
	defer state.teardown()

	start := time.Now()
	func() {
		defer testing.CheckResults(state.t, nil)
		s.scenario.RunFn(state.t)
	}()

	failed := state.t.Failed()
	duration := time.Since(start)

	s.m.RecordIterationResult(s.scenario.Name, metrics.Result(failed), duration.Nanoseconds())
	s.progress.Record(metrics.Result(failed), duration)
	return !failed
}

func (s *ActiveScenario) RecordDroppedIteration() {
	s.m.RecordIterationResult(s.scenario.Name, metrics.DroppedResult, 0)
	s.progress.Record(metrics.DroppedResult, 0)
}
