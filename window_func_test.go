package main

import (
	"reflect"
	"testing"

	"github.com/gonutz/w32/v2"
)

func TestCenter(t *testing.T) {
	disp := w32.RECT{Left: 0, Top: 0, Right: 200, Bottom: 200}
	cur := w32.RECT{Left: 0, Top: 0, Right: 100, Bottom: 100}
	got := center(disp, cur)
	want := w32.RECT{Left: 50, Top: 50, Right: 150, Bottom: 150}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("center = %+v, want %+v", got, want)
	}
}

func TestResizeForDpi(t *testing.T) {
	src := w32.RECT{Left: 10, Top: 20, Right: 110, Bottom: 220}
	// Scale from DPI 96 to 192 (factor 2)
	got := resizeForDpi(src, 96, 192)
	want := w32.RECT{Left: 20, Top: 40, Right: 220, Bottom: 440}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("resizeForDpi = %+v, want %+v", got, want)
	}
}

func TestSameRect(t *testing.T) {
	a := &w32.RECT{Left: 0, Top: 0, Right: 10, Bottom: 10}
	b := &w32.RECT{Left: 0, Top: 0, Right: 10, Bottom: 10}
	if !sameRect(a, b) {
		t.Errorf("sameRect should be true for identical rectangles")
	}
	c := &w32.RECT{Left: 1, Top: 0, Right: 10, Bottom: 10}
	if sameRect(a, c) {
		t.Errorf("sameRect should be false for different rectangles")
	}
	if sameRect(nil, b) {
		t.Errorf("sameRect should be false when one argument is nil")
	}
}
