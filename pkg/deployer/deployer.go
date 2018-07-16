/*
Copyright (C) 2018 Synopsys, Inc.

Licensed to the Apache Software Foundation (ASF) under one
or more contributor license agreements. See the NOTICE file
distributed with this work for additional information
regarding copyright ownership. The ASF licenses this file
to you under the Apache License, Version 2.0 (the
"License"); you may not use this file except in compliance
with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing,
software distributed under the License is distributed on an
"AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
KIND, either express or implied. See the License for the
specific language governing permissions and limitations
under the License.
*/

package deployer

import (
	"github.com/blackducksoftware/cn-crd-controller/pkg/api"
	"github.com/blackducksoftware/cn-crd-controller/pkg/types"
	"github.com/blackducksoftware/cn-crd-controller/pkg/utils"
	utilserror "github.com/blackducksoftware/cn-crd-controller/pkg/utils/error"

	"github.com/koki/short/converter/converters"
	shorttypes "github.com/koki/short/types"

	"k8s.io/client-go/kubernetes"

	extensionsclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"

	log "github.com/sirupsen/logrus"
)

// Deployer handles deploying the components to a cluster
type Deployer struct {
	replicationControllers map[string]*shorttypes.ReplicationController
	pods                   map[string]*shorttypes.Pod
	configMaps             map[string]*shorttypes.ConfigMap
	secrets                map[string]*shorttypes.Secret
	services               map[string]*shorttypes.Service
	serviceAccounts        map[string]*shorttypes.ServiceAccount
	deployments            map[string]*shorttypes.Deployment
	clusterRoles           map[string]*shorttypes.ClusterRole
	clusterRoleBindings    map[string]*shorttypes.ClusterRoleBinding
	crds                   map[string]*shorttypes.CustomResourceDefinition
	namespaces             map[string]*shorttypes.Namespace

	controllers map[string]api.DeployerControllerInterface

	client        *kubernetes.Clientset
	apiextensions *extensionsclient.Clientset
}

// NewDeployer creates a Deployer object
func NewDeployer(client *kubernetes.Clientset, apiextensions *extensionsclient.Clientset) *Deployer {
	d := Deployer{
		client:                 client,
		apiextensions:          apiextensions,
		replicationControllers: make(map[string]*shorttypes.ReplicationController),
		pods:                make(map[string]*shorttypes.Pod),
		configMaps:          make(map[string]*shorttypes.ConfigMap),
		secrets:             make(map[string]*shorttypes.Secret),
		services:            make(map[string]*shorttypes.Service),
		serviceAccounts:     make(map[string]*shorttypes.ServiceAccount),
		deployments:         make(map[string]*shorttypes.Deployment),
		clusterRoles:        make(map[string]*shorttypes.ClusterRole),
		clusterRoleBindings: make(map[string]*shorttypes.ClusterRoleBinding),
		crds:                make(map[string]*shorttypes.CustomResourceDefinition),
		namespaces:          make(map[string]*shorttypes.Namespace),
		controllers:         make(map[string]api.DeployerControllerInterface),
	}
	return &d
}

// AddController will add a custom controller that will be run after all
// components have been deployed.
func (d *Deployer) AddController(name string, c api.DeployerControllerInterface) {
	d.controllers[name] = c
}

// AddConfigMap will add the provided config map to the config maps
// that will be deployed
func (d *Deployer) AddConfigMap(obj *types.ConfigMap) {
	d.configMaps[obj.GetName()] = obj.GetObj()
}

// AddDeployment will add the provided deployment to the deployments
// that will be deployed
func (d *Deployer) AddDeployment(obj *types.Deployment) {
	d.deployments[obj.GetName()] = obj.GetObj()
}

// AddService will add the provided service to the services
// that will be deployed
func (d *Deployer) AddService(obj *types.Service) {
	d.services[obj.GetName()] = obj.GetObj()
}

// AddSecret will add the provided secret to the secrets
// that will be deployed
func (d *Deployer) AddSecret(obj *types.Secret) {
	d.secrets[obj.GetName()] = obj.GetObj()
}

// AddClusterRole will add the provided cluster role to the
// cluster roles that will be deployed
func (d *Deployer) AddClusterRole(obj *types.ClusterRole) {
	d.clusterRoles[obj.GetName()] = obj.GetObj()
}

// AddClusterRoleBinding will add the provided cluster role binding
// to the cluster role bindings that will be deployed
func (d *Deployer) AddClusterRoleBinding(obj *types.ClusterRoleBinding) {
	d.clusterRoleBindings[obj.GetName()] = obj.GetObj()
}

