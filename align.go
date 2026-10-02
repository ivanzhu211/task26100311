package voiceprint

// pairScore 计算两帧的配对分数：余弦相似度加一后除以二，范围 [0,1]。
// 帧为单位向量，余弦相似度即点积；浮点舍入导致的轻微越界按端点处理。
func pairScore(a, b Frame) float64 {
	dot := 0.0
	for k := 0; k < FrameDim; k++ {
		dot += a[k] * b[k]
	}
	if dot > 1 {
		dot = 1
	} else if dot < -1 {
		dot = -1
	}
	return (dot + 1) / 2
}

// 对齐状态：同一侧单独前进的连续计数。
// 一侧单独前进后另一侧计数清零，双方同时前进两者都清零，
// 因此可达状态只有以下 5 种。
const (
	stNone = iota // 上一次为同时前进（或起点）
	stA1          // 待验侧连续单独前进 1 次
	stA2          // 待验侧连续单独前进 2 次
	stB1          // 参考侧连续单独前进 1 次
	stB2          // 参考侧连续单独前进 2 次
	numStates
)

// bestAlignment 计算两段录音在合法对齐下的最高平均配对分。
// 对齐从 (0,0) 开始、在 (n-1,m-1) 结束，每步可双方同时前进一格，
// 或仅一侧前进一格（同一侧连续单独前进最多 maxConsecutiveSkips 次）。
// 对齐得分为访问到的所有帧对（含首尾）配对分的算术平均值。
// 不存在合法完整对齐时返回 0。
func bestAlignment(probe, ref Recording) float64 {
	n, m := len(probe), len(ref)
	if n == 0 || m == 0 {
		return 0
	}

	// 动态规划：按 (i, j, 状态) 记录“访问 count 个帧对时的最大总分”，
	// 最终在终点各状态上取 总分/count 的最大值。
	// 仅依赖上一行与当前行左侧，使用两行滚动数组。
	type cell [numStates][]float64 // 下标 count-1 -> 最大总分，nil 表示不可达
	prev := make([]cell, m)
	curr := make([]cell, m)

	set := func(c *cell, st, count int, sum float64) {
		if c[st] == nil {
			c[st] = make([]float64, count)
			for k := range c[st] {
				c[st][k] = -1
			}
		}
		if len(c[st]) < count {
			grown := make([]float64, count)
			for k := range grown {
				grown[k] = -1
			}
			copy(grown, c[st])
			c[st] = grown
		}
		if sum > c[st][count-1] {
			c[st][count-1] = sum
		}
	}
	merge := func(dst *cell, dstSt int, src *cell, srcSt int, add float64) {
		for cnt, sum := range src[srcSt] {
			if sum >= 0 {
				// src 下标 cnt 对应 count=cnt+1，新增一对后 count=cnt+2。
				set(dst, dstSt, cnt+2, sum+add)
			}
		}
	}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			s := pairScore(probe[i], ref[j])
			c := &curr[j]
			if i == 0 && j == 0 {
				set(c, stNone, 1, s)
				continue
			}
			if i > 0 && j > 0 {
				// 双方同时前进：任意前驱状态转移到 stNone。
				p := &prev[j-1]
				for st := 0; st < numStates; st++ {
					merge(c, stNone, p, st, s)
				}
			}
			if i > 0 {
				// 仅待验侧前进：stNone->stA1，stA1->stA2。
				p := &prev[j]
				merge(c, stA1, p, stNone, s)
				merge(c, stA2, p, stA1, s)
			}
			if j > 0 {
				// 仅参考侧前进：stNone->stB1，stB1->stB2。
				p := &curr[j-1]
				merge(c, stB1, p, stNone, s)
				merge(c, stB2, p, stB1, s)
			}
		}
		prev, curr = curr, prev
		for j := range curr {
			curr[j] = cell{}
		}
	}

	best := 0.0
	last := &prev[m-1]
	for st := 0; st < numStates; st++ {
		for cnt, sum := range last[st] {
			if sum >= 0 {
				if avg := sum / float64(cnt+1); avg > best {
					best = avg
				}
			}
		}
	}
	return best
}
