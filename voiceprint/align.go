package voiceprint

import "math"

// pairScore 计算两帧的配对分数：余弦相似度加一后除以二，范围 [0,1]。
// 浮点舍入导致的轻微越界按端点处理。
func pairScore(a, b Frame) float64 {
	var dot, na, nb float64
	for k := 0; k < len(a); k++ {
		dot += a[k] * b[k]
		na += a[k] * a[k]
		nb += b[k] * b[k]
	}
	sim := 0.0
	if na > 0 && nb > 0 {
		sim = dot / math.Sqrt(na*nb)
	}
	s := (sim + 1) / 2
	if s < 0 {
		return 0
	}
	if s > 1 {
		return 1
	}
	return s
}

// 对齐规则：从 (0,0) 到 (n-1,m-1)，每步为 (1,0)、(0,1) 或 (1,1)。
// 同一侧单独前进最多连续两次；双方同时前进或另一侧单独前进都会重置该侧计数。
// 对齐得分为访问到的所有帧对（含首尾）配对分数的算术平均值。

// sumByCount 记录到达某状态时，各帧对数量对应的最大分数和。
type sumByCount map[int]float64

// alignState 是 DP 状态：当前位置与两侧各自的连续单独前进计数。
type alignState struct {
	i, j       int
	runA, runB int
}

// bestAlignmentScore 返回两段录音在受限对齐规则下的最高平均配对分；
// 不存在合法完整对齐时返回 0。
func bestAlignmentScore(a, b Recording) float64 {
	n, m := len(a), len(b)
	if n == 0 || m == 0 {
		return 0
	}
	// 路径上的帧对数 = 步数 + 1，步数最多 (n-1)+(m-1)。
	maxCount := n + m - 1
	// dp[state][count] = 到达该状态且已访问 count 个帧对时的最大分数和。
	cur := map[alignState]sumByCount{
		{i: 0, j: 0, runA: 0, runB: 0}: {1: pairScore(a[0], b[0])},
	}
	best := math.Inf(-1)
	for len(cur) > 0 {
		next := make(map[alignState]sumByCount)
		for st, sums := range cur {
			for cnt, sum := range sums {
				if st.i == n-1 && st.j == m-1 {
					if avg := sum / float64(cnt); avg > best {
						best = avg
					}
					continue
				}
				if cnt >= maxCount {
					continue
				}
				// 双方同时前进：重置两侧连续计数。
				if st.i+1 < n && st.j+1 < m {
					ns := alignState{i: st.i + 1, j: st.j + 1, runA: 0, runB: 0}
					relax(next, ns, cnt+1, sum+pairScore(a[st.i+1], b[st.j+1]))
				}
				// 仅 a 侧前进：runA+1，重置 runB。
				if st.i+1 < n && st.runA < 2 {
					ns := alignState{i: st.i + 1, j: st.j, runA: st.runA + 1, runB: 0}
					relax(next, ns, cnt+1, sum+pairScore(a[st.i+1], b[st.j]))
				}
				// 仅 b 侧前进：runB+1，重置 runA。
				if st.j+1 < m && st.runB < 2 {
					ns := alignState{i: st.i, j: st.j + 1, runA: 0, runB: st.runB + 1}
					relax(next, ns, cnt+1, sum+pairScore(a[st.i], b[st.j+1]))
				}
			}
		}
		cur = next
	}
	if math.IsInf(best, -1) {
		return 0
	}
	return best
}

// relax 用更大的分数和更新 DP 表项。
func relax(dp map[alignState]sumByCount, st alignState, cnt int, sum float64) {
	sums, ok := dp[st]
	if !ok {
		sums = make(sumByCount)
		dp[st] = sums
	}
	if old, ok := sums[cnt]; !ok || sum > old {
		sums[cnt] = sum
	}
}
