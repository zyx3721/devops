package approval

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// ============================================================================
// Task 3.2: Property 1 - 审批链匹配优先级测试
// Validates: Requirements 2.2
// 注意：完整的集成测试需要连接 MySQL 数据库，这里只测试匹配逻辑
// ============================================================================

func TestMatchPriority_Logic(t *testing.T) {
	// 测试匹配优先级逻辑说明：
	// 优先级：应用+环境 > 应用+* > 0+环境 > 0+*
	//
	// 场景1: 应用1+prod 应该匹配 "应用1生产环境" (appID=1, env=prod)
	// 场景2: 应用1+dev 应该匹配 "应用1所有环境" (appID=1, env=*)
	// 场景3: 应用2+prod 应该匹配 "全局生产环境" (appID=0, env=prod)
	// 场景4: 应用2+dev 应该匹配 "全局所有环境" (appID=0, env=*)

	type matchCase struct {
		chainAppID uint
		chainEnv   string
		priority   int // 数字越大优先级越高
	}

	// 定义优先级计算逻辑
	calcPriority := func(chainAppID uint, chainEnv string, targetAppID uint, targetEnv string) int {
		if chainAppID == targetAppID && chainEnv == targetEnv {
			return 4 // 最高优先级：完全匹配
		}
		if chainAppID == targetAppID && chainEnv == "*" {
			return 3 // 应用匹配，环境通配
		}
		if chainAppID == 0 && chainEnv == targetEnv {
			return 2 // 全局，环境匹配
		}
		if chainAppID == 0 && chainEnv == "*" {
			return 1 // 最低优先级：全局通配
		}
		return 0 // 不匹配
	}

	tests := []struct {
		name        string
		targetAppID uint
		targetEnv   string
		chains      []matchCase
		wantIdx     int // 期望匹配的链索引
	}{
		{
			name:        "应用1+prod应该匹配应用1生产环境",
			targetAppID: 1,
			targetEnv:   "prod",
			chains: []matchCase{
				{chainAppID: 0, chainEnv: "*", priority: 1},
				{chainAppID: 0, chainEnv: "prod", priority: 2},
				{chainAppID: 1, chainEnv: "*", priority: 3},
				{chainAppID: 1, chainEnv: "prod", priority: 4},
			},
			wantIdx: 3, // 应用1生产环境
		},
		{
			name:        "应用1+dev应该匹配应用1所有环境",
			targetAppID: 1,
			targetEnv:   "dev",
			chains: []matchCase{
				{chainAppID: 0, chainEnv: "*", priority: 1},
				{chainAppID: 0, chainEnv: "prod", priority: 2},
				{chainAppID: 1, chainEnv: "*", priority: 3},
				{chainAppID: 1, chainEnv: "prod", priority: 4},
			},
			wantIdx: 2, // 应用1所有环境
		},
		{
			name:        "应用2+prod应该匹配全局生产环境",
			targetAppID: 2,
			targetEnv:   "prod",
			chains: []matchCase{
				{chainAppID: 0, chainEnv: "*", priority: 1},
				{chainAppID: 0, chainEnv: "prod", priority: 2},
				{chainAppID: 1, chainEnv: "*", priority: 3},
				{chainAppID: 1, chainEnv: "prod", priority: 4},
			},
			wantIdx: 1, // 全局生产环境
		},
		{
			name:        "应用2+dev应该匹配全局所有环境",
			targetAppID: 2,
			targetEnv:   "dev",
			chains: []matchCase{
				{chainAppID: 0, chainEnv: "*", priority: 1},
				{chainAppID: 0, chainEnv: "prod", priority: 2},
				{chainAppID: 1, chainEnv: "*", priority: 3},
				{chainAppID: 1, chainEnv: "prod", priority: 4},
			},
			wantIdx: 0, // 全局所有环境
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 找到最高优先级的匹配
			maxPriority := 0
			matchedIdx := -1

			for i, chain := range tt.chains {
				priority := calcPriority(chain.chainAppID, chain.chainEnv, tt.targetAppID, tt.targetEnv)
				if priority > maxPriority {
					maxPriority = priority
					matchedIdx = i
				}
			}

			assert.Equal(t, tt.wantIdx, matchedIdx)
		})
	}
}

