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

func TestExpandLabelSetsExpandsNamesAndNumberedSeries(t *testing.T) {
	got := expandLabelSets([]labelSet{
		names("John Doe", "Joe Doe"),
		numbered("Spare", 1, 3),
	})
	want := []string{"John Doe", "Joe Doe", "Spare 1", "Spare 2", "Spare 3"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expandLabelSets() = %#v, want %#v", got, want)
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
