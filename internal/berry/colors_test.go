package berry

import "testing"

func TestColorProjectsGivesSharedAndDefaultColoursDistinctOnes(t *testing.T) {
	b := Board{Projects: []Project{
		{ID: "a", Color: "#6175ba"},
		{ID: "b", Color: "#123456"}, // picked on purpose
		{ID: undecidedProject, Color: "#6175ba"},
		{ID: "c", Color: "#6175BA"},
		{ID: "team", Color: "#0a84ff"}, // unique, so kept; the palette skips it
		{ID: "d", Color: "#7564d9"},
		{ID: "e", Color: "#abcdef"},
		{ID: "f", Color: "#abcdef"}, // shared with e: both reassigned
	}}
	b.colorProjects()
	want := map[string]string{"a": projectPalette[1], "b": "#123456", undecidedProject: "#8b8d98", "c": projectPalette[2],
		"team": "#0a84ff", "d": projectPalette[3], "e": projectPalette[4], "f": projectPalette[5]}
	for _, p := range b.Projects {
		if p.Color != want[p.ID] {
			t.Errorf("%s colour = %s, want %s", p.ID, p.Color, want[p.ID])
		}
	}
}