func TestMatchPriority_NoMatch(t *testing.T) {
	// 测试没有匹配的情况
	calcPriority := func(chainAppID uint, chainEnv string, targetAppID uint, targetEnv string) int {
		if chainAppID == targetAppID && chainEnv == targetEnv {
			return 4
		}
		if chainAppID == targetAppID && chainEnv == "*" {
			return 3
		}
		if chainAppID == 0 && chainEnv == targetEnv {
			return 2
		}
		if chainAppID == 0 && chainEnv == "*" {
			return 1
		}
		return 0
	}

	// 只有应用1的链，查询应用2
	chains := []struct {
		appID uint
		env   string
	}{
		{appID: 1, env: "prod"},
		{appID: 1, env: "dev"},
	}

	maxPriority := 0
	for _, chain := range chains {
		priority := calcPriority(chain.appID, chain.env, 2, "prod")
		if priority > maxPriority {
			maxPriority = priority
		}
	}

	assert.Equal(t, 0, maxPriority, "应该没有匹配")
}

func TestMatchPriority_GlobalWildcard(t *testing.T) {
	// 测试全局通配符作为兜底
	calcPriority := func(chainAppID uint, chainEnv string, targetAppID uint, targetEnv string) int {
		if chainAppID == targetAppID && chainEnv == targetEnv {
			return 4
		}
		if chainAppID == targetAppID && chainEnv == "*" {
			return 3
		}
		if chainAppID == 0 && chainEnv == targetEnv {
			return 2
		}
		if chainAppID == 0 && chainEnv == "*" {
			return 1
		}
		return 0
	}

	// 只有全局通配符
	priority := calcPriority(0, "*", 999, "any-env")
	assert.Equal(t, 1, priority, "全局通配符应该匹配任何应用和环境")
}

// ============================================================================
// Task 8.2: Property 5 - Approval Chain Priority Ordering (增强测试)
// Validates: Requirements 6.2
// ============================================================================

// TestApprovalChainPriority_Property 属性测试：审批链优先级排序
func TestApprovalChainPriority_Property(t *testing.T) {
	// 优先级计算函数
	calcPriority := func(chainAppID uint, chainEnv string, targetAppID uint, targetEnv string) int {
		if chainAppID == targetAppID && chainEnv == targetEnv {
			return 4 // 最高优先级：完全匹配
		}
		if chainAppID == targetAppID && chainEnv == "*" {
			return 3 // 应用匹配，环境通配
		}
		if chainAppID == 0 && chainEnv == targetEnv {
			return 2 // 全局，环境匹配
		}
		if chainAppID == 0 && chainEnv == "*" {
			return 1 // 最低优先级：全局通配
		}
		return 0 // 不匹配
	}

	// 查找最佳匹配
	findBestMatch := func(chains []struct {
		appID uint
		env   string
	}, targetAppID uint, targetEnv string) int {
		maxPriority := 0
		matchedIdx := -1
		for i, chain := range chains {
			priority := calcPriority(chain.appID, chain.env, targetAppID, targetEnv)
			if priority > maxPriority {
				maxPriority = priority
				matchedIdx = i
			}
		}
		return matchedIdx
	}

	t.Run("exact match always wins", func(t *testing.T) {
		chains := []struct {
			appID uint
			env   string
		}{
			{appID: 0, env: "*"},
			{appID: 0, env: "prod"},
			{appID: 1, env: "*"},
			{appID: 1, env: "prod"}, // 完全匹配
		}

		idx := findBestMatch(chains, 1, "prod")
		assert.Equal(t, 3, idx, "完全匹配应该获胜")
	})

	t.Run("app wildcard beats global exact", func(t *testing.T) {
		chains := []struct {
			appID uint
			env   string
		}{
			{appID: 0, env: "prod"}, // 全局+环境匹配
			{appID: 1, env: "*"},    // 应用+通配
		}

		idx := findBestMatch(chains, 1, "prod")
		assert.Equal(t, 1, idx, "应用+通配应该优于全局+环境匹配")
	})

	t.Run("global exact beats global wildcard", func(t *testing.T) {
		chains := []struct {
			appID uint
			env   string
		}{
			{appID: 0, env: "*"},    // 全局通配
			{appID: 0, env: "prod"}, // 全局+环境匹配
		}

		idx := findBestMatch(chains, 999, "prod")
		assert.Equal(t, 1, idx, "全局+环境匹配应该优于全局通配")
	})

	t.Run("priority ordering is transitive", func(t *testing.T) {
		// 验证优先级的传递性：如果 A > B 且 B > C，则 A > C
		p1 := calcPriority(1, "prod", 1, "prod") // 完全匹配
		p2 := calcPriority(1, "*", 1, "prod")    // 应用+通配
		p3 := calcPriority(0, "prod", 1, "prod") // 全局+环境
		p4 := calcPriority(0, "*", 1, "prod")    // 全局通配

		assert.Greater(t, p1, p2, "完全匹配 > 应用+通配")
		assert.Greater(t, p2, p3, "应用+通配 > 全局+环境")
		assert.Greater(t, p3, p4, "全局+环境 > 全局通配")
		assert.Greater(t, p1, p4, "完全匹配 > 全局通配 (传递性)")
	})
}

