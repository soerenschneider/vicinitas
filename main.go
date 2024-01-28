package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/rs/zerolog/log"
)

const defaultConfigFile = "/etc/vicinitas.yaml"

var (
	flagConfigFile            string
	flagVersion               bool
	sendNotHomePayloadOnError = true
	BuildVersion              string
	CommitHash                string
)

type Prober interface {
	Check(ctx context.Context) (bool, error)
}

type Notifier interface {
	Notify(ctx context.Context, name string, val string) error
}

type App struct {
	probes   map[string]Prober
	notifier Notifier
}

func main() {
	parseFlags()

	log.Info().Msgf("Starting version %s", BuildVersion)
	conf, err := ReadConfig(flagConfigFile)
	if err != nil {
		log.Fatal().Err(err).Msg("could not read config file")
	}

	notifier, err := buildNotifier(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("could not build notifier")
	}

	probes, err := buildProbers(conf)
	if err != nil {
		log.Fatal().Err(err).Msg("could not build probes")
	}

	go func() {
		VersionMetric.WithLabelValues(BuildVersion, CommitHash).Set(1)
		ProcessStartTime.SetToCurrentTime()
		if err := StartMetricsServer(conf.MetricsAddr); err != nil {
			log.Fatal().Err(err).Msg("can not start metrics server")
		}
	}()

	app := &App{
		probes:   probes,
		notifier: notifier,
	}

	app.run()
}

func (app *App) run() {
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT)
		<-sig
		log.Info().Msg("Got signal, quitting")
		cancel()
	}()

	ticker := time.NewTicker(30 * time.Second)
	app.tick(ctx)
	for {
		select {
		case <-ticker.C:
			app.tick(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func translateOutcome(outcome bool) string {
	if outcome {
		return "home"
	}
	return "not_home"
}

func (app *App) tick(ctx context.Context) {
	wg := &sync.WaitGroup{}
	for name, probe := range app.probes {
		wg.Add(1)
		go func(name string, probe Prober, w *sync.WaitGroup) {
			isPresent, err := probe.Check(ctx)
			if err != nil {
				log.Error().Err(err).Msgf("error probing %s", name)
				if !sendNotHomePayloadOnError {
					return
				}
			}

			if err := app.notifier.Notify(ctx, name, translateOutcome(isPresent)); err != nil {
				log.Error().Err(err).Msgf("error dispatching updates for %s: %v", name, err)
			}

			w.Done()
		}(name, probe, wg)
	}

	wg.Wait()
}

func parseFlags() {
	flag.StringVar(&flagConfigFile, "config", defaultConfigFile, "config file")
	flag.BoolVar(&flagVersion, "version", false, "print version and exit")

	flag.Parse()

	if flagVersion {
		fmt.Println(BuildVersion)
		os.Exit(0)
	}
}
