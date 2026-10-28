package artists

import (
	"reflect"
	"testing"
)

func TestGroupSelectionsByForm(t *testing.T) {
	type entry struct {
		Label     string
		WorkCount int
	}
	type form struct {
		Name      string
		WorkTotal int
		Entries   []entry
	}

	cases := []struct {
		name   string
		titles []selectionTitle
		want   []form
	}{
		{
			name:   "bracketed qualifier",
			titles: []selectionTitle{{Title: "Drawings (1484-1499)", WorkCount: 16}},
			want:   []form{{Name: "Drawings", WorkTotal: 16, Entries: []entry{{"1484–1499", 16}}}},
		},
		{
			name:   "colon qualifier",
			titles: []selectionTitle{{Title: "Synthetic: Paintings", WorkCount: 3}},
			want:   []form{{Name: "Synthetic", WorkTotal: 3, Entries: []entry{{"Paintings", 3}}}},
		},
		{
			name: "trailing counts after middle dot, dash and comma",
			titles: []selectionTitle{
				{Title: "Drawings (1484-1499) · 84", WorkCount: 1},
				{Title: "Woodcuts (1489-1500) - 12", WorkCount: 2},
				{Title: "Engravings (1494-1499) — 7", WorkCount: 3},
				{Title: "Etchings (1515), 9", WorkCount: 4},
				{Title: "Drypoints (1512) · 1,234", WorkCount: 5},
				{Title: "Watercolours (1489-1500), 2,345 works", WorkCount: 6},
			},
			want: []form{
				{Name: "Drawings", WorkTotal: 1, Entries: []entry{{"1484–1499", 1}}},
				{Name: "Woodcuts", WorkTotal: 2, Entries: []entry{{"1489–1500", 2}}},
				{Name: "Engravings", WorkTotal: 3, Entries: []entry{{"1494–1499", 3}}},
				{Name: "Etchings", WorkTotal: 4, Entries: []entry{{"1515", 4}}},
				{Name: "Drypoints", WorkTotal: 5, Entries: []entry{{"1512", 5}}},
				{Name: "Watercolours", WorkTotal: 6, Entries: []entry{{"1489–1500", 6}}},
			},
		},
		{
			name: "years and unspaced ranges are not counts",
			titles: []selectionTitle{
				{Title: "Saint Ursula Shrine, 1489", WorkCount: 8},
				{Title: "St Luke Madonna, 1435-40", WorkCount: 2},
				{Title: "Paintings between 1827-1830", WorkCount: 5},
			},
			want: []form{
				{Name: "Saint Ursula Shrine, 1489", WorkTotal: 8, Entries: []entry{{"Saint Ursula Shrine, 1489", 8}}},
				{Name: "St Luke Madonna, 1435-40", WorkTotal: 2, Entries: []entry{{"St Luke Madonna, 1435-40", 2}}},
				{Name: "Paintings between 1827-1830", WorkTotal: 5, Entries: []entry{{"Paintings between 1827-1830", 5}}},
			},
		},
		{
			name: "case-insensitive grouping in first-appearance order",
			titles: []selectionTitle{
				{Title: "Drawings (1484-1499)", WorkCount: 16},
				{Title: "Woodcuts (1489-1500)", WorkCount: 4},
				{Title: "drawings (1500-1505)", WorkCount: 11},
			},
			want: []form{
				{Name: "Drawings", WorkTotal: 27, Entries: []entry{{"1484–1499", 16}, {"1500–1505", 11}}},
				{Name: "Woodcuts", WorkTotal: 4, Entries: []entry{{"1489–1500", 4}}},
			},
		},
		{
			name:   "title that does not parse",
			titles: []selectionTitle{{Title: "The Navicella mosaic", WorkCount: 3}},
			want:   []form{{Name: "The Navicella mosaic", WorkTotal: 3, Entries: []entry{{"The Navicella mosaic", 3}}}},
		},
		{
			name:   "final bracket wins over the first colon",
			titles: []selectionTitle{{Title: "Woodcut series: The Revelation of St John (Apocalypse) (1497-98)", WorkCount: 15}},
			want: []form{{
				Name:      "Woodcut series: The Revelation of St John (Apocalypse)",
				WorkTotal: 15,
				Entries:   []entry{{"1497–98", 15}},
			}},
		},
		{
			name: "Dürer workshop paintings fold into one form",
			titles: []selectionTitle{
				{Title: "Formative years, first Italian journey; paintings until 1496", WorkCount: 10},
				{Title: "Workshop in Nuremberg; paintings (1497-1499)", WorkCount: 9},
				{Title: "Workshop in Nuremberg; paintings (1500-1503)", WorkCount: 8},
				{Title: "Workshop in Nuremberg; paintings (1504)", WorkCount: 2},
				{Title: "Second Italian journey; paintings (1505-1506)", WorkCount: 7},
				{Title: "Workshop in Nuremberg; paintings (1512-1517)", WorkCount: 6},
			},
			want: []form{
				{Name: "Formative years, first Italian journey; paintings until 1496", WorkTotal: 10, Entries: []entry{{"Formative years, first Italian journey; paintings until 1496", 10}}},
				{Name: "Workshop in Nuremberg; paintings", WorkTotal: 25, Entries: []entry{{"1497–1499", 9}, {"1500–1503", 8}, {"1504", 2}, {"1512–1517", 6}}},
				{Name: "Second Italian journey; paintings", WorkTotal: 7, Entries: []entry{{"1505–1506", 7}}},
			},
		},
		{
			name: "empty bracket or empty colon sides do not parse",
			titles: []selectionTitle{
				{Title: "Etchings ()", WorkCount: 1},
				{Title: ": Untitled", WorkCount: 1},
				{Title: "(1500)", WorkCount: 1},
			},
			want: []form{
				{Name: "Etchings ()", WorkTotal: 1, Entries: []entry{{"Etchings ()", 1}}},
				{Name: ": Untitled", WorkTotal: 1, Entries: []entry{{": Untitled", 1}}},
				{Name: "(1500)", WorkTotal: 1, Entries: []entry{{"(1500)", 1}}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := []form{}
			index := 0
			for _, grouped := range groupSelectionsByForm(tc.titles) {
				f := form{Name: grouped.Name, WorkTotal: grouped.WorkTotal}
				for _, selection := range grouped.Selections {
					f.Entries = append(f.Entries, entry{Label: selection.Label, WorkCount: selection.WorkCount})
					if tc.titles[selection.Index].WorkCount != selection.WorkCount {
						t.Errorf("selection index %d does not map back to its input", selection.Index)
					}
					index++
				}
				got = append(got, f)
			}
			if index != len(tc.titles) {
				t.Errorf("grouped %d selections, want %d", index, len(tc.titles))
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("groupSelectionsByForm() =\n%#v\nwant\n%#v", got, tc.want)
			}
		})
	}
}
