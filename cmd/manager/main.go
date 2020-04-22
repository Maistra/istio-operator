package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strconv"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	"github.com/magiconair/properties"
	"github.com/mitchellh/mapstructure"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"

	"github.com/maistra/istio-operator/pkg/apis"
	maistrav1 "github.com/maistra/istio-operator/pkg/apis/maistra/v1"
	"github.com/maistra/istio-operator/pkg/controller"
	"github.com/maistra/istio-operator/pkg/controller/common"
	"github.com/maistra/istio-operator/pkg/version"
)

// Change below variables to serve metrics on different host or port.
var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)
var log = logf.Log.WithName("cmd")

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)

	// number of concurrent reconciler for each controller
	pflag.Int("controlPlaneReconcilers", 1, "The number of concurrent reconcilers for ServiceMeshControlPlane resources")
	pflag.Int("memberRollReconcilers", 1, "The number of concurrent reconcilers for ServiceMeshMemberRoll resources")
	pflag.Int("memberReconcilers", 1, "The number of concurrent reconcilers for ServiceMeshMember resources")

	// flags to configure API request throttling
	pflag.Int("apiBurst", 50, "The number of API requests the operator can make before throttling is activated")
	pflag.Float32("apiQPS", 25, "The max rate of API requests when throttling is active")

	// custom flags for istio operator
	pflag.String("resourceDir", "/usr/local/share/istio-operator", "The location of the resources - helm charts, templates, etc.")
	pflag.String("chartsDir", "", "The root location of the helm charts.")
	pflag.String("defaultTemplatesDir", "", "The root location of the default templates.")
	pflag.String("userTemplatesDir", "", "The root location of the user supplied templates.")

	// config file
	configFile := ""
	pflag.StringVar(&configFile, "config", "/etc/istio-operator/config.properties", "The root location of the user supplied templates.")

	printVersion := false
	pflag.BoolVar(&printVersion, "version", printVersion, "Prints version information and exits")

	pflag.Parse()
	if printVersion {
		fmt.Printf("%s\n", version.Info)
		os.Exit(0)
	}

	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	log.Info(fmt.Sprintf("Starting Istio Operator %s", version.Info))

	if err := initializeConfiguration(configFile); err != nil {
		log.Error(err, "error initializing operator configuration")
		os.Exit(1)
	}

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	cfg.Burst = common.Config.Controller.APIBurst
	cfg.QPS = common.Config.Controller.APIQPS

	ctx := context.Background()
	// Become the leader before proceeding
	err = leader.Become(ctx, "istio-operator-lock")

	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	if err = serveCRMetrics(cfg); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports you want to expose.
	servicePorts := []v1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: v1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}
	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	}

	// CreateServiceMonitors will automatically create the prometheus-operator ServiceMonitor resources
	// necessary to configure Prometheus to scrape metrics from this operator.
	services := []*v1.Service{service}
	_, err = metrics.CreateServiceMonitors(cfg, namespace, services)
	if err != nil {
		log.Info("Could not create ServiceMonitor object", "error", err.Error())
		// If this operator is deployed to a cluster without the prometheus-operator running, it will return
		// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
		if err == metrics.ErrServiceMonitorNotPresent {
			log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
		}
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(maistrav1.SchemeBuilder.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs := common.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}

func initializeConfiguration(configFile string) error {
	v, err := common.NewViper()
	if err != nil {
		return err
	}

	// map flags to config structure
	// controller settings
	v.RegisterAlias("controller.controlPlaneReconcilers", "controlPlaneReconcilers")
	v.RegisterAlias("controller.memberRollReconcilers", "memberRollReconcilers")
	v.RegisterAlias("controller.memberReconcilers", "memberReconcilers")
	v.RegisterAlias("controller.apiBurst", "apiBurst")
	v.RegisterAlias("controller.apiQPS", "apiQPS")

	// rendering settings
	v.RegisterAlias("rendering.resourceDir", "resourceDir")
	v.RegisterAlias("rendering.chartsDir", "chartsDir")
	v.RegisterAlias("rendering.defaultTemplatesDir", "defaultTemplatesDir")
	v.RegisterAlias("rendering.userTemplatesDir", "userTemplatesDir")

	v.BindPFlags(pflag.CommandLine)
	v.AutomaticEnv()
	props, err := patchProperties(configFile)
	if err != nil {
		return err
	}
	if err := v.MergeConfigMap(props); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	if err := v.Unmarshal(common.Config, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "json"
	}); err != nil {
		return err
	}
	return nil
}

// downward api quotes values in the file (fmt.Sprintf("%q")), so we need to Unquote() them
func patchProperties(file string) (map[string]interface{}, error) {
	loader := properties.Loader{Encoding: properties.UTF8, IgnoreMissing: true, DisableExpansion: true}
	props, err := loader.LoadFile(file)
	if err != nil {
		return nil, err
	}
	retVal := make(map[string]interface{})
	for k, v := range props.Map() {
		var err error
		if retVal[k], err = strconv.Unquote(v); err != nil {
			return nil, err
		}
	}
	return retVal, nil
}
