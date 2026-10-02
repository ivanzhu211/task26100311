// Package voiceprint 提供声纹时序核验能力。
// 调用方负责音频解码、质量筛查与 MFCC 提取；
// 本库在进程内存中保存登记特征，并提供登记与核验。
package voiceprint

import (
	"errors"
	"fmt"
	"sort"
)

// FrameDim 是每帧 MFCC 特征的维度。
const FrameDim = 20

// PassThreshold 是核验通过的置信度阈值（含）。
const PassThreshold = 0.82

// maxConsecutiveSkips 是同一侧单独前进的最大连续次数。
const maxConsecutiveSkips = 2

// referenceCount 是每次登记保存的参考录音段数。
const referenceCount = 3

// Frame 是一帧 20 维实数单位向量特征。
type Frame []float64

// Recording 是一段录音按时间顺序排列的帧序列。
type Recording []Frame

// ErrInvalidReference 表示登记特征不合法。
var ErrInvalidReference = errors.New("voiceprint: 登记需要恰好三段非空参考录音，且每帧为 20 维")

// ErrInvalidProbe 表示待验录音不合法。
var ErrInvalidProbe = errors.New("voiceprint: 待验录音不能为空，且每帧为 20 维")

// ErrUnknownIdentity 表示身份未登记。
var ErrUnknownIdentity = errors.New("voiceprint: 身份未登记")

// Verifier 在进程内存中保存各身份的参考特征并提供核验。
type Verifier struct {
	refs map[string][referenceCount]Recording
}

// NewVerifier 创建一个空的核验器。
func NewVerifier() *Verifier {
	return &Verifier{refs: make(map[string][referenceCount]Recording)}
}

func validRecording(r Recording) bool {
	if len(r) == 0 {
		return false
	}
	for _, f := range r {
		if len(f) != FrameDim {
			return false
		}
	}
	return true
}

// Enroll 为一个身份保存恰好三段参考录音的特征，重复登记将覆盖旧特征。
func (v *Verifier) Enroll(id string, refs [referenceCount]Recording) error {
	for _, r := range refs {
		if !validRecording(r) {
			return ErrInvalidReference
		}
	}
	v.refs[id] = refs
	return nil
}

// Verify 用待验录音与已登记的三段参考分别对齐比较，
// 取三个比较结果的中位数作为置信度，达到阈值即通过。
func (v *Verifier) Verify(id string, probe Recording) (confidence float64, pass bool, err error) {
	if !validRecording(probe) {
		return 0, false, ErrInvalidProbe
	}
	refs, ok := v.refs[id]
	if !ok {
		return 0, false, fmt.Errorf("%w: %q", ErrUnknownIdentity, id)
	}
	scores := make([]float64, referenceCount)
	for i, ref := range refs {
		scores[i] = bestAlignment(probe, ref)
	}
	sort.Float64s(scores)
	confidence = scores[referenceCount/2]
	return confidence, confidence >= PassThreshold, nil
}
