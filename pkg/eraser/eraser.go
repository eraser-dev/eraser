package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.opentelemetry.io/otel/metric/global"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/Azure/eraser/pkg/cri"
	"github.com/Azure/eraser/pkg/logger"
	"github.com/Azure/eraser/pkg/metrics"

	"github.com/Azure/eraser/api/unversioned"
	eraserv1 "github.com/Azure/eraser/api/v1"
	util "github.com/Azure/eraser/pkg/utils"
)

var (
	runtimePtr    = flag.String("runtime", "containerd", "container runtime")
	imageListPtr  = flag.String("imagelist", "", "name of ImageList")
	enableProfile = flag.Bool("enable-pprof", false, "enable pprof profiling")
	profilePort   = flag.Int("pprof-port", 6060, "port for pprof profiling. defaulted to 6060 if unspecified")

	// Timeout  of connecting to server (default: 5m).
	timeout  = 5 * time.Minute
	log      = logf.Log.WithName("eraser")
	excluded map[string]struct{}
)

const (
	generalErr = 1
)

func main() {
	flag.Parse()

	if *enableProfile {
		go func() {
			server := &http.Server{
				Addr:              fmt.Sprintf("localhost:%d", *profilePort),
				ReadHeaderTimeout: 3 * time.Second,
			}
			err := server.ListenAndServe()
			log.Error(err, "pprof server failed")
		}()
	}

	cfg := eraserv1.EraserContainerConfig{
		Runtime:     "containerd",
		ProfilePort: 6060,
	}

	text, err := os.ReadFile("/config.json")
	if err != nil {
		panic(err)
	}

	if err := json.Unmarshal(text, &cfg); err != nil {
		panic(err)
	}

	if err := logger.Configure(); err != nil {
		fmt.Fprintln(os.Stderr, "error setting up logger:", err)
		os.Exit(generalErr)
	}

	socketPath, found := util.RuntimeSocketPathMap[*runtimePtr]
	if !found {
		log.Error(fmt.Errorf("unsupported runtime"), "runtime", *runtimePtr)
		os.Exit(generalErr)
	}

	client, err := cri.NewEraserClient(socketPath)
	if err != nil {
		log.Error(err, "failed to get image client")
		os.Exit(generalErr)
	}

	var imagelist []string

	if *imageListPtr == "" {
		var f *os.File
		for {
			var err error

			f, err = os.OpenFile(util.ScanErasePath, os.O_RDONLY, 0)
			if err == nil {
				break
			}
			if !os.IsNotExist(err) {
				log.Error(err, "error opening scanErase pipe")
				os.Exit(generalErr)
			}
			time.Sleep(1 * time.Second)
			continue
		}

		// json data is list of []unversioned.Image
		data, err := io.ReadAll(f)
		if err != nil {
			log.Error(err, "error reading non-compliant images")
			os.Exit(generalErr)
		}
		f.Close()

		nonCompliantImages := []unversioned.Image{}
		if err = json.Unmarshal(data, &nonCompliantImages); err != nil {
			log.Error(err, "error in unmarshal non-compliant images")
			os.Exit(generalErr)
		}

		for _, img := range nonCompliantImages {
			imagelist = append(imagelist, img.ImageID)
		}

		log.Info("successfully created imagelist from scanned non-compliant images")
	} else {
		imagelist, err = util.ParseImageList(*imageListPtr)
		if err != nil {
			log.Error(err, "failed to parse image list file")
			os.Exit(generalErr)
		}
		log.Info("successfully parsed image list file")
	}

	excluded, err = util.ParseExcluded()
	if os.IsNotExist(err) {
		log.Info("configmaps for exclusion do not exist")
	} else if err != nil {
		log.Error(err, "failed to parse exclusion list")
		os.Exit(generalErr)
	}
	if len(excluded) == 0 {
		log.Info("no images to exclude")
	}

	removed, err := removeImages(client, imagelist)
	if err != nil {
		log.Error(err, "failed to remove images")
		os.Exit(generalErr)
	}

	if os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT") != "" {
		// record metrics
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

		exporter, reader, provider := metrics.ConfigureMetrics(ctx, log, os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"))
		global.SetMeterProvider(provider)

		if err := metrics.RecordMetricsEraser(ctx, global.MeterProvider(), int64(removed)); err != nil {
			log.Error(err, "error recording metrics")
		}
		metrics.ExportMetrics(log, exporter, reader, provider)
		cancel()
	}

	if *imageListPtr == "" {
		file, err := os.OpenFile(util.EraseCompleteCollectPath, os.O_WRONLY, 0)
		if err != nil {
			log.Error(err, "unable to open pipe", "pipeFile", util.EraseCompleteCollectPath)
			os.Exit(generalErr)
		}

		if _, err := file.WriteString(util.EraseCompleteMessage); err != nil {
			log.Error(err, "unable to write to pipe", "pipeFile", util.EraseCompleteCollectPath)
			os.Exit(generalErr)
		}

		file.Close()

		file, err = os.OpenFile(util.EraseCompleteScanPath, os.O_WRONLY, fs.ModeNamedPipe)
		// if the scanner is disabled
		if os.IsNotExist(err) {
			return
		}
		if err != nil {
			log.Error(err, "unable to open pipe", "pipeFile", util.EraseCompleteCollectPath)
			os.Exit(generalErr)
		}

		if _, err := file.WriteString(util.EraseCompleteMessage); err != nil {
			log.Error(err, "unable to write to pipe", "pipeFile", util.EraseCompleteCollectPath)
			os.Exit(generalErr)
		}

		file.Close()
	}
}
