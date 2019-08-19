package certificateauthority

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// TrustedCaConfigMapName is the name of the Marketplace ConfigMap that store Certificate Authority information.
	TrustedCaConfigMapName = "marketplace-trusted-ca"

	// TrustedCaMountPath is the path to the directory where the Certificate Authority should be mounted.
	TrustedCaMountPath = "/etc/pki/ca-trust/extracted/pem/"

	// The key value that stores Certificate Authorities.
	caBundleKey = "ca-bundle.crt"

	// The path where we will mount the Certificate Authorities.
	caBundlePath = "tls-ca-bundle.pem"
)

// MountConfigMap creates a Volume and VolumeMount for a ConfigMap of the same name and
// adds it to a deployment.
func MountConfigMap(name, mountPath string, deployment *appsv1.Deployment) {
	// Create and add the Volume to the deployment.
	deployment.Spec.Template.Spec.Volumes = []corev1.Volume{
		corev1.Volume{
			Name: name,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: name,
					},
					Items: []corev1.KeyToPath{
						corev1.KeyToPath{
							Key:  caBundleKey,
							Path: caBundlePath,
						},
					},
				},
			},
		},
	}

	// Create and add the VolumeMount to the first container in a deployment.
	deployment.Spec.Template.Spec.Containers[0].VolumeMounts = []corev1.VolumeMount{
		corev1.VolumeMount{
			Name:      name,
			MountPath: mountPath,
		},
	}
}
