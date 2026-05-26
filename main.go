package main

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log/slog"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	brotherql "github.com/suapapa/go_brother-ql"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
)

const (
	defaultPrinterAddr = "172.19.10.13"
	defaultTimeout     = 30 * time.Second
)

const (
	printerModel     = "QL-820NWB"
	printerBackend   = "network"
	printerLabel     = "62"
	printerRotate    = "0"
	printerThreshold = 70.0
)

const (
	labelWidth  = 1181
	labelHeight = 566
	maxTextW    = 1181
	maxTextH    = 566
	fontStart   = 245.0
	fontMin     = 8.0
)

//go:embed assets/template.png
var templatePNG []byte

var boldFont = mustParseFont(gobold.TTF)

var seriesPattern = regexp.MustCompile(`^(.*?)(\d+)\.\.(\d+)$`)

type BrotherPrinter struct {
	addr    string
	timeout time.Duration
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("tagprint failed", "err", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := loadConfig(args)
	if err != nil {
		return err
	}

	names, err := requestedLabels(cfg)
	if err != nil {
		return err
	}
	if cfg.limit > 0 && cfg.limit < len(names) {
		names = names[:cfg.limit]
	}

	if cfg.preview {
		return writePreview(cfg.previewPath, names[0])
	}

	printer := BrotherPrinter{
		addr:    cfg.printerAddr,
		timeout: cfg.timeout,
	}

	for i, name := range names {
		slog.Info("printing label", "label", name, "index", i+1, "total", len(names), "printer", cfg.printerAddr)
		if err := printer.Print(context.Background(), name); err != nil {
			return fmt.Errorf("print %q: %w", name, err)
		}
	}
	return nil
}

type config struct {
	printerAddr string
	timeout     time.Duration
	limit       int
	preview     bool
	previewPath string
	labels      []string
	series      []string
}

func loadConfig(args []string) (config, error) {
	cfg := config{
		printerAddr: env("PRINTER_ADDR", defaultPrinterAddr),
		timeout:     defaultTimeout,
		previewPath: "preview.png",
	}

	if v := os.Getenv("PRINT_TIMEOUT"); v != "" {
		timeout, err := time.ParseDuration(v)
		if err != nil || timeout <= 0 {
			return config{}, errors.New("PRINT_TIMEOUT must be a positive duration")
		}
		cfg.timeout = timeout
	}

	flags := flag.NewFlagSet("tagprint", flag.ContinueOnError)
	flags.StringVar(&cfg.printerAddr, "printer", cfg.printerAddr, "Brother printer address")
	flags.DurationVar(&cfg.timeout, "timeout", cfg.timeout, "per-label print timeout")
	flags.IntVar(&cfg.limit, "limit", 0, "maximum labels to print; 0 prints all labels")
	flags.BoolVar(&cfg.preview, "preview", false, "render the first label to a PNG instead of printing")
	flags.StringVar(&cfg.previewPath, "preview-path", cfg.previewPath, "preview PNG path")
	flags.Var((*labelFlags)(&cfg.labels), "label", "one-off label text; may be repeated")
	flags.Var((*labelFlags)(&cfg.series), "series", `numbered label series, such as "Spare 1..6" or "CRT 03..08"; may be repeated`)
	if err := flags.Parse(args); err != nil {
		return config{}, err
	}

	if cfg.printerAddr == "" {
		return config{}, errors.New("printer address is required")
	}
	if cfg.timeout <= 0 {
		return config{}, errors.New("timeout must be positive")
	}
	if cfg.limit < 0 {
		return config{}, errors.New("limit must be zero or greater")
	}
	if flags.NArg() != 0 {
		return config{}, fmt.Errorf("unexpected arguments: %s", strings.Join(flags.Args(), " "))
	}

	return cfg, nil
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

type labelFlags []string

func (f *labelFlags) String() string {
	if f == nil {
		return ""
	}
	return strings.Join(*f, ", ")
}

func (f *labelFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("label must not be empty")
	}
	*f = append(*f, value)
	return nil
}

func requestedLabels(cfg config) ([]string, error) {
	requested := append([]string(nil), cfg.labels...)
	for _, spec := range cfg.series {
		expanded, err := expandSeries(spec)
		if err != nil {
			return nil, err
		}
		requested = append(requested, expanded...)
	}
	if len(requested) == 0 {
		return nil, errors.New("at least one -label or -series is required")
	}
	return requested, nil
}

func expandSeries(spec string) ([]string, error) {
	matches := seriesPattern.FindStringSubmatch(strings.TrimSpace(spec))
	if matches == nil {
		return nil, fmt.Errorf("series %q must look like \"Spare 1..6\" or \"CRT 03..08\"", spec)
	}

	prefix := matches[1]
	startText := matches[2]
	endText := matches[3]

	start, err := strconv.Atoi(startText)
	if err != nil {
		return nil, fmt.Errorf("parse series start %q: %w", startText, err)
	}
	end, err := strconv.Atoi(endText)
	if err != nil {
		return nil, fmt.Errorf("parse series end %q: %w", endText, err)
	}

	width := len(startText)
	if len(endText) > width {
		width = len(endText)
	}

	step := 1
	if end < start {
		step = -1
	}

	var labels []string
	for i := start; ; i += step {
		labels = append(labels, prefix+fmt.Sprintf("%0*d", width, i))
		if i == end {
			break
		}
	}
	return labels, nil
}

func writePreview(path, name string) error {
	img, err := renderNameLabel(name)
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create preview: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	if err := png.Encode(file, img); err != nil {
		return fmt.Errorf("write preview: %w", err)
	}
	return nil
}

func (p BrotherPrinter) Print(ctx context.Context, name string) error {
	img, err := renderNameLabel(name)
	if err != nil {
		return err
	}

	printCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	printer, err := brotherql.NewLabelPrinter(printCtx, printerModel, printerBackend, p.addr)
	if err != nil {
		return fmt.Errorf("create printer: %w", err)
	}

	start := time.Now()
	slog.Info("printer write started", "label", name, "addr", p.addr, "timeout", p.timeout)
	printErr := printer.Print(printCtx, []image.Image{img}, printOptions())
	if printErr == nil {
		slog.Info("printer write completed", "label", name, "duration", time.Since(start))
	}

	closeErr := printer.Close()
	if closeErr != nil {
		slog.Error("printer connection close failed", "label", name, "err", closeErr)
	} else {
		slog.Info("printer connection closed", "label", name)
	}

	if printErr != nil || closeErr != nil {
		return errors.Join(printErr, closeErr)
	}
	return nil
}

func printOptions() brotherql.PrintOptions {
	opts := brotherql.NewDefaultOptions(printerLabel)
	opts.Cut = true
	opts.Dither = false
	opts.Compress = false
	opts.Red = false
	opts.Rotate = printerRotate
	opts.Dpi600 = false
	opts.Hq = true
	opts.Threshold = printerThreshold

	return opts
}

func renderNameLabel(name string) (image.Image, error) {
	base, err := png.Decode(bytes.NewReader(templatePNG))
	if err != nil {
		return nil, fmt.Errorf("decode template: %w", err)
	}

	canvas := image.NewRGBA(image.Rect(0, 0, labelWidth, labelHeight))
	draw.Draw(canvas, canvas.Bounds(), image.White, image.Point{}, draw.Src)
	draw.Draw(canvas, canvas.Bounds(), base, image.Point{}, draw.Over)

	face, err := fittingFace(name)
	if err != nil {
		return nil, err
	}
	defer closeFace(face)

	metrics := face.Metrics()
	width := font.MeasureString(face, name).Ceil()
	textHeight := (metrics.Ascent + metrics.Descent).Ceil()
	x := (labelWidth - width) / 2
	y := (labelHeight-textHeight)/2 + metrics.Ascent.Ceil()

	drawer := &font.Drawer{
		Dst:  canvas,
		Src:  image.Black,
		Face: face,
		Dot:  fixed.P(x, y),
	}
	drawer.DrawString(name)

	return canvas, nil
}

func fittingFace(text string) (font.Face, error) {
	for size := fontStart; size >= fontMin; size-- {
		face, err := fontFace(boldFont, size)
		if err != nil {
			return nil, err
		}
		metrics := face.Metrics()
		if font.MeasureString(face, text).Ceil() <= maxTextW && (metrics.Ascent+metrics.Descent).Ceil() <= maxTextH {
			return face, nil
		}
		closeFace(face)
	}
	return fontFace(boldFont, fontMin)
}

func mustParseFont(fontBytes []byte) *sfnt.Font {
	parsed, err := opentype.Parse(fontBytes)
	if err != nil {
		panic(err)
	}
	return parsed
}

func fontFace(parsed *sfnt.Font, size float64) (font.Face, error) {
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, fmt.Errorf("create font face: %w", err)
	}
	return face, nil
}

func closeFace(face font.Face) {
	if closer, ok := face.(interface{ Close() error }); ok {
		_ = closer.Close()
	}
}
