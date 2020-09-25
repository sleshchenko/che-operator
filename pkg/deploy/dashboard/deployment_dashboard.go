//
// Copyright (c) 2012-2019 Red Hat, Inc.
// This program and the accompanying materials are made
// available under the terms of the Eclipse Public License 2.0
// which is available at https://www.eclipse.org/legal/epl-2.0/
//
// SPDX-License-Identifier: EPL-2.0
//
// Contributors:
//   Red Hat, Inc. - initial API and implementation
//
package dashboard

import (
	"github.com/eclipse/che-operator/pkg/deploy"
	"strconv"
	"strings"

	orgv1 "github.com/eclipse/che-operator/pkg/apis/org/v1"
	"github.com/eclipse/che-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

func SyncDashboardDeploymentToCluster(deployContext *deploy.DeployContext) deploy.DeploymentProvisioningStatus {
	component := getDashboardComponent(deployContext)
	clusterDeployment, err := deploy.GetClusterDeployment(component, deployContext.CheCluster.Namespace, deployContext.ClusterAPI.Client)
	if err != nil {
		return deploy.DeploymentProvisioningStatus{
			ProvisioningStatus: deploy.ProvisioningStatus{Err: err},
		}
	}

	specDeployment, err := getSpecDashboardDeployment(deployContext)
	if err != nil {
		return deploy.DeploymentProvisioningStatus{
			ProvisioningStatus: deploy.ProvisioningStatus{Err: err},
		}
	}

	return deploy.SyncDeploymentToCluster(deployContext, specDeployment, clusterDeployment, nil, nil)
}

func getDashboardComponent(deployContext *deploy.DeployContext) string {
	cheFlavor := deploy.DefaultCheFlavor(deployContext.CheCluster)
	return cheFlavor + "-dashboard"
}

func getSpecDashboardDeployment(deployContext *deploy.DeployContext) (*appsv1.Deployment, error) {
	isOpenShift, _, err := util.DetectOpenShift()
	if err != nil {
		return nil, err
	}

	terminationGracePeriodSeconds := int64(30)
	cheFlavor := deploy.DefaultCheFlavor(deployContext.CheCluster)
	dashboardComponent := getDashboardComponent(deployContext)
	labels := deploy.GetLabels(deployContext.CheCluster, dashboardComponent)
	memRequest := util.GetValue(deployContext.CheCluster.Spec.Server.ServerMemoryRequest, deploy.DefaultServerMemoryRequest)

	memLimit := util.GetValue(deployContext.CheCluster.Spec.Server.ServerMemoryLimit, deploy.DefaultServerMemoryLimit)
	dashboardImageAndTag := "quay.io/eclipse/che-dashboard:next"
	pullPolicy := corev1.PullPolicy(util.GetValue(string(deployContext.CheCluster.Spec.Server.CheImagePullPolicy), deploy.DefaultPullPolicyFromDockerImage(dashboardImageAndTag)))

	deployment := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      dashboardComponent,
			Namespace: deployContext.CheCluster.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:            cheFlavor,
							ImagePullPolicy: pullPolicy,
							Image:           dashboardImageAndTag,
							Ports: []corev1.ContainerPort{
								{
									Name:          "http",
									ContainerPort: 8080,
									Protocol:      "TCP",
								},
							},
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(memRequest),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceMemory: resource.MustParse(memLimit),
								},
							},
						},
					},
					RestartPolicy:                 "Always",
					TerminationGracePeriodSeconds: &terminationGracePeriodSeconds,
				},
			},
		},
	}

	// configure probes if debug isn't set
	//deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
	//	Handler: corev1.Handler{
	//		HTTPGet: &corev1.HTTPGetAction{
	//			Path: "/api/system/state",
	//			Port: intstr.IntOrString{
	//				Type:   intstr.Int,
	//				IntVal: int32(8080),
	//			},
	//			Scheme: corev1.URISchemeHTTP,
	//		},
	//	},
	//	// After POD start, the POD will be seen as ready after a minimum of 15 seconds and we expect it to be seen as ready until a maximum of 200 seconds
	//	// 200 s = InitialDelaySeconds + PeriodSeconds * (FailureThreshold - 1) + TimeoutSeconds
	//	InitialDelaySeconds: 25,
	//	FailureThreshold:    18,
	//	TimeoutSeconds:      5,
	//	PeriodSeconds:       10,
	//	SuccessThreshold:    1,
	//}
	//deployment.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{
	//	Handler: corev1.Handler{
	//		HTTPGet: &corev1.HTTPGetAction{
	//			Path: "/api/system/state",
	//			Port: intstr.IntOrString{
	//				Type:   intstr.Int,
	//				IntVal: int32(8080),
	//			},
	//			Scheme: corev1.URISchemeHTTP,
	//		},
	//	},
	//	// After POD start, don't initiate liveness probe while the POD is still expected to be declared as ready by the readiness probe
	//	InitialDelaySeconds: 200,
	//	FailureThreshold:    3,
	//	TimeoutSeconds:      3,
	//	PeriodSeconds:       10,
	//	SuccessThreshold:    1,
	//}

	if !isOpenShift {
		runAsUser, err := strconv.ParseInt(util.GetValue(deployContext.CheCluster.Spec.K8s.SecurityContextRunAsUser, deploy.DefaultSecurityContextRunAsUser), 10, 64)
		if err != nil {
			return nil, err
		}
		fsGroup, err := strconv.ParseInt(util.GetValue(deployContext.CheCluster.Spec.K8s.SecurityContextFsGroup, deploy.DefaultSecurityContextFsGroup), 10, 64)
		if err != nil {
			return nil, err
		}
		deployment.Spec.Template.Spec.SecurityContext = &corev1.PodSecurityContext{
			RunAsUser: &runAsUser,
			FSGroup:   &fsGroup,
		}
	}

	if !util.IsTestMode() {
		err = controllerutil.SetControllerReference(deployContext.CheCluster, deployment, deployContext.ClusterAPI.Scheme)
		if err != nil {
			return nil, err
		}
	}

	return deployment, nil
}

// GetFullCheServerImageLink evaluate full cheImage link(with repo and tag)
// based on Checluster information and image defaults from env variables
func GetFullDashboardServerImageLink(checluster *orgv1.CheCluster) string {
	if len(checluster.Spec.Server.CheImage) > 0 {
		cheServerImageTag := util.GetValue(checluster.Spec.Server.CheImageTag, deploy.DefaultCheVersion())
		return checluster.Spec.Server.CheImage + ":" + cheServerImageTag
	}

	defaultCheServerImage := deploy.DefaultCheServerImage(checluster)
	if len(checluster.Spec.Server.CheImageTag) == 0 {
		return defaultCheServerImage
	}

	// For back compatibility with version < 7.9.0:
	// if cr.Spec.Server.CheImage is empty, but cr.Spec.Server.CheImageTag is not empty,
	// parse from default Che image(value comes from env variable) "Che image repository"
	// and return "Che image", like concatenation: "cheImageRepo:cheImageTag"
	separator := map[bool]string{true: "@", false: ":"}[strings.Contains(defaultCheServerImage, "@")]
	imageParts := strings.Split(defaultCheServerImage, separator)
	return imageParts[0] + ":" + checluster.Spec.Server.CheImageTag
}
