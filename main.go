//go:generate go run pkg/codegen/cleanup/main.go
//go:generate go run pkg/codegen/main.go
//go:generate /bin/bash scripts/generate-manifest

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/ehazlett/simplelog"
	ctlnode "github.com/rancher/wrangler/v3/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/rancher/wrangler/v3/pkg/start"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/harvester/node-manager/pkg/controller/cloudinit"
	"github.com/harvester/node-manager/pkg/controller/ksmtuned"
	"github.com/harvester/node-manager/pkg/controller/nodeconfig"
	ctlnodeharvester "github.com/harvester/node-manager/pkg/generated/controllers/node.harvesterhci.io"
	"github.com/harvester/node-manager/pkg/metrics"
	"github.com/harvester/node-manager/pkg/monitor"
	"github.com/harvester/node-manager/pkg/option"
	"github.com/harvester/node-manager/pkg/utils"
	"github.com/harvester/node-manager/pkg/version"
)

var (
	VERSION = "v0.0.0-dev"
)

func main() {
	var opt option.Option

	app := cli.NewApp()
	app.Name = "harvester-node-manager"
	app.Version = VERSION
	app.Usage = "Harvester Node Manager, to help with cluster node configuration. Options kubeconfig or masterurl are required."
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "kubeconfig, k",
			EnvVars:     []string{"KUBECONFIG"},
			Value:       "",
			Usage:       "Kubernetes config files, e.g. $HOME/.kube/config",
			Destination: &opt.KubeConfig,
		},
		&cli.StringFlag{
			Name:        "node, n",
			EnvVars:     []string{"NODENAME"},
			Value:       "",
			Usage:       "Specify the node name",
			Destination: &opt.NodeName,
		},
		&cli.StringFlag{
			Name:        "profile-listen-address",
			Value:       "0.0.0.0:6060",
			DefaultText: "0.0.0.0:6060",
			Usage:       "Address to listen on for profiling",
			Destination: &opt.ProfilerAddress,
		},
		&cli.StringFlag{
			Name:        "log-format",
			EnvVars:     []string{"NDM_LOG_FORMAT"},
			Usage:       "Log format",
			Value:       "text",
			DefaultText: "text",
			Destination: &opt.LogFormat,
		},
		&cli.BoolFlag{
			Name:        "trace",
			EnvVars:     []string{"TRACE"},
			Usage:       "Run trace logs",
			Destination: &opt.Trace,
		},
		&cli.BoolFlag{
			Name:        "debug",
			EnvVars:     []string{"DEBUG"},
			Usage:       "enable debug logs",
			Destination: &opt.Debug,
		},
		&cli.IntFlag{
			Name:        "threadiness",
			Value:       2,
			DefaultText: "2",
			Destination: &opt.Threadiness,
		},
	}

	app.Action = func(_ *cli.Context) error {
		initProfiling(&opt)
		initLogs(&opt)
		return run(&opt)
	}

	if err := app.Run(os.Args); err != nil {
		klog.Fatal(err)
	}
}

func initProfiling(opt *option.Option) {
	// enable profiler
	if opt.ProfilerAddress != "" {
		profilerServer := &http.Server{
			Addr:              opt.ProfilerAddress,
			ReadHeaderTimeout: 10 * time.Second,
		}
		go func() {
			log.Println(profilerServer.ListenAndServe())
		}()
	}
}

func initLogs(opt *option.Option) {
	switch opt.LogFormat {
	case "simple":
		logrus.SetFormatter(&simplelog.StandardFormatter{})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
	logrus.SetOutput(os.Stdout)
	logrus.Infof("Ksmtuned controller %s is starting", version.FriendlyVersion())
	if opt.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("Loglevel set to [%v]", logrus.DebugLevel)
	}
	if opt.Trace {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.Tracef("Loglevel set to [%v]", logrus.TraceLevel)
	}
}

func run(opt *option.Option) error {
	ctx := signals.SetupSignalContext()
	mtx := &sync.Mutex{}

	cfg, err := clientcmd.BuildConfigFromFlags(opt.MasterURL, opt.KubeConfig)
	if err != nil {
		klog.Fatalf("Error building config from flags: %s", err.Error())
	}

	nodes, err := ctlnode.NewFactoryFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("error building node controllers: %s", err.Error())
	}

	nodectl, err := ctlnodeharvester.NewFactoryFromConfig(cfg)
	if err != nil {
		return fmt.Errorf("error building harvester-node-manager controllers: %s", err.Error())
	}
	nodecfg := nodectl.Node().V1beta1().NodeConfig()
	nds := nodes.Core().V1().Node()
	cloudinits := nodectl.Node().V1beta1().CloudInit()
	events := nodes.Core().V1().Event()

	var ksmtunedController *ksmtuned.Controller
	run := func(ctx context.Context) {
		kts := nodectl.Node().V1beta1().Ksmtuned()
		if ksmtunedController, err = ksmtuned.Register(
			ctx,
			opt.NodeName,
			kts,
			nds,
		); err != nil {
			logrus.Fatalf("failed to register ksmtuned controller: %s", err)
		}

		if _, err = nodeconfig.Register(
			ctx,
			opt.NodeName,
			nodecfg,
			nds,
			mtx,
		); err != nil {
			logrus.Fatalf("failed to register ksmtuned controller: %s", err)
		}

		cloudinit.Register(ctx, opt.NodeName, cloudinits, nds.Cache(), events)

		if err := start.All(ctx, opt.Threadiness, nodectl, nodes); err != nil {
			logrus.Fatalf("error starting, %s", err.Error())
		}
	}

	// start monitoring
	monitorTemplate := monitor.NewMonitorTemplate(ctx, mtx, nodecfg, nds, cloudinits, opt.NodeName)
	monitorNnumbers := len(utils.GetToMonitorServices())

	monitorModules := make([]interface{}, 0, monitorNnumbers)
	for _, serviceName := range utils.GetToMonitorServices() {
		monitorModule := monitor.InitServiceMonitor(monitorTemplate, serviceName)
		monitorModules = append(monitorModules, monitorModule)
	}
	monitor.StartsAllMonitors(monitorModules)

	go metrics.Run()

	run(ctx)

	<-ctx.Done()
	return ksmtunedController.Ksmtuned.Stop()
}
