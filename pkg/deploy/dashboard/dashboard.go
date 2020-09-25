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
	"github.com/eclipse/che-operator/pkg/deploy/expose"
	"github.com/eclipse/che-operator/pkg/util"
	"github.com/sirupsen/logrus"
)

const (
	Dashboard = "dashboard"
)

/**
 * Create dashboard resources.
 */
func SyncDashboardToCluster(deployContext *deploy.DeployContext, cheHost string) (done bool, err error) {
	// Deploy plugin registry
	deploymentStatus := SyncDashboardDeploymentToCluster(deployContext)
	if !util.IsTestMode() {
		if !deploymentStatus.Continue {
			logrus.Info("Waiting on deployment '" + Dashboard + "' to be ready")
			if deploymentStatus.Err != nil {
				logrus.Error(deploymentStatus.Err)
			}

			return false, deploymentStatus.Err
		}
	}

	// Create a new registry service
	dashboardComponent := getDashboardComponent(deployContext)
	dashboardLabels := deploy.GetLabels(deployContext.CheCluster, dashboardComponent)
	serviceStatus := deploy.SyncServiceToCluster(deployContext, dashboardComponent, []string{"http"}, []int32{8080}, dashboardLabels)
	if !util.IsTestMode() {
		if !serviceStatus.Continue {
			logrus.Info("Waiting on service '" + deploy.PluginRegistry + "' to be ready")
			if serviceStatus.Err != nil {
				logrus.Error(serviceStatus.Err)
			}

			return false, serviceStatus.Err
		}
	}

	_, done, err = expose.ExposeWithHost(deployContext, cheHost, dashboardComponent, cheHost, "/dashboard")

	return done, err
}