// AddCustomDefinedResource will add the provided custom defined resource
// to the custom defined resources that will be deployed
func (d *Deployer) AddCustomDefinedResource(obj *types.CustomResourceDefinition) {
	d.crds[obj.GetName()] = obj.GetObj()
}

// AddReplicationConroller will add the provided replication controller
// to the replication controllers that will be deployed
func (d *Deployer) AddReplicationConroller(obj *types.ReplicationController) {
	d.replicationControllers[obj.GetName()] = obj.GetObj()
}

// AddNamespace will add the provided namespace to the
// namespaces that will be deployed
func (d *Deployer) AddNamespace(obj *types.Namespace) {
	d.namespaces[obj.GetName()] = obj.GetObj()
}

// Run starts the deployer and deploys all components to the cluster
func (d *Deployer) Run() error {
	allErrs := map[utils.ComponentType][]error{}

	err := d.deployNamespaces()
	if len(err) > 0 {
		allErrs[utils.NamespaceComponent] = err
	}

	err = d.deployCRDs()
	if len(err) > 0 {
		allErrs[utils.CRDComponent] = err
	}

	err = d.deployServiceAccounts()
	if len(err) > 0 {
		allErrs[utils.ServiceAccountComponent] = err
	}

	errMap := d.deployRBAC()
	if len(errMap) > 0 {
		for k, v := range errMap {
			allErrs[k] = v
		}
	}

	err = d.deployConfigMaps()
	if len(err) > 0 {
		allErrs[utils.ConfigMapComponent] = err
	}

	err = d.deploySecrets()
	if len(err) > 0 {
		allErrs[utils.SecretComponent] = err
	}

	err = d.deployReplicationControllers()
	if len(err) > 0 {
		allErrs[utils.ReplicationControllerComponent] = err
	}

	err = d.deployPods()
	if len(err) > 0 {
		allErrs[utils.PodComponent] = err
	}

	err = d.deployDeployments()
	if len(err) > 0 {
		allErrs[utils.DeploymentComponent] = err
	}

	err = d.deployServices()
	if len(err) > 0 {
		allErrs[utils.ServiceComponent] = err
	}

	return utilserror.NewDeployErrors(allErrs)
}

// StartControllers will start all the configured controllers
func (d *Deployer) StartControllers(stopCh chan struct{}) map[string][]error {
	errs := make(map[string][]error)

	// Run the controllers if there are any configured
	if len(d.controllers) > 0 {
		errCh := make(chan map[string]error)
		for n, c := range d.controllers {
			go func(name string, controller api.DeployerControllerInterface) {
				err := controller.Run(stopCh)
				if err != nil {
					errCh <- map[string]error{name: err}
				}
			}(n, c)
		}

	controllerRun:
		for {
			select {
			case e := <-errCh:
				for k, v := range e {
					errs[k] = append(errs[k], v)
				}
			case <-stopCh:
				break controllerRun
			}
		}
	}
	return errs
}

