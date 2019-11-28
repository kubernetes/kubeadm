/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"os"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/klog"
	"k8s.io/klog/klogr"
	ctrl "sigs.k8s.io/controller-runtime"

	operatorv1 "k8s.io/kubeadm/operator/api/v1alpha1"
	"k8s.io/kubeadm/operator/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)

	_ = operatorv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

type managerMode string

const (
	modeManager = managerMode("manager")
	modeAgent   = managerMode("agent")
)

func main() {
	klog.InitFlags(nil)
	var mode string
	var pod string
	var namespace string
	var image string
	var nodeName string
	var operation string
	var metricsAddr string
	var metricsRBAC bool
	var enableLeaderElection bool

	// common flags
	flag.StringVar(&mode, "mode", string(modeManager), "One of [manger, agent]")
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to")

	// manager flags
	flag.StringVar(&pod, "manager-pod", "", "The pod the manager is running in")
	flag.StringVar(&namespace, "manager-namespace", "", "The namespace the manager is running in")                                               //TODO: implement in all the controllers
	flag.StringVar(&image, "agent-image", "", "The image that should be used for agent the DaemonSet. If empty, the manager image will be used") //TODO: remove; always use manager image
	flag.BoolVar(&metricsRBAC, "agent-metrics-rbac", true, "Use RBAC authn/z for the /metrics endpoint of agents")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager")

	// agent flags
	flag.StringVar(&nodeName, "agent-node-name", "", "The node that the agent manager should control")
	flag.StringVar(&operation, "agent-operation", "", "The operation that the agent manager should control. If empty, the agent will control headless Task only")

	flag.Parse()

	ctrl.SetLogger(klogr.New())

	if managerMode(mode) != modeManager && managerMode(mode) != modeAgent {
		setupLog.Error(errors.New("invalid value"), "unable to create controllers with an invalid --mode value")
		os.Exit(1)
	}

	if managerMode(mode) == modeAgent {
		enableLeaderElection = false
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		LeaderElection:     enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if managerMode(mode) == modeManager {
		if err = (&controllers.OperationReconciler{
			Client:               mgr.GetClient(),
			ManagerContainerName: pod,
			ManagerNamespace:     namespace,
			AgentImage:           image,
			MetricsRBAC:          metricsRBAC,
			Log:                  ctrl.Log.WithName("controllers").WithName("Operation"),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Operation")
			os.Exit(1)
		}

		if err = (&controllers.RuntimeTaskGroupReconciler{
			Client: mgr.GetClient(),
			Log:    ctrl.Log.WithName("controllers").WithName("RuntimeTaskGroup"),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "RuntimeTaskGroup")
			os.Exit(1)
		}

		setupLog.Info("starting controller manager", "manager-pod", pod, "manager-namespace", namespace, "agent-image", image, "agent-metrics-RBAC", metricsRBAC)
	}

	if managerMode(mode) == modeAgent {
		if nodeName == "" {
			setupLog.Error(err, "unable to create controller without the --agent-node-name value set", "controller", "RuntimeTask")
			os.Exit(1)
		}
		if nodeName == "" {
			setupLog.Error(err, "unable to create controller without the --agent-operation value set", "controller", "RuntimeTask")
			os.Exit(1)
		}

		if err = (&controllers.RuntimeTaskReconciler{
			Client:    mgr.GetClient(),
			NodeName:  nodeName,
			Operation: operation,
			Log:       ctrl.Log.WithName("controllers").WithName("RuntimeTask").WithValues("node-name", nodeName),
		}).SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "RuntimeTask")
			os.Exit(1)
		}
		setupLog.Info("starting agent manager", "agent-node", nodeName, "agent-operation", operation)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
