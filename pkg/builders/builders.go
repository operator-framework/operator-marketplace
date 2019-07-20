package builders

import (
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v1"
	"github.com/operator-framework/operator-marketplace/pkg/apis/operators/v2"
)

// CscOwnerNameLabel is the label used to indicate the name of the CatalogSourceConfig
// that owns this resources. When this label is set, the reconciler should handle these
// resources when the CatalogSourceConfig is deleted.
const CscOwnerNameLabel string = "csc-owner-name"

// CscOwnerNamespaceLabel is the label used to indicate the namespace of the CatalogSourceConfig
// that owns this resources. When this label is set, the reconciler should handle these
// resources when the CatalogSourceConfig is deleted.
const CscOwnerNamespaceLabel string = "csc-owner-namespace"

// OpsrcOwnerNameLabel is the label used to indicate the name of the OperatorSource
// that owns this resources. When this label is set, the reconciler should handle these
// resources when the OperatorSource is deleted.
const OpsrcOwnerNameLabel string = "opsrc-owner-name"

// OpsrcOwnerNamespaceLabel is the label used to indicate the namespace of the OperatorSource
// that owns this resources. When this label is set, the reconciler should handle these
// resources when the OperatorSource is deleted.
const OpsrcOwnerNamespaceLabel string = "opsrc-owner-namespace"

// GetOwnerLabel returns a map with either the CatalogSourceConfig or the OperatorSource owner
// name and namespace labels depending what kind is the owner
func GetOwnerLabel(name, namespace, owner string) map[string]string {
	switch owner {
	case v1.OperatorSourceKind:
		return map[string]string{
			OpsrcOwnerNameLabel:      name,
			OpsrcOwnerNamespaceLabel: namespace,
		}
	case v2.CatalogSourceConfigKind:
		return map[string]string{
			CscOwnerNameLabel:      name,
			CscOwnerNamespaceLabel: namespace,
		}
	default:
		return map[string]string{}
	}
}

// HasOwnerLabels determines whether owner labels are present in a given set of labels
func HasOwnerLabels(labels map[string]string, owner string) bool {
	switch owner {
	case v1.OperatorSourceKind:
		_, hasNameLabel := labels[OpsrcOwnerNameLabel]
		_, hasNamespaceLabel := labels[OpsrcOwnerNamespaceLabel]
		return hasNameLabel && hasNamespaceLabel
	case v2.CatalogSourceConfigKind:
		_, hasNameLabel := labels[CscOwnerNameLabel]
		_, hasNamespaceLabel := labels[CscOwnerNamespaceLabel]
		return hasNameLabel && hasNamespaceLabel
	default:
		return false
	}
}
