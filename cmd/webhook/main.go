package main

import (
	"context"
	"os"

	"github.com/harvester/webhook/pkg/config"
	"github.com/harvester/webhook/pkg/server"
	"github.com/harvester/webhook/pkg/server/admission"
	"github.com/rancher/wrangler/v3/pkg/kubeconfig"
	"github.com/rancher/wrangler/v3/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/rest"

	"github.com/harvester/node-manager/pkg/admitter"
	"github.com/harvester/node-manager/pkg/mutator"
)

const webhookName = "harvester-node-manager-webhook"

func main() {
	var options config.Options
	var logLevel string

	flags := []cli.Flag{
		&cli.StringFlag{
			Name:        "loglevel",
			Usage:       "Specify log level",
			EnvVars:     []string{"LOGLEVEL"},
			Value:       "info",
			Destination: &logLevel,
		},
		&cli.IntFlag{
			Name:        "threadiness",
			EnvVars:     []string{"THREADINESS"},
			Usage:       "Specify controller threads",
			Value:       5,
			Destination: &options.Threadiness,
		},
		&cli.IntFlag{
			Name:        "https-port",
			EnvVars:     []string{"WEBHOOK_SERVER_HTTPS_PORT"},
			Usage:       "HTTPS listen port",
			Value:       8443,
			Destination: &options.HTTPSListenPort,
		},
		&cli.StringFlag{
			Name:        "namespace",
			EnvVars:     []string{"NAMESPACE"},
			Destination: &options.Namespace,
			Usage:       "The harvester namespace",
			Value:       "harvester-system",
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "controller-user",
			EnvVars:     []string{"CONTROLLER_USER_NAME"},
			Destination: &options.ControllerUsername,
			Value:       "harvester-node-manager",
			Usage:       "The harvester controller username",
		},
		&cli.StringFlag{
			Name:        "gc-user",
			EnvVars:     []string{"GARBAGE_COLLECTION_USER_NAME"},
			Destination: &options.GarbageCollectionUsername,
			Usage:       "The system username that performs garbage collection",
			Value:       "system:serviceaccount:kube-system:generic-garbage-collector",
		},
	}

	cfg, err := kubeconfig.GetNonInteractiveClientConfig(os.Getenv("KUBECONFIG")).ClientConfig()
	if err != nil {
		logrus.Fatal(err)
	}

	ctx := signals.SetupSignalContext()

	app := cli.NewApp()
	app.Flags = flags
	app.Action = func(_ *cli.Context) error {
		setLogLevel(logLevel)
		err := runWebhookServer(ctx, cfg, &options)
		return err
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatalf("run webhook server failed: %v", err)
	}
}

func runWebhookServer(ctx context.Context, cfg *rest.Config, options *config.Options) error {
	webhookServer := server.NewWebhookServer(ctx, cfg, webhookName, options)

	cloudinitValidator, err := admitter.NewCloudInitValidator(cfg)
	if err != nil {
		return err
	}

	var validators = []admission.Validator{
		cloudinitValidator,
	}

	if err := webhookServer.RegisterValidators(validators...); err != nil {
		return err
	}

	var mutators = []admission.Mutator{
		mutator.NewCloudInitMutator(),
	}

	if err := webhookServer.RegisterMutators(mutators...); err != nil {
		return err
	}

	if err := webhookServer.Start(); err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}

func setLogLevel(level string) {
	ll, err := logrus.ParseLevel(level)
	if err != nil {
		ll = logrus.DebugLevel
	}
	// set global log level
	logrus.SetLevel(ll)
}
