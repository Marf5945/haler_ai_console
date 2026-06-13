package source_trust

import "testing"

// 測試 .gov 網域分類為 INSTITUTIONAL_BUT_UNVERIFIED
func TestClassifyGovDomain(t *testing.T) {
	e := Classify("https://www.cdc.gov/health/report", "", nil)
	if e.DomainClass != DomainOfficialGov {
		t.Errorf("gov domain class = %s, want official_gov_domain", e.DomainClass)
	}
	if e.SourceTrustLabel != LabelInstitutionalButUnverified {
		t.Errorf("gov label = %s, want INSTITUTIONAL_BUT_UNVERIFIED", e.SourceTrustLabel)
	}
}

// 測試 .edu 網域
func TestClassifyEduDomain(t *testing.T) {
	e := Classify("https://www.mit.edu/research/paper", "", nil)
	if e.DomainClass != DomainOfficialEdu {
		t.Errorf("edu domain class = %s", e.DomainClass)
	}
	if e.SourceTrustLabel != LabelInstitutionalButUnverified {
		t.Errorf("edu label = %s", e.SourceTrustLabel)
	}
}

// 測試 UGC 內容指紋覆蓋 .edu（§9.4 硬規則）
func TestClassifyEduWithUGC(t *testing.T) {
	e := Classify("https://forum.mit.edu/discussion", "網友分享的經驗", nil)
	if e.SourceTrustLabel != LabelUserGenerated {
		t.Errorf("edu + UGC should be USER_GENERATED, got %s", e.SourceTrustLabel)
	}
}

// 測試論壇內容指紋
func TestClassifyForumContent(t *testing.T) {
	e := Classify("https://example.com/article", "這是一個論壇的留言", nil)
	if e.SourceTrustLabel != LabelUserGenerated {
		t.Errorf("forum content should be USER_GENERATED, got %s", e.SourceTrustLabel)
	}
	if len(e.ContentFlags) == 0 {
		t.Error("should have content flags")
	}
}

// 測試可疑網域
func TestClassifySuspiciousDomain(t *testing.T) {
	e := Classify("https://192.168.1.1/page", "", nil)
	if e.DomainClass != DomainSuspicious {
		t.Errorf("IP domain = %s, want suspicious_domain", e.DomainClass)
	}
	if e.SourceTrustLabel != LabelLowTrust {
		t.Errorf("suspicious label = %s, want LOW_TRUST", e.SourceTrustLabel)
	}
}

// 測試視覺標記 + 未知網域 → PENDING_SOURCE_REVIEW（§9.3）
func TestClassifyVisualClaimUnknownDomain(t *testing.T) {
	e := Classify("https://some-unknown-site.com/page", "",
		[]VisualFlag{VisualOfficialLogo})
	if e.SourceTrustLabel != LabelPendingSourceReview {
		t.Errorf("visual + unknown domain = %s, want PENDING_SOURCE_REVIEW", e.SourceTrustLabel)
	}
}

// 測試一般未知網域 → UNVERIFIED
func TestClassifyUnknownDomain(t *testing.T) {
	e := Classify("https://random-blog.com/post", "", nil)
	if e.SourceTrustLabel != LabelUnverified {
		t.Errorf("unknown domain = %s, want UNVERIFIED", e.SourceTrustLabel)
	}
}

// 測試正規化主機名稱去除 www
func TestCanonicalHostname(t *testing.T) {
	e := Classify("https://www.example.com/page", "", nil)
	if e.CanonicalHostname != "example.com" {
		t.Errorf("canonical hostname = %s, want example.com", e.CanonicalHostname)
	}
}

// 測試 UGC 子路徑偵測
func TestClassifyUGCSubpath(t *testing.T) {
	e := Classify("https://stackoverflow.com/questions/12345", "", nil)
	if e.DomainClass != DomainUserGeneratedSubpath {
		t.Errorf("SO questions path = %s, want user_generated_subpath", e.DomainClass)
	}
	if e.SourceTrustLabel != LabelUserGenerated {
		t.Errorf("UGC subpath label = %s", e.SourceTrustLabel)
	}
}

