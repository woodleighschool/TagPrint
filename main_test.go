package main

import (
	"context"
	"errors"
	"image"
	"io"
	"net"
	"reflect"
	"testing"
	"time"
)

func TestRequestedLabelsExpandsLabelsAndSeries(t *testing.T) {
	cfg := config{
		labels: []string{"John Doe", "Joe Doe"},
		series: []string{"Spare 1..3", "CRT 03..04"},
	}

	got, err := requestedLabels(cfg)
	if err != nil {
		t.Fatalf("requestedLabels: %v", err)
	}
	want := []string{"John Doe", "Joe Doe", "Spare 1", "Spare 2", "Spare 3", "CRT 03", "CRT 04"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requestedLabels() = %#v, want %#v", got, want)
	}
}

func TestRequestedLabelsRequiresAtLeastOneArgBasedLabel(t *testing.T) {
	_, err := requestedLabels(config{})
	if err == nil {
		t.Fatalf("requestedLabels() error = nil, want error")
	}
}

func TestLoadConfigAcceptsArgBasedLabels(t *testing.T) {
	cfg, err := loadConfig([]string{"-label", "CRT 03", "-label", "CRT 04", "-series", "Spare 1..2"})
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	wantLabels := []string{"CRT 03", "CRT 04"}
	wantSeries := []string{"Spare 1..2"}

	if !reflect.DeepEqual(cfg.labels, wantLabels) {
		t.Fatalf("cfg.labels = %#v, want %#v", cfg.labels, wantLabels)
	}
	if !reflect.DeepEqual(cfg.series, wantSeries) {
		t.Fatalf("cfg.series = %#v, want %#v", cfg.series, wantSeries)
	}
}

func TestRenderNameLabelKeepsTemplateSize(t *testing.T) {
	img, err := renderNameLabel("A deliberately long placeholder label")
	if err != nil {
		t.Fatalf("renderNameLabel: %v", err)
	}

	want := image.Rect(0, 0, labelWidth, labelHeight)
	if got := img.Bounds(); got != want {
		t.Fatalf("image bounds = %v, want %v", got, want)
	}
}

func TestRenderNameLabelUsesOpaqueWhiteBackground(t *testing.T) {
	img, err := renderNameLabel("John Doe")
	if err != nil {
		t.Fatalf("renderNameLabel: %v", err)
	}

	r, g, b, a := img.At(0, 0).RGBA()
	if r != 0xffff || g != 0xffff || b != 0xffff || a != 0xffff {
		t.Fatalf("top-left pixel = rgba(%#x, %#x, %#x, %#x), want opaque white", r, g, b, a)
	}
}

func TestBrotherPrinterClosesNetworkConnectionAfterPrint(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	t.Cleanup(func() {
		_ = listener.Close()
	})

	readDone := make(chan error, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			readDone <- err
			return
		}
		defer func() {
			_ = conn.Close()
		}()

		if err := conn.SetReadDeadline(time.Now().Add(2 * time.Second)); err != nil {
			readDone <- err
			return
		}

		buf := make([]byte, 32*1024)
		total := 0
		for {
			n, err := conn.Read(buf)
			total += n
			if errors.Is(err, io.EOF) {
				if total == 0 {
					readDone <- errors.New("printer received no bytes")
					return
				}
				readDone <- nil
				return
			}
			if err != nil {
				readDone <- err
				return
			}
		}
	}()

	printer := BrotherPrinter{
		addr:    listener.Addr().String(),
		timeout: 2 * time.Second,
	}

	if err := printer.Print(context.Background(), "CODEX TEST"); err != nil {
		t.Fatalf("print: %v", err)
	}

	if err := <-readDone; err != nil {
		t.Fatalf("printer connection was not closed cleanly after print: %v", err)
	}
}