func (d *Deployer) deployCRDs() []error {
	errs := []error{}

	for name, crdObj := range d.crds {
		wrapper := &shorttypes.CRDWrapper{CRD: *crdObj}
		crd, err := converters.Convert_Koki_CRD_to_Kube(wrapper)
		if err != nil {
			errs = append(errs, err)
		}
		log.Infof("Creating custom defined resource %s", name)
		_, err = d.apiextensions.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (d *Deployer) deployServiceAccounts() []error {
	errs := []error{}

	for name, saObj := range d.serviceAccounts {
		wrapper := &shorttypes.ServiceAccountWrapper{ServiceAccount: *saObj}
		sa, err := converters.Convert_Koki_ServiceAccount_to_Kube_ServiceAccount(wrapper)
		if err != nil {
			errs = append(errs, err)
		}
		log.Infof("Creating service account %s", name)
		_, err = d.client.Core().ServiceAccounts(sa.Namespace).Create(sa)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (d *Deployer) deployRBAC() map[utils.ComponentType][]error {
	errs := map[utils.ComponentType][]error{}

	for name, crObj := range d.clusterRoles {
		wrapper := &shorttypes.ClusterRoleWrapper{ClusterRole: *crObj}
		cr, err := converters.Convert_Koki_ClusterRole_to_Kube(wrapper)
		if err != nil {
			errs[utils.ClusterRoleComponent] = append(errs[utils.ClusterRoleComponent], err)
		}
		log.Infof("Creating cluster role %s", name)
		_, err = d.client.Rbac().ClusterRoles().Create(cr)
		if err != nil {
			errs[utils.ClusterRoleComponent] = append(errs[utils.ClusterRoleComponent], err)
		}
	}

	for name, crbObj := range d.clusterRoleBindings {
		wrapper := &shorttypes.ClusterRoleBindingWrapper{ClusterRoleBinding: *crbObj}
		crb, err := converters.Convert_Koki_ClusterRoleBinding_to_Kube(wrapper)
		if err != nil {
			errs[utils.ClusterRoleBindingComponent] = append(errs[utils.ClusterRoleComponent], err)
		}
		log.Infof("Creating cluster role binding %s", name)
		_, err = d.client.Rbac().ClusterRoleBindings().Create(crb)
		if err != nil {
			errs[utils.ClusterRoleBindingComponent] = append(errs[utils.ClusterRoleComponent], err)
		}
	}
	return errs
}

func (d *Deployer) deployConfigMaps() []error {
	errs := []error{}

	for name, cmObj := range d.configMaps {
		wrapper := &shorttypes.ConfigMapWrapper{ConfigMap: *cmObj}
		cm, err := converters.Convert_Koki_ConfigMap_to_Kube_v1_ConfigMap(wrapper)
		if err != nil {
			errs = append(errs, err)
		}
		log.Infof("Creating config map %s", name)
		_, err = d.client.Core().ConfigMaps(cm.Namespace).Create(cm)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (d *Deployer) deploySecrets() []error {
	errs := []error{}

	for name, secretObj := range d.secrets {
		wrapper := &shorttypes.SecretWrapper{Secret: *secretObj}
		secret, err := converters.Convert_Koki_Secret_to_Kube_v1_Secret(wrapper)
		if err != nil {
			errs = append(errs, err)
		}
		log.Infof("Creating secret %s", name)
		_, err = d.client.Core().Secrets(secret.Namespace).Create(secret)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (d *Deployer) deployReplicationControllers() []error {
	errs := []error{}

	for name, rcObj := range d.replicationControllers {
		wrapper := &shorttypes.ReplicationControllerWrapper{ReplicationController: *rcObj}
		rc, err := converters.Convert_Koki_ReplicationController_to_Kube_v1_ReplicationController(wrapper)
		if err != nil {
			errs = append(errs, err)
		}

		log.Infof("Creating replication controller %s", name)
		_, err = d.client.Core().ReplicationControllers(rc.Namespace).Create(rc)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (d *Deployer) deployPods() []error {
	errs := []error{}

	for name, pObj := range d.pods {
		wrapper := &shorttypes.PodWrapper{Pod: *pObj}
		pod, err := converters.Convert_Koki_Pod_to_Kube_v1_Pod(wrapper)
		if err != nil {
			errs = append(errs, err)
		}

		log.Infof("Creating pod %s", name)
		_, err = d.client.Core().Pods(pod.Namespace).Create(pod)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (d *Deployer) deployDeployments() []error {
	errs := []error{}

	for name, dObj := range d.deployments {
		wrapper := &shorttypes.DeploymentWrapper{Deployment: *dObj}
		deploy, err := converters.Convert_Koki_Deployment_to_Kube_apps_v1beta2_Deployment(wrapper)
		if err != nil {
			errs = append(errs, err)
		}

		log.Infof("Creating deployment %s", name)
		_, err = d.client.AppsV1beta2().Deployments(deploy.Namespace).Create(deploy)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (d *Deployer) deployServices() []error {
	errs := []error{}

	for name, svcObj := range d.services {
		sWrapper := &shorttypes.ServiceWrapper{Service: *svcObj}
		svc, err := converters.Convert_Koki_Service_To_Kube_v1_Service(sWrapper)
		if err != nil {
			errs = append(errs, err)
		}

		log.Infof("Creating service %s", name)
		_, err = d.client.Core().Services(svc.Namespace).Create(svc)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}

func (d *Deployer) deployNamespaces() []error {
	errs := []error{}

	for name, nsObj := range d.namespaces {
		wrapper := &shorttypes.NamespaceWrapper{Namespace: *nsObj}
		ns, err := converters.Convert_Koki_Namespace_to_Kube_Namespace(wrapper)
		if err != nil {
			errs = append(errs, err)
		}
		log.Infof("Creating namespace %s", name)
		_, err = d.client.Core().Namespaces().Create(ns)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}