// 測試 AUTH_OK 計算
func TestComputeAuthOK(t *testing.T) {
	// 條件全滿：VERIFIED + active + 無 UGC flag
	ok := ComputeAuthOK(SourceTrustEvidence{
		SourceTrustLabel: LabelVerifiedAuthority,
		AllowlistStatus:  "active",
		ContentFlags:     nil,
	})
	if !ok {
		t.Error("should be AUTH_OK when all conditions met")
	}

	// 非 VERIFIED → false
	ok = ComputeAuthOK(SourceTrustEvidence{
		SourceTrustLabel: LabelInstitutionalButUnverified,
		AllowlistStatus:  "active",
	})
	if ok {
		t.Error("non-VERIFIED should not be AUTH_OK")
	}

	// VERIFIED 但 allowlist expired → false
	ok = ComputeAuthOK(SourceTrustEvidence{
		SourceTrustLabel: LabelVerifiedAuthority,
		AllowlistStatus:  "expired",
	})
	if ok {
		t.Error("expired allowlist should not be AUTH_OK")
	}

	// VERIFIED + active 但有 disclaimer flag → false
	ok = ComputeAuthOK(SourceTrustEvidence{
		SourceTrustLabel: LabelVerifiedAuthority,
		AllowlistStatus:  "active",
		ContentFlags:     []ContentFlag{ContentDisclaimer},
	})
	if ok {
		t.Error("disclaimer flag should prevent AUTH_OK")
	}
}

// 測試 ShouldBlock
func TestShouldBlock(t *testing.T) {
	// PENDING + 高影響 → blocking
	if !ShouldBlock(LabelPendingSourceReview, true, "general") {
		t.Error("PENDING + high impact should block")
	}
	// PENDING + 一般 → non-blocking
	if ShouldBlock(LabelPendingSourceReview, false, "general_qa") {
		t.Error("PENDING + normal should not block")
	}
	// PENDING + 敏感用途 → blocking
	if !ShouldBlock(LabelPendingSourceReview, false, "legal") {
		t.Error("PENDING + legal should block")
	}
	// LOW_TRUST → always blocking
	if !ShouldBlock(LabelLowTrust, false, "general") {
		t.Error("LOW_TRUST should always block")
	}
}

// 測試 high-impact domain 不可移除 built-in
func TestHighImpactValidation(t *testing.T) {
	if err := ValidateUserRemoval("legal"); err == nil {
		t.Error("should not allow removing built-in domain")
	}
	if err := ValidateUserRemoval("custom_domain"); err != nil {
		t.Errorf("should allow removing custom domain: %v", err)
	}
}

// 測試 UGC ranking 不改變 label
func TestRankingNeverUpgradesLabel(t *testing.T) {
	score := AdjustRanking(50, QualitySignal{
		UpvoteCount:      500,
		AcceptedAnswer:   true,
		AuthorReputation: 10000,
	})
	if score <= 50 {
		t.Error("quality signals should increase ranking")
	}
	if score > 100 {
		t.Error("score should not exceed 100")
	}
	// 驗證 label 不變（由呼叫端確保，這裡只測分數邏輯）
}

// 測試 ScopeFingerprint 一致性
func TestScopeFingerprint(t *testing.T) {
	input1 := ScopeFingerprintInput{
		CanonicalHostname: "example.edu",
		URLPatternHash:    "sha256:abc",
		ContentType:       []string{"html", "pdf"},
		SourcePurpose:     []string{"research_reference"},
		AllowedFor:        []string{"rag_ranking"},
		NotAllowedFor:     RequiredNotAllowedFor,
	}
	input2 := ScopeFingerprintInput{
		CanonicalHostname: "example.edu",
		URLPatternHash:    "sha256:abc",
		ContentType:       []string{"pdf", "html"}, // 順序不同
		SourcePurpose:     []string{"research_reference"},
		AllowedFor:        []string{"rag_ranking"},
		NotAllowedFor:     RequiredNotAllowedFor,
	}
	fp1 := ComputeFingerprint(input1)
	fp2 := ComputeFingerprint(input2)
	if fp1 != fp2 {
		t.Error("fingerprints should be identical regardless of order")
	}

	// 改變 hostname 應產生不同 fingerprint
	input3 := input1
	input3.CanonicalHostname = "other.edu"
	fp3 := ComputeFingerprint(input3)
	if fp1 == fp3 {
		t.Error("different hostname should produce different fingerprint")
	}
}

// 測試使用者友善標籤
func TestUserFriendlyLabel(t *testing.T) {
	cases := map[SourceTrustLabel]string{
		LabelVerifiedAuthority:   "已驗證",
		LabelUserGenerated:       "社群來源",
		LabelPendingSourceReview: "待審查",
		LabelLowTrust:            "低信任",
	}
	for label, want := range cases {
		got := UserFriendlyLabel(label)
		if got != want {
			t.Errorf("UserFriendlyLabel(%s) = %s, want %s", label, got, want)
		}
	}
}
