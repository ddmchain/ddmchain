package check

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"testing"
	"time"
)

var allSuites []interface{}

func Suite(suite interface{}) interface{} {
	allSuites = append(allSuites, suite)
	return suite
}

var (
	oldFilterFlag  = flag.String("gocheck.f", "", "Regular expression selecting which tests and/or suites to run")
	oldVerboseFlag = flag.Bool("gocheck.v", false, "Verbose mode")
	oldStreamFlag  = flag.Bool("gocheck.vv", false, "Super verbose mode (disables output caching)")
	oldBenchFlag   = flag.Bool("gocheck.b", false, "Run benchmarks")
	oldBenchTime   = flag.Duration("gocheck.btime", 1*time.Second, "approximate run time for each benchmark")
	oldListFlag    = flag.Bool("gocheck.list", false, "List the names of all tests that will be run")
	oldWorkFlag    = flag.Bool("gocheck.work", false, "Display and do not remove the test working directory")

	newFilterFlag  = flag.String("check.f", "", "Regular expression selecting which tests and/or suites to run")
	newVerboseFlag = flag.Bool("check.v", false, "Verbose mode")
	newStreamFlag  = flag.Bool("check.vv", false, "Super verbose mode (disables output caching)")
	newBenchFlag   = flag.Bool("check.b", false, "Run benchmarks")
	newBenchTime   = flag.Duration("check.btime", 1*time.Second, "approximate run time for each benchmark")
	newBenchMem    = flag.Bool("check.bmem", false, "Report memory benchmarks")
	newListFlag    = flag.Bool("check.list", false, "List the names of all tests that will be run")
	newWorkFlag    = flag.Bool("check.work", false, "Display and do not remove the test working directory")
)

func TestingT(testingT *testing.T) {
	benchTime := *newBenchTime
	if benchTime == 1*time.Second {
		benchTime = *oldBenchTime
	}
	conf := &RunConf{
		Filter:        *oldFilterFlag + *newFilterFlag,
		Verbose:       *oldVerboseFlag || *newVerboseFlag,
		Stream:        *oldStreamFlag || *newStreamFlag,
		Benchmark:     *oldBenchFlag || *newBenchFlag,
		BenchmarkTime: benchTime,
		BenchmarkMem:  *newBenchMem,
		KeepWorkDir:   *oldWorkFlag || *newWorkFlag,
	}
	if *oldListFlag || *newListFlag {
		w := bufio.NewWriter(os.Stdout)
		for _, name := range ListAll(conf) {
			fmt.Fprintln(w, name)
		}
		w.Flush()
		return
	}
	result := RunAll(conf)
	println(result.String())
	if !result.Passed() {
		testingT.Fail()
	}
}

func RunAll(runConf *RunConf) *Result {
	result := Result{}
	for _, suite := range allSuites {
		result.Add(Run(suite, runConf))
	}
	return &result
}

func Run(suite interface{}, runConf *RunConf) *Result {
	runner := newSuiteRunner(suite, runConf)
	return runner.run()
}

func ListAll(runConf *RunConf) []string {
	var names []string
	for _, suite := range allSuites {
		names = append(names, List(suite, runConf)...)
	}
	return names
}

func List(suite interface{}, runConf *RunConf) []string {
	var names []string
	runner := newSuiteRunner(suite, runConf)
	for _, t := range runner.tests {
		names = append(names, t.String())
	}
	return names
}

func (r *Result) Add(other *Result) {
	r.Succeeded += other.Succeeded
	r.Skipped += other.Skipped
	r.Failed += other.Failed
	r.Panicked += other.Panicked
	r.FixturePanicked += other.FixturePanicked
	r.ExpectedFailures += other.ExpectedFailures
	r.Missed += other.Missed
	if r.WorkDir != "" && other.WorkDir != "" {
		r.WorkDir += ":" + other.WorkDir
	} else if other.WorkDir != "" {
		r.WorkDir = other.WorkDir
	}
}

func (r *Result) Passed() bool {
	return (r.Failed == 0 && r.Panicked == 0 &&
		r.FixturePanicked == 0 && r.Missed == 0 &&
		r.RunError == nil)
}

func (r *Result) String() string {
	if r.RunError != nil {
		return "ERROR: " + r.RunError.Error()
	}

	var value string
	if r.Failed == 0 && r.Panicked == 0 && r.FixturePanicked == 0 &&
		r.Missed == 0 {
		value = "OK: "
	} else {
		value = "OOPS: "
	}
	value += fmt.Sprintf("%d passed", r.Succeeded)
	if r.Skipped != 0 {
		value += fmt.Sprintf(", %d skipped", r.Skipped)
	}
	if r.ExpectedFailures != 0 {
		value += fmt.Sprintf(", %d expected failures", r.ExpectedFailures)
	}
	if r.Failed != 0 {
		value += fmt.Sprintf(", %d FAILED", r.Failed)
	}
	if r.Panicked != 0 {
		value += fmt.Sprintf(", %d PANICKED", r.Panicked)
	}
	if r.FixturePanicked != 0 {
		value += fmt.Sprintf(", %d FIXTURE-PANICKED", r.FixturePanicked)
	}
	if r.Missed != 0 {
		value += fmt.Sprintf(", %d MISSED", r.Missed)
	}
	if r.WorkDir != "" {
		value += "\nWORK=" + r.WorkDir
	}
	return value
}
