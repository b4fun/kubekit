package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"

	"github.com/b4fun/podkit"
	"github.com/b4fun/podkit/examples"
	"github.com/b4fun/podkit/podstream"
)

var (
	flagNamespace     string
	flagLabelSelector string
	flagContainer     string
	flagFollow        bool
	flagKeyword       string
	flagKubeConfig    *string
)

func setupFlags() {
	flagKubeConfig = examples.BindCLIFlags(flag.CommandLine)
	flag.StringVar(&flagNamespace, "namespace", "", "Specify the namespace to use.")
	flag.StringVar(&flagLabelSelector, "selector", "", "Selector (label query) to filter on.")
	flag.StringVar(&flagContainer, "container", "", "Print the logs of this container.")
	flag.BoolVar(&flagFollow, "follow", false, "Specify if the logs should be streamed.")
	flag.StringVar(&flagKeyword, "keyword", "", "Specify the keyword to filter on.")

	flag.Parse()

	if flagNamespace == "" {
		flagNamespace = "default"
	}

	if flagLabelSelector == "" {
		panic("label selector is required")
	}
}

func main() {
	setupFlags()

	kubeClient, err := examples.OutOfClusterKubeClient(*flagKubeConfig)
	if err != nil {
		panic(err.Error())
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	options := []podstream.Option{
		podstream.WithLogger(podkit.LogFunc(func(format string, args ...interface{}) {
			fmt.Printf(format+"\n", args...)
		})),
		podstream.ConsumeLogsWithFunc(func(x []podstream.LogEntry) {
			for _, log := range x {
				fmt.Println(log.Log)
			}
		}),
	}
	if flagFollow {
		options = append(options, podstream.FollowSelectedPods(flagLabelSelector))
	} else {
		options = append(options, podstream.FromSelectedPods(flagLabelSelector))
	}
	if flagContainer != "" {
		options = append(options, podstream.FromContainer(flagContainer))
	}
	if flagKeyword != "" {
		options = append(options, podstream.FilterWithRegex(flagKeyword))
	}

	done := make(chan struct{})
	go func() {
		defer close(done)

		if err := podstream.Stream(
			ctx.Done(),
			kubeClient.CoreV1().Pods(flagNamespace),
			options...,
		); err != nil {
			panic(err)
		}
	}()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	select {
	case <-sigs:
	case <-done:
	}
}
