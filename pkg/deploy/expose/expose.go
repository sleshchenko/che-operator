package expose

import (
	"github.com/eclipse/che-operator/pkg/deploy"
	"github.com/eclipse/che-operator/pkg/deploy/gateway"
	"github.com/eclipse/che-operator/pkg/util"
	"github.com/sirupsen/logrus"
)

func Expose(deployContext *deploy.DeployContext, cheHost string, endpointName string) (endpont string, done bool, err error) {
	return ExposeWithHost(deployContext, cheHost, endpointName, "", "")
}

func ExposeWithHost(deployContext *deploy.DeployContext, cheHost string, serviceName string, host string, path string) (endpoint string, done bool, err error) {
	exposureStrategy := util.GetServerExposureStrategy(deployContext.CheCluster, deploy.DefaultServerExposureStrategy)
	var domain string
	if exposureStrategy == "multi-host" {
		// this won't get used on openshift, because there we're intentionally let Openshift decide on the domain name
		domain = host + "-" + deployContext.CheCluster.Namespace + "." + deployContext.CheCluster.Spec.K8s.IngressDomain
		endpoint = domain
	} else {
		domain = cheHost
		endpoint = domain + "/" + host
	}

	gatewayConfig := "che-gateway-route-" + serviceName
	singleHostExposureType := deploy.GetSingleHostExposureType(deployContext.CheCluster)
	useGateway := exposureStrategy == "single-host" && (util.IsOpenShift || singleHostExposureType == "gateway")

	if !util.IsOpenShift {
		if useGateway {
			cfg := gateway.GetGatewayRouteConfig(deployContext, gatewayConfig, "/"+host, 10, "http://"+serviceName+":8080", true)
			clusterCfg, err := deploy.SyncConfigMapToCluster(deployContext, &cfg)
			if !util.IsTestMode() {
				if clusterCfg == nil {
					if err != nil {
						logrus.Error(err)
					}
					return "", false, err
				}
			}
			if err := deploy.DeleteIngressIfExists(serviceName, deployContext); !util.IsTestMode() && err != nil {
				logrus.Error(err)
			}
		} else {
			additionalLabels := deployContext.CheCluster.Spec.Server.PluginRegistryIngress.Labels
			//TODO Set custom path
			ingress, err := deploy.SyncIngressToCluster(deployContext, serviceName, domain, serviceName, 8080, additionalLabels)
			if !util.IsTestMode() {
				if ingress == nil {
					logrus.Infof("Waiting on ingress '%s' to be ready", serviceName)
					if err != nil {
						logrus.Error(err)
					}
					return "", false, err
				}
			}
			if err := gateway.DeleteGatewayRouteConfig(gatewayConfig, deployContext); !util.IsTestMode() && err != nil {
				logrus.Error(err)
			}
		}
	} else {
		if useGateway {
			cfg := gateway.GetGatewayRouteConfig(deployContext, gatewayConfig, "/"+host, 10, "http://"+serviceName+":8080", true)
			clusterCfg, err := deploy.SyncConfigMapToCluster(deployContext, &cfg)
			if !util.IsTestMode() {
				if clusterCfg == nil {
					if err != nil {
						logrus.Error(err)
					}
					return "", false, err
				}
			}
			if err := deploy.DeleteRouteIfExists(serviceName, deployContext); !util.IsTestMode() && err != nil {
				logrus.Error(err)
			}
		} else {
			// the empty string for a host is intentional here - we let OpenShift decide on the hostname
			additionalLabels := deployContext.CheCluster.Spec.Server.PluginRegistryIngress.Labels
			route, err := deploy.SyncRouteWithPathToCluster(deployContext, serviceName, host, path, serviceName, 8080, additionalLabels)
			if !util.IsTestMode() {
				if route == nil {
					logrus.Infof("Waiting on route '%s' to be ready", serviceName)
					if err != nil {
						logrus.Error(err)
					}

					return "", false, err
				}
			}
			if err := gateway.DeleteGatewayRouteConfig(gatewayConfig, deployContext); !util.IsTestMode() && err != nil {
				logrus.Error(err)
			}
			if !util.IsTestMode() {
				endpoint = route.Spec.Host
			}
		}
	}
	return endpoint, true, nil
}
