package cost

// 辅助函数

func getTrendString(factor float64) string {
	if factor > 1.05 {
		return "up"
	} else if factor < 0.95 {
		return "down"
	}
	return "stable"
}

func getImpactLevel(cost float64) string {
	if cost > 100 {
		return "high"
	} else if cost > 50 {
		return "medium"
	}
	return "low"
}

func getScoreStatus(score, maxScore int) string {
	ratio := float64(score) / float64(maxScore)
	if ratio >= 0.8 {
		return "good"
	} else if ratio >= 0.5 {
		return "warning"
	}
	return "critical"
}

func getTrendFromScore(score int) string {
	if score >= 80 {
		return "stable"
	} else if score >= 60 {
		return "improving"
	}
	return "declining"
}
