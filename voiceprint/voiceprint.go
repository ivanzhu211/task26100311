// Package voiceprint 提供声纹时序核验能力：登记参考特征并在内存中保存，
// 核验时对待验录音与参考录音做受限时间对齐，输出置信度与是否通过。
package voiceprint

import (
	"errors"
	"fmt"
	"sort"
)

// enrollCount 是一次登记所需的参考录音段数。
const enrollCount = 3

// passThreshold 是核验通过的置信度阈值。
const passThreshold = 0.82

// Frame 是一帧 20 维实数单位向量特征。
type Frame [20]float64

// Recording 是一段录音按时间顺序排列的帧序列。
type Recording []Frame

// Verdict 是核验结果：置信度与是否通过。
type Verdict struct {
	Confidence float64
	Passed     bool
}

// Verifier 在进程内存中保存各身份的参考特征。
type Verifier struct {
	refs map[string][]Recording
}

// NewVerifier 创建空的核验器。
func NewVerifier() *Verifier {
	return &Verifier{refs: make(map[string][]Recording)}
}

// Enroll 为一个身份保存恰好三段参考录音的特征，特征被复制到进程内存。
func (v *Verifier) Enroll(id string, refs []Recording) error {
	if len(refs) != enrollCount {
		return fmt.Errorf("voiceprint: enroll %q: need exactly %d reference recordings, got %d", id, enrollCount, len(refs))
	}
	stored := make([]Recording, len(refs))
	for i, r := range refs {
		if len(r) == 0 {
			return fmt.Errorf("voiceprint: enroll %q: reference %d is empty", id, i)
		}
		stored[i] = append(Recording(nil), r...)
	}
	v.refs[id] = stored
	return nil
}

// Verify 接收身份及一段待验录音，返回置信度和是否通过。
// 置信度为待验录音与三段参考录音各自最高对齐平均分的中位数。
func (v *Verifier) Verify(id string, probe Recording) (Verdict, error) {
	refs, ok := v.refs[id]
	if !ok {
		return Verdict{}, fmt.Errorf("voiceprint: verify: unknown identity %q", id)
	}
	if len(probe) == 0 {
		return Verdict{}, errors.New("voiceprint: verify: empty probe recording")
	}
	scores := make([]float64, len(refs))
	for i, ref := range refs {
		scores[i] = bestAlignmentScore(probe, ref)
	}
	sort.Float64s(scores)
	confidence := scores[len(scores)/2]
	return Verdict{Confidence: confidence, Passed: confidence >= passThreshold}, nil
}
