package index

import (
	"math"
	"testing"
)

func TestIDF_RareTermOutscoresCommonTerm(t *testing.T) {
	rare := IDF(1000, 1)
	common := IDF(1000, 900)

	if rare <= common {
		t.Errorf("expected rare IDF (%v) > than common IDF (%v)", rare, common)
	}
}

func TestIDF_KnownValues(t *testing.T) {
	// Algebraically verified values from the spec formula:
	//   IDF = ln( (N - df + 0.5) / (df + 0.5) + 1 )
	//
	// IDF(10, 2)  = ln(8.5/2.5 + 1)  = ln(4.4)   ≈ 1.4816
	// IDF(100, 5) = ln(95.5/5.5 + 1) = ln(18.36) ≈ 2.9106
	cases := []struct {
		n, df int
		want  float64
	}{
		{10, 2, 1.4816},
		{100, 5, 2.9106},
	}
	for _, tc := range cases {
		got := IDF(tc.n, tc.df)
		if math.Abs(got-tc.want) > 1e-3 {
			t.Errorf("IDF(%d, %d) = %v, want ~%v", tc.n, tc.df, got, tc.want)
		}
	}
}

func TestTermScore_ZeroTFReturnsZero(t *testing.T) {
	if score := TermScore(0, 100, 10); score != 0 {
		t.Errorf("expected TermScore with zero TF to be 0, got %v", score)
	}
}

func TestTermScore_IdentityAtAverageLength(t *testing.T) {
	got := TermScore(1, 100, 100)
	if math.Abs(got-1.0) > 1e-9 {
		t.Errorf("expected TermScore to be 1 when doc length equals avg doc length, got %v", got)
	}
}

func TestTermScore_SaturationIsMonotonicAndDiminishing(t *testing.T) {
	avgDL := 100.0
	s1 := TermScore(1, avgDL, avgDL)
	s2 := TermScore(2, avgDL, avgDL)
	s10 := TermScore(10, avgDL, avgDL)
	s100 := TermScore(100, avgDL, avgDL)

	if !(s1 < s2 && s2 < s10 && s10 < s100) {
		t.Errorf("scores not monotonic in TF: s1=%v s2=%v s10=%v s100=%v", s1, s2, s10, s100)
	}

	gain1to2 := s2 - s1
	gain2to10 := (s10 - s2) / 8.0 // normalize by TF increase

	if !(gain1to2 > gain2to10) {
		t.Errorf("diminishing returns not observed: gain1to2=%v gain2to10=%v", gain1to2, gain2to10)
	}
}

func TestTermScore_LongerDocsScoreLower(t *testing.T) {
	avgDL := 100.0
	short := TermScore(2, 50, avgDL)
	avg := TermScore(2, 150, avgDL)
	long := TermScore(2, 300, avgDL)

	if !(short > avg && avg > long) {
		t.Errorf("longer docs should score lower: short=%v avg=%v long=%v", short, avg, long)
	}
}

func TestFieldWeight(t *testing.T) {
	cases := []struct {
		name string
		f    Field
		want float64
	}{
		{"title gets 3x", FieldTitle, 3.0},
		{"abstract gets 1x", FieldAbstract, 1.0},
		{"unknown field gets 0", Field(0), 0.0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FieldWeight(tc.f); got != tc.want {
				t.Errorf("FieldWeight(%v) = %v, want %v", tc.f, got, tc.want)
			}
		})
	}
}
