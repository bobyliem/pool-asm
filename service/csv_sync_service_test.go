package service

import (
	"testing"
)

func TestColLetterToIndex(t *testing.T) {
	cases := map[string]int{
		"A":  0,
		"F":  5,
		"G":  6,
		"I":  8,
		"K":  10,
		"M":  12,
		"N":  13,
		"O":  14,
		"AA": 26,
	}
	for col, want := range cases {
		if got := colLetterToIndex(col); got != want {
			t.Errorf("colLetterToIndex(%q) = %d; want %d", col, got, want)
		}
	}
}

func TestParseFlexibleDate(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		{"16 Sep 26", "2026-09-16"},
		{"5 Nov 26", "2026-11-05"},
		{"2026-11-08", "2026-11-08"},
		{"08/11/2026", "2026-11-08"},
		{"", nil},
	}
	for _, tc := range cases {
		if tc.in == "" {
			if got := formatColumnValue("ss_skb", "tgl", tc.in); got != nil {
				t.Errorf("formatColumnValue(tgl, %q) = %v; want nil", tc.in, got)
			}
			continue
		}
		if got := parseFlexibleDate(tc.in); got != tc.want {
			t.Errorf("parseFlexibleDate(%q) = %v; want %v", tc.in, got, tc.want)
		}
	}
}

func TestFormatColumnValue(t *testing.T) {
	if got := formatColumnValue("imp_service_order", "released_at", "2024-10-13T07:45:14+07:00"); got != "2024-10-13 07:45:14" {
		t.Errorf("released_at got %v", got)
	}
	if got := formatColumnValue("imp_service_order", "completed_at", ""); got != nil {
		t.Errorf("completed_at empty got %v; want nil", got)
	}
	if got := formatColumnValue("imp_visitor", "entrydate", ""); got != "" {
		t.Errorf("imp_visitor.entrydate empty got %v; want empty string", got)
	}
	if got := formatColumnValue("imp_visitor", "exitdate", "2026-11-08T16:30:25Z"); got != "2026-11-08 16:30:25" {
		t.Errorf("exitdate got %v", got)
	}
	if got := formatColumnValue("imp_pending_pr", "total", "1,350,000.4"); got != 1350000 {
		t.Errorf("imp_pending_pr total got %v", got)
	}
	if got := formatColumnValue("ss_pok", "volume", "12,500.75"); got != 12500.75 {
		t.Errorf("ss_pok volume got %v", got)
	}
}
