package voiceprint

import "testing"

// unitFrame 返回第 k 维为 1 的单位向量帧。
func unitFrame(k int) Frame {
	var f Frame
	f[k] = 1
	return f
}

// repeat 返回由同一帧重复 n 次组成的录音。
func repeat(f Frame, n int) Recording {
	r := make(Recording, n)
	for i := range r {
		r[i] = f
	}
	return r
}

// enrollRefs 用三段相同参考登记一个身份。
func enrollRefs(t *testing.T, v *Verifier, id string, ref Recording) {
	t.Helper()
	if err := v.Enroll(id, []Recording{ref, ref, ref}); err != nil {
		t.Fatalf("Enroll(%q) failed: %v", id, err)
	}
}

// TestOrthogonalFramesFullConfidence 覆盖：三段参考与待验录音均为相同顺序的
// 两个正交单位帧 [e1,e2] 时，置信度应为 1。
// 该用例同时区分最高总分与最高平均分：对角对齐 (e1,e1),(e2,e2) 总分 2、平均 1；
// 单侧拉伸对齐 (e1,e1),(e2,e1),(e2,e2) 总分 2.5 更高，但平均 2.5/3≈0.833 更低。
// 规则取最高平均分，因此结果为 1 而非 0.833。
func TestOrthogonalFramesFullConfidence(t *testing.T) {
	e1, e2 := unitFrame(0), unitFrame(1)
	ref := Recording{e1, e2}

	if got := bestAlignmentScore(ref, ref); got != 1 {
		t.Fatalf("bestAlignmentScore = %v, want 1 (max average, not max sum 2.5/3)", got)
	}

	v := NewVerifier()
	enrollRefs(t, v, "alice", ref)
	verdict, err := v.Verify("alice", Recording{e1, e2})
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if verdict.Confidence != 1 {
		t.Errorf("Confidence = %v, want 1", verdict.Confidence)
	}
	if !verdict.Passed {
		t.Errorf("Passed = false, want true")
	}
}

// TestSingleFrameRefStretchLimits 覆盖：单帧参考与三个相同帧可以完整对齐
// （单侧连续前进两次，置信度 1），与四个相同帧不能完整对齐（需连续前进三次，
// 超出限制，无合法完整对齐，置信度 0）。
func TestSingleFrameRefStretchLimits(t *testing.T) {
	f := unitFrame(0)
	ref := Recording{f}

	v := NewVerifier()
	enrollRefs(t, v, "bob", ref)

	ok3, err := v.Verify("bob", repeat(f, 3))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if ok3.Confidence != 1 || !ok3.Passed {
		t.Errorf("3-frame probe: got %+v, want Confidence 1 and Passed true", ok3)
	}

	no4, err := v.Verify("bob", repeat(f, 4))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if no4.Confidence != 0 || no4.Passed {
		t.Errorf("4-frame probe: got %+v, want Confidence 0 and Passed false", no4)
	}
}

// TestPairScoreClamping 覆盖配对分数范围与端点处理。
func TestPairScoreClamping(t *testing.T) {
	e1, e2 := unitFrame(0), unitFrame(1)
	if s := pairScore(e1, e1); s != 1 {
		t.Errorf("pairScore(e1,e1) = %v, want 1", s)
	}
	if s := pairScore(e1, e2); s != 0.5 {
		t.Errorf("pairScore(e1,e2) = %v, want 0.5", s)
	}
	neg := e1
	neg[0] = -1
	if s := pairScore(e1, neg); s != 0 {
		t.Errorf("pairScore(e1,-e1) = %v, want 0", s)
	}
}

// TestMedianConfidence 覆盖三段参考比较结果取中位数。
func TestMedianConfidence(t *testing.T) {
	f := unitFrame(0)
	v := NewVerifier()
	// 三段参考得分分别为 1、1、0，中位数为 1。
	refs := []Recording{repeat(f, 3), repeat(f, 3), repeat(f, 4)}
	if err := v.Enroll("carol", refs); err != nil {
		t.Fatalf("Enroll failed: %v", err)
	}
	verdict, err := v.Verify("carol", repeat(f, 3))
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if verdict.Confidence != 1 || !verdict.Passed {
		t.Errorf("got %+v, want Confidence 1 and Passed true", verdict.Confidence)
	}
}

// TestEnrollValidation 覆盖登记参数校验。
func TestEnrollValidation(t *testing.T) {
	v := NewVerifier()
	f := unitFrame(0)
	if err := v.Enroll("dave", []Recording{repeat(f, 2), repeat(f, 2)}); err == nil {
		t.Error("Enroll with 2 refs: want error, got nil")
	}
	if err := v.Enroll("dave", []Recording{repeat(f, 2), {}, repeat(f, 2)}); err == nil {
		t.Error("Enroll with empty ref: want error, got nil")
	}
	if _, err := v.Verify("nobody", repeat(f, 2)); err == nil {
		t.Error("Verify unknown identity: want error, got nil")
	}
}
