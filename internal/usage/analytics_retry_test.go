package usage

import (
	"testing"
	"time"

	"github.com/yetone/magpie/internal/catalog"
)

func TestAnalyticsHandedOnAttemptPreservesBillingWithoutFailure(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	records := []Record{
		{Time: now.Add(-time.Minute), RouteID: 1, Agent: "codex", Provider: "p", Model: "m", Status: 429, HandedOn: true, Input: 100},
		{Time: now, RouteID: 1, Agent: "codex", Provider: "p", Model: "m", Status: 200, Tries: 2, Input: 200, Output: 50, TTFT: 200, Millis: 1200},
	}
	lookup := mockPriceLookup(map[string]*catalog.Price{"p/m": {Input: 10, Output: 20}})
	data := analyzeWith(Today, AnalyticsFilter{}, now, records, lookup)
	check := func(name string, summary AnalyticsSummary) {
		t.Helper()
		if summary.Calls != 1 || summary.Errors != 0 || summary.RateLimited != 0 || summary.ServerErr != 0 || summary.OtherErr != 0 || summary.Canceled != 0 {
			t.Fatalf("%s counts a handed-on attempt as a request failure: %+v", name, summary)
		}
		if summary.SuccessRate == nil || *summary.SuccessRate != 1 || summary.ErrorRate == nil || *summary.ErrorRate != 0 {
			t.Fatalf("%s rates must describe the successful final outcome: success=%v error=%v", name, summary.SuccessRate, summary.ErrorRate)
		}
		if summary.Input != 300 || summary.Output != 50 || summary.Cost < 0.003999 || summary.Cost > 0.004001 {
			t.Fatalf("%s lost billing from a handed-on attempt: input=%d output=%d cost=%v", name, summary.Input, summary.Output, summary.Cost)
		}
	}
	check("overall", data.Summary)
	for dimension, ranking := range data.Rankings {
		for entity, summary := range ranking.Summaries {
			check(dimension+"/"+entity, summary)
		}
	}
	for _, point := range data.ErrorTrend {
		if point.RateLimited != 0 || point.ServerErr != 0 || point.OtherErr != 0 {
			t.Fatalf("error trend counts a handed-on attempt: %+v", point)
		}
	}
	calls, err := recentCallsWith(Today, AnalyticsFilter{}, "error_rate", 50, now, records, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) != 0 {
		t.Fatalf("failed-request drilldown includes a handed-on attempt: %+v", calls)
	}
	costCalls, err := recentCallsWith(Today, AnalyticsFilter{}, "cost", 50, now, records, lookup)
	if err != nil {
		t.Fatal(err)
	}
	if len(costCalls) != 2 {
		t.Fatalf("cost drilldown lost a billed attempt: %+v", costCalls)
	}
}
