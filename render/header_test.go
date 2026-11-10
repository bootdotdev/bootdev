package render

import (
	"strings"
	"testing"
)

func TestLessonHeaderPersistsInFinalReport(t *testing.T) {
	m := initModel(true, false)
	m.runContext = RunContext{Action: "Submitting lesson:", LessonTitle: "9.3 - Sorting Chirps", BaseURL: "http://localhost:8080"}
	m.steps = []stepModel{{description: "Reset database"}}
	for _, finalized := range []bool{false, true} {
		m.finalized = finalized
		view := m.View()
		for _, expected := range []string{"Submitting lesson:\n9.3 - Sorting Chirps", "Target: http://localhost:8080"} {
			if strings.Count(view, expected) != 1 {
				t.Fatalf("expected one %q in:\n%s", expected, view)
			}
		}
		if strings.Index(view, "Submitting lesson:") > strings.Index(view, "Reset database") {
			t.Fatal("header follows steps")
		}
	}
}

func TestHeaderWithoutTargetShowsOnlyLesson(t *testing.T) {
	view := renderRunHeader(RunContext{LessonTitle: "1.3 - Example"})
	if view != "Running lesson:\n1.3 - Example\n\n" {
		t.Fatalf("unexpected header: %s", view)
	}
}
