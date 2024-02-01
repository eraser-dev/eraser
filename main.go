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
	"context"
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
	"sigs.k8s.io/yaml"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"github.com/eraser-dev/eraser/api/unversioned"
	"github.com/eraser-dev/eraser/api/unversioned/config"
	eraserv1 "github.com/eraser-dev/eraser/api/v1"
	eraserv1alpha1 "github.com/eraser-dev/eraser/api/v1alpha1"
	v1alpha1Config "github.com/eraser-dev/eraser/api/v1alpha1/config"
	eraserv1alpha2 "github.com/eraser-dev/eraser/api/v1alpha2"
	v1alpha2Config "github.com/eraser-dev/eraser/api/v1alpha2/config"
	eraserv1alpha3 "github.com/eraser-dev/eraser/api/v1alpha3"
	v1alpha3Config "github.com/eraser-dev/eraser/api/v1alpha3/config"
	"github.com/eraser-dev/eraser/controllers"
	"github.com/eraser-dev/eraser/pkg/logger"
	"github.com/eraser-dev/eraser/pkg/utils"
	"github.com/eraser-dev/eraser/version"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")

	fromV1alpha1 = eraserv1alpha1.Convert_v1alpha1_EraserConfig_To_unversioned_EraserConfig
	fromV1alpha2 = eraserv1alpha2.Convert_v1alpha2_EraserConfig_To_unversioned_EraserConfig
	fromV1alpha3 = eraserv1alpha3.Convert_v1alpha3_EraserConfig_To_unversioned_EraserConfig
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(eraserv1alpha1.AddToScheme(scheme))
	utilruntime.Must(eraserv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

type apiVersion struct {
	APIVersion string `json:"apiVersion"`
}

type convertFunc[T any] func(*T, *unversioned.EraserConfig, conversion.Scope) error

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		<-ctx.Done()
		os.Exit(1)
	}()

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
		NewCache: cache.BuilderWithOptions(cache.Options{
			SelectorsByObject: cache.SelectorsByObject{
				// to watch eraser pods
				&corev1.Pod{}: {
					Field: fields.OneTermEqualSelector("metadata.namespace", utils.GetNamespace()),
				},
				// to watch eraser podTemplates
				&corev1.PodTemplate{}: {
					Field: fields.OneTermEqualSelector("metadata.namespace", utils.GetNamespace()),
				},
				// to watch eraser-manager-configs
				&corev1.ConfigMap{}: {
					Field: fields.OneTermEqualSelector("metadata.namespace", utils.GetNamespace()),
				},
				// to watch ImageJobs
				&eraserv1.ImageJob{}: {},
				// to watch ImageLists
				&eraserv1.ImageList{}: {},
			},
		}),
	}

	if configFile == "" {
		setupLog.Error(fmt.Errorf("config file was not supplied"), "aborting")
		os.Exit(1)
	}

	cfg, err := getConfig(configFile)
	if err != nil {
		setupLog.Error(err, "error getting configuration")
		os.Exit(1)
	}

	setupLog.V(1).Info("eraser config",
		"manager", cfg.Manager,
		"components", cfg.Components,
		"options", fmt.Sprintf("%#v\n", options),
		"typeMeta", fmt.Sprintf("%#v\n", cfg.TypeMeta),
	)

	eraserOpts := config.NewManager(cfg)
	managerOpts := cfg.Manager

	watcher, err := setupWatcher(configFile)
	if err != nil {
		setupLog.Error(err, "unable to set up configuration file watch")
		os.Exit(1)
	}

	go startConfigWatch(cancel, watcher, eraserOpts, configFile)

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

func getConfig(configFile string) (*unversioned.EraserConfig, error) {
	fileBytes, err := os.ReadFile(configFile)
	if err != nil {
		setupLog.Error(err, "configuration is either missing or invalid")
		os.Exit(1)
	}

	var av apiVersion
	if err := yaml.Unmarshal(fileBytes, &av); err != nil {
		setupLog.Error(err, "cannot unmarshal yaml", "bytes", string(fileBytes), "apiVersion", av)
		os.Exit(1)
	}

	switch av.APIVersion {
	case "eraser.sh/v1alpha1":
		return getUnversioned(fileBytes, v1alpha1Config.Default(), fromV1alpha1)
	case "eraser.sh/v1alpha2":
		return getUnversioned(fileBytes, v1alpha2Config.Default(), fromV1alpha2)
	case "eraser.sh/v1alpha3":
		return getUnversioned(fileBytes, v1alpha3Config.Default(), fromV1alpha3)
	default:
		setupLog.Error(fmt.Errorf("unknown api version"), "error", "apiVersion", av.APIVersion)
		return nil, err
	}
}

func getUnversioned[T any](b []byte, defaults *T, convert convertFunc[T]) (*unversioned.EraserConfig, error) {
	cfg := defaults

	if err := yaml.Unmarshal(b, cfg); err != nil {
		setupLog.Error(err, "configuration is either missing or invalid")
		return nil, err
	}

	var unv unversioned.EraserConfig
	if err := convert(cfg, &unv, nil); err != nil {
		return nil, err
	}

	return &unv, nil
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

func startConfigWatch(cancel context.CancelFunc, watcher *inotify.Watcher, eraserOpts *config.Manager, filename string) {
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

			var err error
			oldConfig := new(unversioned.EraserConfig)

			*oldConfig, err = eraserOpts.Read()
			if err != nil {
				setupLog.Error(err, "configuration could not be read", "event", ev, "filename", filename)
			}

			newConfig, err := getConfig(filename)
			if err != nil {
				setupLog.Error(err, "configuration is missing or invalid", "event", ev, "filename", filename)
				continue
			}

			if err = eraserOpts.Update(newConfig); err != nil {
				setupLog.Error(err, "configuration update failed")
				continue
			}

			// read back the new configuration
			*newConfig, err = eraserOpts.Read()
			if err != nil {
				setupLog.Error(err, "unable to read back new configuration")
				continue
			}

			if needsRestart(oldConfig, newConfig) {
				setupLog.Info("configurations differ in an irreconcileable way, restarting", "old", oldConfig.Components, "new", newConfig.Components)
				// restarts the manager
				cancel()
			}

			setupLog.V(1).Info("new configuration", "manager", newConfig.Manager, "components", newConfig.Components)
		case err := <-watcher.Error:
			setupLog.Error(err, "file watcher error")
		}
	}
}

func needsRestart(oldConfig, newConfig *unversioned.EraserConfig) bool {
	type check struct {
		collector bool
		scanner   bool
	}

	oldComponents := check{collector: oldConfig.Components.Collector.Enabled, scanner: oldConfig.Components.Scanner.Enabled}
	newComponents := check{collector: newConfig.Components.Collector.Enabled, scanner: newConfig.Components.Scanner.Enabled}
	return oldComponents != newComponents
}
