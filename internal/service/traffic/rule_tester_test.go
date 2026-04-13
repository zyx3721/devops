package traffic

import (
	"math"
	"testing"
	"time"
)

// TestCalculateLatencyStats_EmptyArray 测试空数组情况
func TestCalculateLatencyStats_EmptyArray(t *testing.T) {
	latencies := []time.Duration{}

	avg, p50, p90, p99 := calculateLatencyStats(latencies)

	if avg != 0 || p50 != 0 || p90 != 0 || p99 != 0 {
		t.Errorf("空数组应返回全0，实际: avg=%.2f, p50=%.2f, p90=%.2f, p99=%.2f", avg, p50, p90, p99)
	}
}

// TestCalculateLatencyStats_SingleElement 测试单元素数组
func TestCalculateLatencyStats_SingleElement(t *testing.T) {
	latencies := []time.Duration{100 * time.Millisecond}

	avg, p50, p90, p99 := calculateLatencyStats(latencies)

	expected := 100.0
	if avg != expected || p50 != expected || p90 != expected || p99 != expected {
		t.Errorf("单元素数组所有值应为 %.2f，实际: avg=%.2f, p50=%.2f, p90=%.2f, p99=%.2f",
			expected, avg, p50, p90, p99)
	}
}

// TestCalculateLatencyStats_SortedArray 测试已排序数组
func TestCalculateLatencyStats_SortedArray(t *testing.T) {
	// 创建 100 个元素的已排序数组：10ms, 20ms, ..., 1000ms
	latencies := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		latencies[i] = time.Duration(i+1) * 10 * time.Millisecond
	}

	avg, p50, p90, p99 := calculateLatencyStats(latencies)

	// 验证平均值：(10+20+...+1000)/100 = 505
	expectedAvg := 505.0
	if math.Abs(avg-expectedAvg) > 0.1 {
		t.Errorf("平均值应为 %.2f，实际: %.2f", expectedAvg, avg)
	}

	// 验证 P50：索引 50 对应第 51 个元素是 510ms
	expectedP50 := 510.0
	if p50 != expectedP50 {
		t.Errorf("P50 应为 %.2f，实际: %.2f", expectedP50, p50)
	}

	// 验证 P90：索引 90 对应第 91 个元素是 910ms
	expectedP90 := 910.0
	if p90 != expectedP90 {
		t.Errorf("P90 应为 %.2f，实际: %.2f", expectedP90, p90)
	}

	// 验证 P99：索引 99 对应第 100 个元素是 1000ms
	expectedP99 := 1000.0
	if p99 != expectedP99 {
		t.Errorf("P99 应为 %.2f，实际: %.2f", expectedP99, p99)
	}
}

// TestCalculateLatencyStats_ReverseSortedArray 测试逆序数组
func TestCalculateLatencyStats_ReverseSortedArray(t *testing.T) {
	// 创建 100 个元素的逆序数组：1000ms, 990ms, ..., 10ms
	latencies := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		latencies[i] = time.Duration(100-i) * 10 * time.Millisecond
	}

	avg, p50, p90, p99 := calculateLatencyStats(latencies)

	// 排序后应该和已排序数组的结果一致
	expectedAvg := 505.0
	if math.Abs(avg-expectedAvg) > 0.1 {
		t.Errorf("平均值应为 %.2f，实际: %.2f", expectedAvg, avg)
	}

	expectedP50 := 510.0
	if p50 != expectedP50 {
		t.Errorf("P50 应为 %.2f，实际: %.2f", expectedP50, p50)
	}

	expectedP90 := 910.0
	if p90 != expectedP90 {
		t.Errorf("P90 应为 %.2f，实际: %.2f", expectedP90, p90)
	}

	expectedP99 := 1000.0
	if p99 != expectedP99 {
		t.Errorf("P99 应为 %.2f，实际: %.2f", expectedP99, p99)
	}
}

// TestCalculateLatencyStats_RandomArray 测试随机数组
func TestCalculateLatencyStats_RandomArray(t *testing.T) {
	// 创建随机数组
	latencies := []time.Duration{
		50 * time.Millisecond,
		200 * time.Millisecond,
		100 * time.Millisecond,
		300 * time.Millisecond,
		150 * time.Millisecond,
		250 * time.Millisecond,
		80 * time.Millisecond,
		180 * time.Millisecond,
		120 * time.Millisecond,
		220 * time.Millisecond,
	}

	avg, p50, p90, p99 := calculateLatencyStats(latencies)

	// 验证平均值：(50+200+100+300+150+250+80+180+120+220)/10 = 165
	expectedAvg := 165.0
	if math.Abs(avg-expectedAvg) > 0.1 {
		t.Errorf("平均值应为 %.2f，实际: %.2f", expectedAvg, avg)
	}

	// 验证百分位数在合理范围内
	if p50 < 50 || p50 > 300 {
		t.Errorf("P50 应在 50-300 范围内，实际: %.2f", p50)
	}

	if p90 < p50 || p90 > 300 {
		t.Errorf("P90 应大于等于 P50 且小于等于 300，实际: %.2f", p90)
	}

	if p99 < p90 || p99 > 300 {
		t.Errorf("P99 应大于等于 P90 且小于等于 300，实际: %.2f", p99)
	}
}

