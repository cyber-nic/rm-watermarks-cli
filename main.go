// Package main implements a watermark removal tool using inpainting techniques.
// To remove a watermark from a JPEG image containing text using OpenCV in Go (using the GoCV package), you would typically follow these steps:
// Read the Image: Load the JPEG image.
// Preprocess: Convert to grayscale, apply Gaussian blur to smooth the image.
// Thresholding: Apply adaptive thresholding to highlight text and suppress noise.
// Watermark Detection: Manually or automatically detect the region of the watermark.
// Watermark Removal: Use inpainting techniques to remove the watermark.
// Post-processing: Additional steps like smoothing to improve the visual quality.

package main

import (
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"
	"gopkg.in/yaml.v3"
)

const (
	CarbonCopyThreshold float32 = 96
)

type Mask struct {
	File    string `yaml:"file"`
	Gravity string `yaml:"gravity"`
	Foreground bool `yaml:"foreground"`
}

type AppConfig struct {
	Debug  bool
	Info   bool
	Visual bool
	Human  bool
	Masks  []Mask
}

func main() {
	// Read flags
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	srcPath := flag.String("src", "", "sets input image path")
	dstPath := flag.String("dst", "", "sets destination image path")
	debugFlag := flag.Bool("debug", false, "Debug logging level")
	configFilename := flag.String("config", "local.env.yaml", "Config File")
	flag.Parse()

	// Read config file
	configFile, err := os.ReadFile(*configFilename)
	if err != nil {
		panic(err)
	}

	// Unmarshal the JSON data into a Config struct
	var cfg AppConfig
	err = yaml.Unmarshal(configFile, &cfg)
	if err != nil {
		panic(err)
	}

	debug := cfg.Debug
	if *debugFlag {
		debug = *debugFlag
	}

	// Perform input validation
	if *srcPath == "" || *dstPath == "" {
		panic("src, dst, and mask are all required")
	}

	// Set log level
	zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	if cfg.Info {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
	if cfg.Human {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	// Start
	start := time.Now()
	base := filepath.Base(*srcPath)
	log.Debug().Str("image", *srcPath).Msg(base)

	// Read image
	src := gocv.IMRead(*srcPath, gocv.IMReadColor)
	defer src.Close()

	// Compute image metrics
	// b captures the overall average brightness of the image
	// m represents the average of the channel-wise means, indicating the image's overall color balance
	// s measures the average spread of pixel values across channels, reflecting the image's overall contrast or detail level
	b, m, s := ComputeImageChannelMetrics(src)

	// Invert colors if carbon copy
	img := src.Clone()
	defer img.Close()
	if b < CarbonCopyThreshold {
		img = InvertColors(src)
	}

	// Detect if color image
	color := IsColor(img)

	// Remove colors. Inpainting works best on grayscale images
	img = RemoveColors(img.Clone())

	// Compute binary image using mean threshold
	thresh := s
	if color {
		dt := (m - s) / 2
		// t = 1.2 *s
		// t += dt
		thresh = m - dt
	}

	// Create init empty mask
	mask := gocv.NewMatWithSize(src.Rows(), src.Cols(), gocv.MatTypeCV8UC1)
	mask.SetTo(gocv.Scalar{Val1: 0, Val2: 0, Val3: 0, Val4: 255})
	defer mask.Close()

	// Aggregate masks
	for _, m := range cfg.Masks {
		perf := time.Now()

		// Read watermark mask template
		maskTpl := gocv.IMRead(m.File, gocv.IMReadGrayScale)
		defer maskTpl.Close()

		// Compute image specific watermark mask
		_, bin, fg, msk := ComputeWatermarkMask(img, maskTpl, m.Gravity, thresh, m.Foreground)
		defer msk.Close()

		// Aggregate masks
		gocv.BitwiseOr(mask.Clone(), msk, &mask)

		if cfg.Visual {
			// gocv.NewWindow("crop").IMShow(crop)
			gocv.NewWindow("bin").IMShow(bin)
			gocv.NewWindow("fg").IMShow(fg)
			// gocv.NewWindow("mask").IMShow(maskTpl)
			gocv.WaitKey(0)
		}
		
		log.Debug().
			Int64("duration(ms)", (time.Since(perf)).Milliseconds()).
			Str("mask", m.File).Msg(base)
	}

	// Apply inpainting to remove the watermark
	out := RemoveWatermark(img, mask)
	defer out.Close()

	if cfg.Visual {
		gocv.NewWindow("src").IMShow(src)
		// gocv.NewWindow("gray").IMShow(img)
		gocv.NewWindow("mask").IMShow(mask)
		gocv.NewWindow("Result").IMShow(out)
		gocv.WaitKey(0)
		return
	}

	// Write file
	if ok := gocv.IMWrite(*dstPath, out); !ok {
		panic("error writing image to disk")
	}

	// Done
	log.Info().
		Int64("duration(ms)", (time.Since(start)).Milliseconds()).
		Float32("brightness", b).
		Float32("mean", m).
		Float32("stdDev", s).
		Float32("threshold", thresh).
		Bool("color", color).
		Str("dst", *dstPath).
		Msg(base)
}
