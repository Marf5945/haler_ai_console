package builtin

import "testing"

func TestTokenizeDianliaoQuery(t *testing.T) {
	got := TokenizeDianliaoQuery("Murr.M12, 10m")
	want := map[string]bool{"murr": true, "m12": true, "10m": true}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for _, g := range got {
		if !want[g] {
			t.Fatalf("unexpected token %q in %v", g, got)
		}
	}
}

func TestSearchDianliaoLocal(t *testing.T) {
	recs := []DianliaoRecord{
		{Name: "Murr.M12 公母接線", PartNoQuanta: "Q-001", Spec: "IO link 10m"},
		{Name: "端子台", PartNoQuanta: "Q-002", Spec: "20A"},
		{Name: "斷路器", PartNoQuanta: "Q-003", Spec: "3P"},
	}
	res := SearchDianliaoLocal(recs, "murr 10m", 8)
	if len(res) == 0 || res[0].Record.PartNoQuanta != "Q-001" {
		t.Fatalf("expected Q-001 top, got %+v", res)
	}
	// 料號精確相符要排最前（大加分）
	res2 := SearchDianliaoLocal(recs, "Q-002", 8)
	if len(res2) == 0 || res2[0].Record.PartNoQuanta != "Q-002" {
		t.Fatalf("expected exact partno top, got %+v", res2)
	}
	// 完全不相干 → 無結果
	if r := SearchDianliaoLocal(recs, "xyzzy", 8); len(r) != 0 {
		t.Fatalf("expected no match, got %+v", r)
	}
}
