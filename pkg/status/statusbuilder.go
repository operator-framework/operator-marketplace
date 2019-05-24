package status

import (
	configv1 "github.com/openshift/api/config/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1"
)

// StatusBuilder builds a new slice of ClusterOperatorStatusConditions.
type StatusBuilder struct {
	statusList []configv1.ClusterOperatorStatusCondition
}

// StatusList returns a slice of ClusterOperatorStatusConditions.
func (s *StatusBuilder) StatusList() *[]configv1.ClusterOperatorStatusCondition {
	return &s.statusList
}

func (s *StatusBuilder) WithStatus(conditionType configv1.ClusterStatusConditionType, conditionStatus configv1.ConditionStatus, conditionMessage string) *StatusBuilder {
	time := v1.Now()
	s.statusList = append(s.statusList, configv1.ClusterOperatorStatusCondition{
		Type:               conditionType,
		Status:             conditionStatus,
		Message:            conditionMessage,
		LastTransitionTime: time,
	})

	return s
}
