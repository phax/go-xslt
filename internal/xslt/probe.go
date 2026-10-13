package xslt

// stepProbe, when set, is called every stepProbeEvery instructions a run
// executes. It exists for tests that measure memory: a reading taken at a fixed
// point of the PROGRAM (after N instructions, with a collection forced just
// before it) is the live heap at that point and does not depend on how fast the
// machine is or on what else is running — unlike sampling the heap on a timer,
// which measures the collector's lag as much as the engine. Nil in production.
var (
	stepProbe      func()
	stepProbeEvery int64
)
