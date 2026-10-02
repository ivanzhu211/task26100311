package voiceprint

import (
	"math"
	"testing"
)

// unitFrame 返回第 k 个标准基单位向量构成的 20 维帧。
func unitFrame(k int) Frame {
	f := make(Frame, FrameDim)
	f[k] = 1
	return f
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}

func enrollOrtho(t *testing.T, v *Verifier, id string, rec Recording) {
	t.Helper()
	if err := v.Enroll(id, [3]Recording{rec, rec, rec}); err != nil {
		t.Fatalf("Enroll(%q) 失败: %v", id, err)
	}
}

// 三段参考与待验录音均为相同顺序的两个正交单位帧 [e1, e2]。
// 总分最高的对齐不止一种：对角对齐 (e1,e1),(e2,e2) 总分 2、共 2 对，平均 1；
// 带一次单侧前进的对齐如 (e1,e1),(e2,e1),(e2,e2) 总分也是 2、共 3 对，平均 2/3。
// 取最高平均分应得置信度 1，若误用最高总分对应的对齐则只能得到 2/3。
func TestVerifyOrthogonalFramesConfidenceOne(t *testing.T) {
	v := NewVerifier()
	rec := Recording{unitFrame(0), unitFrame(1)}
	enrollOrtho(t, v, "alice", rec)

	confidence, pass, err := v.Verify("alice", rec)
	if err != nil {
		t.Fatalf("Verify 失败: %v", err)
	}
	if !almostEqual(confidence, 1) {
		t.Errorf("置信度 = %v, 期望 1", confidence)
	}
	if !pass {
		t.Error("期望通过，实际被拒绝")
	}
}

// 单帧参考与三个相同帧可以完整对齐（待验侧连续单独前进两次），
// 与四个相同帧不存在合法完整对齐（需要连续三次），
// 三段参考相同时置信度分别为 1 和 0。
func TestVerifySingleFrameReference(t *testing.T) {
	v := NewVerifier()
	ref := Recording{unitFrame(0)}
	enrollOrtho(t, v, "bob", ref)

	probe3 := Recording{unitFrame(0), unitFrame(0), unitFrame(0)}
	confidence, pass, err := v.Verify("bob", probe3)
	if err != nil {
		t.Fatalf("Verify(3 帧) 失败: %v", err)
	}
	if !almostEqual(confidence, 1) {
		t.Errorf("3 帧待验置信度 = %v, 期望 1", confidence)
	}
	if !pass {
		t.Error("3 帧待验期望通过，实际被拒绝")
	}

	probe4 := Recording{unitFrame(0), unitFrame(0), unitFrame(0), unitFrame(0)}
	confidence, pass, err = v.Verify("bob", probe4)
	if err != nil {
		t.Fatalf("Verify(4 帧) 失败: %v", err)
	}
	if !almostEqual(confidence, 0) {
		t.Errorf("4 帧待验置信度 = %v, 期望 0", confidence)
	}
	if pass {
		t.Error("4 帧待验期望拒绝，实际通过")
	}
}

func TestEnrollValidation(t *testing.T) {
	v := NewVerifier()
	good := Recording{unitFrame(0)}

	if err := v.Enroll("empty", [3]Recording{good, good, {}}); err != ErrInvalidReference {
		t.Errorf("空参考录音: err = %v, 期望 %v", err, ErrInvalidReference)
	}
	badDim := Recording{make(Frame, FrameDim+1)}
	if err := v.Enroll("dim", [3]Recording{good, badDim, good}); err != ErrInvalidReference {
		t.Errorf("维度错误: err = %v, 期望 %v", err, ErrInvalidReference)
	}
	if err := v.Enroll("ok", [3]Recording{good, good, good}); err != nil {
		t.Errorf("合法登记: err = %v, 期望 nil", err)
	}
}

func TestVerifyErrors(t *testing.T) {
	v := NewVerifier()
	enrollOrtho(t, v, "carol", Recording{unitFrame(0)})

	if _, _, err := v.Verify("carol", Recording{}); err != ErrInvalidProbe {
		t.Errorf("空待验录音: err = %v, 期望 %v", err, ErrInvalidProbe)
	}
	if _, _, err := v.Verify("nobody", Recording{unitFrame(0)}); err == nil {
		t.Error("未登记身份: 期望返回错误")
	}
}

// 中位数取三段参考比较结果的中间值。
func TestVerifyMedianOfThree(t *testing.T) {
	v := NewVerifier()
	e1 := unitFrame(0)
	// 与待验 [e1,e1,e1] 的比较结果分别为 1、1、0。
	refs := [3]Recording{
		{e1},
		{e1},
		{e1, e1, e1, e1},
	}
	if err := v.Enroll("dave", refs); err != nil {
		t.Fatalf("Enroll 失败: %v", err)
	}
	confidence, pass, err := v.Verify("dave", Recording{e1, e1, e1})
	if err != nil {
		t.Fatalf("Verify 失败: %v", err)
	}
	if !almostEqual(confidence, 1) {
		t.Errorf("置信度 = %v, 期望中位数 1", confidence)
	}
	if !pass {
		t.Error("期望通过，实际被拒绝")
	}
}
