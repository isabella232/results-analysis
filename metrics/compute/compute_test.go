// Copyright 2017 The WPT Dashboard Project. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be
// found in the LICENSE file.

package compute

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/web-platform-tests/results-analysis/metrics"
	"github.com/web-platform-tests/wpt.fyi/shared"
)

var timeA = time.Unix(0, 0)
var timeB = time.Unix(0, 1)
var runA = metrics.TestRunLegacy{
	ProductAtRevision: shared.ProductAtRevision{
		Product: shared.Product{
			BrowserName:    "ABrowser",
			BrowserVersion: "1.0",
			OSName:         "MyOS",
			OSVersion:      "1.0",
		},
		Revision: "abcd",
	},
	ResultsURL: "http://example.com/a_run.json",
	CreatedAt:  timeA,
}
var runB = metrics.TestRunLegacy{
	ProductAtRevision: shared.ProductAtRevision{
		Product: shared.Product{
			BrowserName:    "BBrowser",
			BrowserVersion: "1.0",
			OSName:         "dcba",
			OSVersion:      "1.0",
		},
		Revision: "dcba",
	},
	ResultsURL: "http://example.com/b_run.json",
	CreatedAt:  timeB,
}

func TestGatherResultsById_TwoRuns_SameTest(t *testing.T) {
	testName := "Do a thing"
	results := &[]metrics.TestRunResults{
		{
			&runA,
			&metrics.TestResults{
				"A test",
				"OK",
				&testName,
				[]metrics.SubTest{},
			},
		},
		{
			&runB,
			&metrics.TestResults{
				"A test",
				"ERROR",
				&testName,
				[]metrics.SubTest{},
			},
		},
	}

	gathered := GatherResultsById(context.Background(), results)
	assert.Equal(t, 1, len(gathered)) // Merged to single TestId: {"A test",""}.
	for testID, runStatusMap := range gathered {
		assert.Equal(t, metrics.TestID{"A test", ""}, testID)
		assert.Equal(t, 2, len(runStatusMap))
		assert.Equal(t, metrics.CompleteTestStatus{
			metrics.TestStatusFromString("OK"),
			metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
		}, runStatusMap[runA.BrowserName])
		assert.Equal(t, metrics.CompleteTestStatus{
			metrics.TestStatusFromString("ERROR"),
			metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
		}, runStatusMap[runB.BrowserName])
	}
}

func TestGatherResultsById_TwoRuns_DiffTests(t *testing.T) {
	testName := "Do a thing"
	results := &[]metrics.TestRunResults{
		{
			&runA,
			&metrics.TestResults{
				"A test",
				"OK",
				&testName,
				[]metrics.SubTest{},
			},
		},
		{
			&runA,
			&metrics.TestResults{
				"Shared test",
				"ERROR",
				&testName,
				[]metrics.SubTest{},
			},
		},
		{
			&runB,
			&metrics.TestResults{
				"Shared test",
				"OK",
				&testName,
				[]metrics.SubTest{},
			},
		},
		{
			&runB,
			&metrics.TestResults{
				"B test",
				"ERROR",
				&testName,
				[]metrics.SubTest{},
			},
		},
	}
	gathered := GatherResultsById(context.Background(), results)
	assert.Equal(t, 3, len(gathered)) // A, Shared, B.
	assert.Equal(t, 1, len(gathered[metrics.TestID{"A test", ""}]))
	assert.Equal(t, metrics.CompleteTestStatus{
		metrics.TestStatusFromString("OK"),
		metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
	}, gathered[metrics.TestID{"A test", ""}][runA.BrowserName])
	assert.Equal(t, 2, len(gathered[metrics.TestID{"Shared test", ""}]))
	assert.Equal(t, metrics.CompleteTestStatus{
		metrics.TestStatusFromString("ERROR"),
		metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
	}, gathered[metrics.TestID{"Shared test", ""}][runA.BrowserName])
	assert.Equal(t, metrics.CompleteTestStatus{
		metrics.TestStatusFromString("OK"),
		metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
	}, gathered[metrics.TestID{"Shared test", ""}][runB.BrowserName])
	assert.Equal(t, 1, len(gathered[metrics.TestID{"B test", ""}]))
	assert.Equal(t, metrics.CompleteTestStatus{
		metrics.TestStatusFromString("ERROR"),
		metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
	}, gathered[metrics.TestID{"B test", ""}][runB.BrowserName])
}

