package cohort

import (
	"testing"
)

func TestEvaluate_GlobMatch(t *testing.T) {
	rule := CohortRule{
		Operator: "AND",
		Conditions: []RuleCondition{
			{Field: "device_model", Op: "glob", Value: "Pixel*"},
		},
	}
	attrs := map[string]string{"device_model": "Pixel 7 Pro"}
	if !Evaluate(rule, attrs) {
		t.Error("Pixel 7 Pro should match Pixel*")
	}
}

func TestEvaluate_GlobNoMatch(t *testing.T) {
	rule := CohortRule{
		Operator: "AND",
		Conditions: []RuleCondition{
			{Field: "device_model", Op: "glob", Value: "Pixel*"},
		},
	}
	attrs := map[string]string{"device_model": "Samsung Galaxy S24"}
	if Evaluate(rule, attrs) {
		t.Error("Samsung should not match Pixel*")
	}
}

func TestEvaluate_SemverRange(t *testing.T) {
	rule := CohortRule{
		Operator: "AND",
		Conditions: []RuleCondition{
			{Field: "os_version", Op: "semver_range", Value: ">=14.0 <15.0"},
		},
	}
	if !Evaluate(rule, map[string]string{"os_version": "14.0"}) {
		t.Error("14.0 should be in range >=14.0 <15.0")
	}
	if !Evaluate(rule, map[string]string{"os_version": "14.5.1"}) {
		t.Error("14.5.1 should be in range")
	}
	if Evaluate(rule, map[string]string{"os_version": "15.0"}) {
		t.Error("15.0 should NOT be in range")
	}
	if Evaluate(rule, map[string]string{"os_version": "13.9"}) {
		t.Error("13.9 should NOT be in range")
	}
}

func TestEvaluate_NestedAndOr(t *testing.T) {
	rule := CohortRule{
		Operator: "AND",
		Conditions: []RuleCondition{
			{Field: "device_model", Op: "glob", Value: "Pixel*"},
		},
		Children: []CohortRule{
			{
				Operator: "OR",
				Conditions: []RuleCondition{
					{Field: "locale", Op: "prefix", Value: "en_US"},
					{Field: "locale", Op: "prefix", Value: "en_GB"},
				},
			},
		},
	}
	if !Evaluate(rule, map[string]string{"device_model": "Pixel 7", "locale": "en_US_POSIX"}) {
		t.Error("Pixel 7 + en_US should match")
	}
	if Evaluate(rule, map[string]string{"device_model": "Pixel 7", "locale": "de_DE"}) {
		t.Error("Pixel 7 + de_DE should NOT match (locale fails OR)")
	}
}

func TestEvaluate_MissingAttribute_NoMatch(t *testing.T) {
	rule := CohortRule{
		Operator: "AND",
		Conditions: []RuleCondition{
			{Field: "carrier", Op: "equals", Value: "Verizon"},
		},
	}
	if Evaluate(rule, map[string]string{"device_model": "Pixel 7"}) {
		t.Error("Missing attribute should not match")
	}
}

func TestEvaluate_EmptyConditions_MatchesAll(t *testing.T) {
	rule := CohortRule{Operator: "AND", Conditions: []RuleCondition{}}
	if !Evaluate(rule, map[string]string{"anything": "value"}) {
		t.Error("Empty conditions should match all devices")
	}
}

func TestEvaluate_NotOperator(t *testing.T) {
	rule := CohortRule{
		Operator: "NOT",
		Conditions: []RuleCondition{
			{Field: "device_model", Op: "glob", Value: "Samsung*"},
		},
	}
	if !Evaluate(rule, map[string]string{"device_model": "Pixel 7"}) {
		t.Error("NOT Samsung should match Pixel")
	}
	if Evaluate(rule, map[string]string{"device_model": "Samsung Galaxy"}) {
		t.Error("NOT Samsung should NOT match Samsung")
	}
}

func TestEvaluate_Equals(t *testing.T) {
	rule := CohortRule{
		Operator: "AND",
		Conditions: []RuleCondition{
			{Field: "app_version", Op: "equals", Value: "3.2.1"},
		},
	}
	if !Evaluate(rule, map[string]string{"app_version": "3.2.1"}) {
		t.Error("Exact match should work")
	}
	if Evaluate(rule, map[string]string{"app_version": "3.2.2"}) {
		t.Error("Different version should not match")
	}
}

func TestEvaluate_PrivacyMinSize(t *testing.T) {
	err := ValidateCohortRule(CohortRule{
		Operator: "AND",
		Conditions: []RuleCondition{
			{Field: "device_id", Op: "equals", Value: "specific-device"},
		},
	})
	if err == nil {
		t.Error("device_id equals targeting should be rejected for privacy")
	}
}
