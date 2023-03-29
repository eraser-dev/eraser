/*
Copyright 2021.

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
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/utils/inotify"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	eraserv1 "github.com/Azure/eraser/api/v1"
	eraserv1alpha1 "github.com/Azure/eraser/api/v1alpha1"
	"github.com/Azure/eraser/api/v1alpha1/config"
	eraserv1alpha2 "github.com/Azure/eraser/api/v1alpha2"
	"github.com/Azure/eraser/controllers"
	"github.com/Azure/eraser/pkg/logger"
	"github.com/Azure/eraser/version"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(eraserv1alpha1.AddToScheme(scheme))
	utilruntime.Must(eraserv1.AddToScheme(scheme))
	utilruntime.Must(eraserv1alpha2.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "config", "",
		"The controller will load its initial configuration from this file. "+
			"Omit this flag to use the default configuration values. "+
			"Command-line flags override configuration from this file.")
	flag.Parse()

	if err := logger.Configure(); err != nil {
		setupLog.Error(err, "unable to configure logger")
		os.Exit(1)
	}

	// these can all be overwritten using EraserConfig.
	options := ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     ":8889",
		Port:                   9443,
		HealthProbeBindAddress: ":8081",
		LeaderElection:         false,
	}

	if configFile == "" {
		setupLog.Error(fmt.Errorf("config file was not supplied"), "aborting")
		os.Exit(1)
	}

	cfg := config.Default()
	if configFile != "" {
		o, err := options.AndFrom(ctrl.ConfigFile().AtPath(configFile).OfKind(cfg))
		if err != nil {
			setupLog.Error(err, "configuration is either missing or invalid")
			os.Exit(1)
		}

		options = o
	}

	setupLog.V(1).Info("eraser config",
		"manager", cfg.Manager,
		"component", cfg.Components,
		"options", fmt.Sprintf("%#v\n", options),
	)

	eraserOpts := config.NewManager(cfg)
	managerOpts := cfg.Manager

	watcher, err := setupWatcher(configFile)
	if err != nil {
		setupLog.Error(err, "unable to set up configuration file watch")
		os.Exit(1)
	}

	go startConfigWatch(watcher, eraserOpts, configFile)

	if managerOpts.Profile.Enabled {
		go func() {
			server := &http.Server{
				Addr:              fmt.Sprintf("localhost:%d", managerOpts.Profile.Port),
				ReadHeaderTimeout: 3 * time.Second,
			}
			err := server.ListenAndServe()
			setupLog.Error(err, "pprof server failed")
		}()
	}

	config := ctrl.GetConfigOrDie()
	config.UserAgent = version.GetUserAgent("manager")

	setupLog.Info("setting up manager", "userAgent", config.UserAgent)

	mgr, err := ctrl.NewManager(config, options)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	setupLog.Info("setup controllers")
	if err = controllers.SetupWithManager(mgr, eraserOpts); err != nil {
		setupLog.Error(err, "unable to setup controllers")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// Kubernetes manages configmap volume updates by creating a new file,
// changing the symlink, then deleting the old file. Hence, we want to
// watch for IN_DELETE_SELF events. In case the watch is dropped, we need
// to reestablish, so watch of IN_IGNORED too.
// https://ahmet.im/blog/kubernetes-inotify/ for more information.
func setupWatcher(configFile string) (*inotify.Watcher, error) {
	watcher, err := inotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	err = watcher.AddWatch(configFile, inotify.InDeleteSelf|inotify.InIgnored)
	if err != nil {
		return nil, err
	}
	return watcher, nil
}

func startConfigWatch(watcher *inotify.Watcher, eraserOpts *config.Manager, filename string) {
	for {
		select {
		case ev := <-watcher.Event:
			// by default inotify removes a watch on a file on an IN_DELETE_SELF
			// event, so we have to remove and reinstate the watch
			setupLog.V(1).Info("event", "event", ev)
			if ev.Mask&inotify.InIgnored != 0 {
				err := watcher.RemoveWatch(filename)
				if err != nil {
					setupLog.Error(err, "unable to remove watch on config")
				}

				err = watcher.AddWatch(filename, inotify.InDeleteSelf|inotify.InIgnored)
				if err != nil {
					setupLog.Error(err, "unable to set up new watch on configuration")
				}
				continue
			}

			cfg := config.Default()
			_, err := ctrl.Options{Scheme: runtime.NewScheme()}.AndFrom(ctrl.ConfigFile().AtPath(filename).OfKind(cfg))
			if err != nil {
				setupLog.Error(err, "configuration is missing or invalid", "event", ev, "filename", filename)
				continue
			}

			if err = eraserOpts.Update(cfg); err != nil {
				setupLog.Error(err, "configuration update failed")
				continue
			}

			newC, err := eraserOpts.Read()
			if err != nil {
				setupLog.Error(err, "unable to read back new configuration")
				continue
			}

			setupLog.V(1).Info("new configuration", "manager", newC.Manager, "components", newC.Components)
		case err := <-watcher.Error:
			setupLog.Error(err, "file watcher error")
		}
	}
}