func TestGatherResultsById_OneRun_SubTest(t *testing.T) {
	testName := "Do a thing"
	subName1 := "First sub-test"
	subName2 := "Second sub-test"
	subStatus1 := "A-OK!"
	subStatus2 := "Oops..."
	results := &[]metrics.TestRunResults{
		{
			&runA,
			&metrics.TestResults{
				"A test",
				"OK",
				&testName,
				[]metrics.SubTest{
					{
						subName1,
						"PASS",
						&subStatus1,
					},
					{
						subName2,
						"FAIL",
						&subStatus2,
					},
				},
			},
		},
	}
	gathered := GatherResultsById(context.Background(), results)
	assert.Equal(t, 3, len(gathered)) // Top-level test + 2 sub-tests.
	testIds := make([]metrics.TestID, 0, len(gathered))
	for testId, _ := range gathered {
		testIds = append(testIds, testId)
	}
	assert.ElementsMatch(t, [...]metrics.TestID{
		{"A test", ""},
		{"A test", subName1},
		{"A test", subName2},
	}, testIds)
	assert.Equal(t, metrics.CompleteTestStatus{
		metrics.TestStatusFromString("OK"),
		metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
	}, gathered[metrics.TestID{"A test", ""}][runA.BrowserName])
	assert.Equal(t, metrics.CompleteTestStatus{
		metrics.TestStatusFromString("OK"),
		metrics.SubTestStatusFromString("PASS"),
	}, gathered[metrics.TestID{"A test", subName1}][runA.BrowserName])
	assert.Equal(t, metrics.CompleteTestStatus{
		metrics.TestStatusFromString("OK"),
		metrics.SubTestStatusFromString("FAIL"),
	}, gathered[metrics.TestID{"A test", subName2}][runA.BrowserName])
}

func getPrecomputedStatusz() *TestRunsStatus {
	statusz := make(TestRunsStatus)
	status1 := metrics.CompleteTestStatus{
		metrics.TestStatusFromString("OK"),
		metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
	}
	status2 := metrics.CompleteTestStatus{
		metrics.TestStatusFromString("ERROR"),
		metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
	}
	status3 := metrics.CompleteTestStatus{
		metrics.TestStatusFromString("PASS"),
		metrics.SubTestStatusFromString("STATUS_UNKNOWN"),
	}
	subStatus1 := metrics.CompleteTestStatus{
		metrics.TestStatusFromString("OK"),
		metrics.SubTestStatusFromString("PASS"),
	}
	subStatus2 := metrics.CompleteTestStatus{
		metrics.TestStatusFromString("OK"),
		metrics.SubTestStatusFromString("NOT_RUN"),
	}
	ab1 := metrics.TestID{"a/b/1", ""}
	ab2 := metrics.TestID{"a/b/2", ""}
	ac1 := metrics.TestID{"a/c/1", ""}
	ac1x := metrics.TestID{"a/c/1", "x"}
	ac1y := metrics.TestID{"a/c/1", "y"}
	ac1z := metrics.TestID{"a/c/1", "z"}
	statusz[ab1] = make(map[string]metrics.CompleteTestStatus)
	statusz[ab2] = make(map[string]metrics.CompleteTestStatus)
	statusz[ac1] = make(map[string]metrics.CompleteTestStatus)
	statusz[ac1x] = make(map[string]metrics.CompleteTestStatus)
	statusz[ac1y] = make(map[string]metrics.CompleteTestStatus)
	statusz[ac1z] = make(map[string]metrics.CompleteTestStatus)
	statusz[ab1][runA.BrowserName] = status1
	statusz[ab1][runB.BrowserName] = status2
	statusz[ab2][runB.BrowserName] = status3
	statusz[ac1][runA.BrowserName] = status1
	statusz[ac1x][runA.BrowserName] = subStatus1
	statusz[ac1y][runA.BrowserName] = subStatus2
	statusz[ac1z][runA.BrowserName] = subStatus2

	return &statusz
}

func TestComputeTotals(t *testing.T) {
	statusz := getPrecomputedStatusz()

	totals := ComputeTotals(statusz)
	assert.Equal(t, 6, len(totals))   // a, a/b, a/c, a/b/1, a/b/2, a/c/1.
	assert.Equal(t, 6, totals["a"])   // a/b/1, a/b/2, a/c/1, a/c/1:x, a/c/1:y, a/c/1:z.
	assert.Equal(t, 2, totals["a/b"]) // a/b/1, a/b/2.
	assert.Equal(t, 1, totals["a/b/1"])
	assert.Equal(t, 1, totals["a/b/2"])
	assert.Equal(t, 4, totals["a/c"])   // a/c/1, a/c/1:x, a/c/1:y, a/c/1:z.
	assert.Equal(t, 4, totals["a/c/1"]) // a/c/1, a/c/1:x, a/c/1:y, a/c/1:z.
}

func TestComputePassRateMetric(t *testing.T) {
	statusz := getPrecomputedStatusz()

	noTopLevelPasses := ComputePassRateMetric(2, statusz, OkAndUnknownOrPasses)
	topLevelPasses := ComputePassRateMetric(2, statusz, OkOrPassesAndUnknownOrPasses)

	assert.Equal(t, 6, len(noTopLevelPasses))
	assert.Equal(t, 6, len(topLevelPasses))

	// a/b/1: runA=OK, runB=ERROR.
	assert.Equal(t, []int{0, 1, 0}, noTopLevelPasses["a/b/1"])
	assert.Equal(t, []int{0, 1, 0}, topLevelPasses["a/b/1"])

	// a/c/1: runA=OK.
	assert.Equal(t, []int{2, 2, 0}, noTopLevelPasses["a/c/1"])
	assert.Equal(t, []int{2, 2, 0}, topLevelPasses["a/c/1"])

	// a/b/2: runB=PASS (and not OK): leads to [1, 0, 0].
	assert.Equal(t, []int{1, 0, 0}, noTopLevelPasses["a/b/2"])
	// a/b/2: runB=PASS acceptable; leads to [0, 1, 0].
	assert.Equal(t, []int{0, 1, 0}, topLevelPasses["a/b/2"])
}
