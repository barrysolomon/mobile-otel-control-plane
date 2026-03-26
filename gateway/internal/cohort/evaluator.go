package cohort

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// Evaluate checks if a device's attributes match a cohort rule tree.
func Evaluate(rule CohortRule, attrs map[string]string) bool {
	switch rule.Operator {
	case "AND":
		return evaluateAND(rule, attrs)
	case "OR":
		return evaluateOR(rule, attrs)
	case "NOT":
		return !evaluateAND(CohortRule{Operator: "AND", Conditions: rule.Conditions, Children: rule.Children}, attrs)
	default:
		return evaluateAND(rule, attrs)
	}
}

func evaluateAND(rule CohortRule, attrs map[string]string) bool {
	for _, cond := range rule.Conditions {
		if !evaluateCondition(cond, attrs) {
			return false
		}
	}
	for _, child := range rule.Children {
		if !Evaluate(child, attrs) {
			return false
		}
	}
	return true
}

func evaluateOR(rule CohortRule, attrs map[string]string) bool {
	if len(rule.Conditions) == 0 && len(rule.Children) == 0 {
		return true
	}
	for _, cond := range rule.Conditions {
		if evaluateCondition(cond, attrs) {
			return true
		}
	}
	for _, child := range rule.Children {
		if Evaluate(child, attrs) {
			return true
		}
	}
	return false
}

func evaluateCondition(cond RuleCondition, attrs map[string]string) bool {
	val, exists := attrs[cond.Field]
	if !exists {
		return false
	}

	switch cond.Op {
	case "equals":
		return val == cond.Value
	case "glob":
		matched, _ := filepath.Match(cond.Value, val)
		return matched
	case "prefix":
		return strings.HasPrefix(val, cond.Value)
	case "contains":
		return strings.Contains(val, cond.Value)
	case "in":
		for _, item := range strings.Split(cond.Value, ",") {
			if strings.TrimSpace(item) == val {
				return true
			}
		}
		return false
	case "lt", "gt":
		return compareNumeric(val, cond.Value, cond.Op)
	case "semver_range":
		return evaluateSemverRange(val, cond.Value)
	default:
		return false
	}
}

func compareNumeric(val, target, op string) bool {
	v, err1 := strconv.ParseFloat(val, 64)
	t, err2 := strconv.ParseFloat(target, 64)
	if err1 != nil || err2 != nil {
		return false
	}
	switch op {
	case "lt":
		return v < t
	case "gt":
		return v > t
	default:
		return false
	}
}

// evaluateSemverRange handles ">=14.0 <15.0" style ranges.
func evaluateSemverRange(version, rangeExpr string) bool {
	parts := strings.Fields(rangeExpr)
	ver := parseSemver(version)
	if ver == nil {
		return false
	}

	for _, part := range parts {
		if strings.HasPrefix(part, ">=") {
			target := parseSemver(strings.TrimPrefix(part, ">="))
			if target == nil || compareSemver(ver, target) < 0 {
				return false
			}
		} else if strings.HasPrefix(part, ">") {
			target := parseSemver(strings.TrimPrefix(part, ">"))
			if target == nil || compareSemver(ver, target) <= 0 {
				return false
			}
		} else if strings.HasPrefix(part, "<=") {
			target := parseSemver(strings.TrimPrefix(part, "<="))
			if target == nil || compareSemver(ver, target) > 0 {
				return false
			}
		} else if strings.HasPrefix(part, "<") {
			target := parseSemver(strings.TrimPrefix(part, "<"))
			if target == nil || compareSemver(ver, target) >= 0 {
				return false
			}
		}
	}
	return true
}

type semver struct {
	major, minor, patch int
}

func parseSemver(s string) *semver {
	if idx := strings.Index(s, "-"); idx >= 0 {
		s = s[:idx]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 {
		return nil
	}
	v := &semver{}
	v.major, _ = strconv.Atoi(parts[0])
	if len(parts) > 1 {
		v.minor, _ = strconv.Atoi(parts[1])
	}
	if len(parts) > 2 {
		v.patch, _ = strconv.Atoi(parts[2])
	}
	return v
}

func compareSemver(a, b *semver) int {
	if a.major != b.major {
		return a.major - b.major
	}
	if a.minor != b.minor {
		return a.minor - b.minor
	}
	return a.patch - b.patch
}

// ValidateCohortRule checks privacy and structural constraints.
func ValidateCohortRule(rule CohortRule) error {
	for _, cond := range rule.Conditions {
		if cond.Field == "device_id" && cond.Op == "equals" {
			return fmt.Errorf("targeting individual device_id not allowed in fleet cohorts (privacy)")
		}
	}
	for _, child := range rule.Children {
		if err := ValidateCohortRule(child); err != nil {
			return err
		}
	}
	return nil
}