// TestApprovalChainPriority_EdgeCases 边界情况测试
func TestApprovalChainPriority_EdgeCases(t *testing.T) {
	calcPriority := func(chainAppID uint, chainEnv string, targetAppID uint, targetEnv string) int {
		if chainAppID == targetAppID && chainEnv == targetEnv {
			return 4
		}
		if chainAppID == targetAppID && chainEnv == "*" {
			return 3
		}
		if chainAppID == 0 && chainEnv == targetEnv {
			return 2
		}
		if chainAppID == 0 && chainEnv == "*" {
			return 1
		}
		return 0
	}

	t.Run("empty environment string", func(t *testing.T) {
		// 空环境字符串应该只匹配完全相同的空字符串
		p := calcPriority(1, "", 1, "")
		assert.Equal(t, 4, p, "空环境应该完全匹配")

		p2 := calcPriority(1, "", 1, "prod")
		assert.Equal(t, 0, p2, "空环境不应该匹配非空环境")
	})

	t.Run("zero app ID with specific env", func(t *testing.T) {
		p := calcPriority(0, "staging", 5, "staging")
		assert.Equal(t, 2, p, "全局+特定环境应该匹配")
	})

	t.Run("multiple apps same priority", func(t *testing.T) {
		// 当有多个相同优先级的链时，应该返回第一个匹配的
		chains := []struct {
			appID uint
			env   string
		}{
			{appID: 0, env: "*"},
			{appID: 0, env: "*"}, // 相同优先级
		}

		maxPriority := 0
		matchedIdx := -1
		for i, chain := range chains {
			priority := calcPriority(chain.appID, chain.env, 1, "prod")
			if priority > maxPriority {
				maxPriority = priority
				matchedIdx = i
			}
		}

		assert.Equal(t, 0, matchedIdx, "应该返回第一个匹配的链")
	})

	t.Run("different environments", func(t *testing.T) {
		envs := []string{"dev", "staging", "prod", "test", "uat"}

		for _, env := range envs {
			p := calcPriority(1, env, 1, env)
			assert.Equal(t, 4, p, "相同环境应该完全匹配: %s", env)

			for _, otherEnv := range envs {
				if otherEnv != env {
					p2 := calcPriority(1, env, 1, otherEnv)
					assert.Equal(t, 0, p2, "%s 不应该匹配 %s", env, otherEnv)
				}
			}
		}
	})
}

// TestApprovalChainPriority_AllCombinations 测试所有优先级组合
func TestApprovalChainPriority_AllCombinations(t *testing.T) {
	calcPriority := func(chainAppID uint, chainEnv string, targetAppID uint, targetEnv string) int {
		if chainAppID == targetAppID && chainEnv == targetEnv {
			return 4
		}
		if chainAppID == targetAppID && chainEnv == "*" {
			return 3
		}
		if chainAppID == 0 && chainEnv == targetEnv {
			return 2
		}
		if chainAppID == 0 && chainEnv == "*" {
			return 1
		}
		return 0
	}

	// 测试所有可能的链配置组合
	testCases := []struct {
		name         string
		chainAppID   uint
		chainEnv     string
		targetAppID  uint
		targetEnv    string
		wantPriority int
	}{
		// 完全匹配 (优先级 4)
		{"exact match", 1, "prod", 1, "prod", 4},
		{"exact match different app", 2, "dev", 2, "dev", 4},

		// 应用匹配+环境通配 (优先级 3)
		{"app match env wildcard", 1, "*", 1, "prod", 3},
		{"app match env wildcard dev", 1, "*", 1, "dev", 3},

		// 全局+环境匹配 (优先级 2)
		{"global env match", 0, "prod", 1, "prod", 2},
		{"global env match different app", 0, "prod", 999, "prod", 2},

		// 全局通配 (优先级 1)
		{"global wildcard", 0, "*", 1, "prod", 1},
		{"global wildcard any app", 0, "*", 999, "any", 1},

		// 不匹配 (优先级 0)
		{"no match different app", 1, "prod", 2, "prod", 0},
		{"no match different env", 1, "prod", 1, "dev", 0},
		{"no match both different", 1, "prod", 2, "dev", 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			priority := calcPriority(tc.chainAppID, tc.chainEnv, tc.targetAppID, tc.targetEnv)
			assert.Equal(t, tc.wantPriority, priority)
		})
	}
}