// TestCalculateLatencyStats_DoesNotModifyInput 测试不修改原数组
func TestCalculateLatencyStats_DoesNotModifyInput(t *testing.T) {
	// 创建逆序数组
	latencies := []time.Duration{
		500 * time.Millisecond,
		400 * time.Millisecond,
		300 * time.Millisecond,
		200 * time.Millisecond,
		100 * time.Millisecond,
	}

	// 保存原始数据的副本
	original := make([]time.Duration, len(latencies))
	copy(original, latencies)

	// 调用函数
	calculateLatencyStats(latencies)

	// 验证原数组未被修改
	for i := range latencies {
		if latencies[i] != original[i] {
			t.Errorf("原数组被修改：索引 %d，原值 %v，现值 %v", i, original[i], latencies[i])
		}
	}
}

// TestCalculateLatencyStats_LargeArray 测试大规模数组
func TestCalculateLatencyStats_LargeArray(t *testing.T) {
	// 创建 10,000 个元素的数组
	size := 10000
	latencies := make([]time.Duration, size)
	for i := 0; i < size; i++ {
		latencies[i] = time.Duration(i+1) * time.Millisecond
	}

	// 测量执行时间
	start := time.Now()
	avg, p50, p90, p99 := calculateLatencyStats(latencies)
	elapsed := time.Since(start)

	// 验证执行时间应小于 100ms
	if elapsed > 100*time.Millisecond {
		t.Errorf("大规模数组处理时间过长：%v（应小于 100ms）", elapsed)
	}

	// 验证结果合理性
	expectedAvg := 5000.5
	if math.Abs(avg-expectedAvg) > 1.0 {
		t.Errorf("平均值应约为 %.2f，实际: %.2f", expectedAvg, avg)
	}

	expectedP50 := 5000.0
	if math.Abs(p50-expectedP50) > 1.0 {
		t.Errorf("P50 应约为 %.2f，实际: %.2f", expectedP50, p50)
	}

	expectedP90 := 9000.0
	if math.Abs(p90-expectedP90) > 1.0 {
		t.Errorf("P90 应约为 %.2f，实际: %.2f", expectedP90, p90)
	}

	expectedP99 := 9900.0
	if math.Abs(p99-expectedP99) > 1.0 {
		t.Errorf("P99 应约为 %.2f，实际: %.2f", expectedP99, p99)
	}

	t.Logf("处理 %d 个样本耗时: %v", size, elapsed)
}

// ============================================================
// 性能基准测试
// ============================================================

// BenchmarkCalculateLatencyStats_100 小规模数据基准测试（100 样本）
func BenchmarkCalculateLatencyStats_100(b *testing.B) {
	latencies := make([]time.Duration, 100)
	for i := 0; i < 100; i++ {
		latencies[i] = time.Duration(i+1) * time.Millisecond
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateLatencyStats(latencies)
	}
}

// BenchmarkCalculateLatencyStats_1000 中规模数据基准测试（1,000 样本）
func BenchmarkCalculateLatencyStats_1000(b *testing.B) {
	latencies := make([]time.Duration, 1000)
	for i := 0; i < 1000; i++ {
		latencies[i] = time.Duration(i+1) * time.Millisecond
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateLatencyStats(latencies)
	}
}

// BenchmarkCalculateLatencyStats_10000 大规模数据基准测试（10,000 样本）
func BenchmarkCalculateLatencyStats_10000(b *testing.B) {
	latencies := make([]time.Duration, 10000)
	for i := 0; i < 10000; i++ {
		latencies[i] = time.Duration(i+1) * time.Millisecond
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateLatencyStats(latencies)
	}
}

// BenchmarkCalculateLatencyStats_100000 超大规模数据基准测试（100,000 样本）
func BenchmarkCalculateLatencyStats_100000(b *testing.B) {
	latencies := make([]time.Duration, 100000)
	for i := 0; i < 100000; i++ {
		latencies[i] = time.Duration(i+1) * time.Millisecond
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateLatencyStats(latencies)
	}
}
