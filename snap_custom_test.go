package main

import (
	"testing"

	"github.com/gonutz/w32/v2"
)

func TestPushLeft(t *testing.T) {
	disp := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	cur := w32.RECT{Left: 500, Top: 200, Right: 1000, Bottom: 700} // Width: 500, Height: 500

	expected := w32.RECT{
		Left:   0,
		Top:    200,
		Right:  500,
		Bottom: 700,
	}

	result := pushLeft(disp, cur)

	if result.Left != expected.Left || result.Top != expected.Top ||
		result.Right != expected.Right || result.Bottom != expected.Bottom {
		t.Errorf("pushLeft() = %v, want %v", result, expected)
	}
}

func TestPushRight(t *testing.T) {
	disp := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	cur := w32.RECT{Left: 500, Top: 200, Right: 1000, Bottom: 700} // Width: 500, Height: 500

	expected := w32.RECT{
		Left:   1920 - 500,
		Top:    200,
		Right:  1920,
		Bottom: 700,
	}

	result := pushRight(disp, cur)

	if result.Left != expected.Left || result.Top != expected.Top ||
		result.Right != expected.Right || result.Bottom != expected.Bottom {
		t.Errorf("pushRight() = %v, want %v", result, expected)
	}
}

func TestPushTop(t *testing.T) {
	disp := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	cur := w32.RECT{Left: 500, Top: 200, Right: 1000, Bottom: 700} // Width: 500, Height: 500

	expected := w32.RECT{
		Left:   500,
		Top:    0,
		Right:  1000,
		Bottom: 500,
	}

	result := pushTop(disp, cur)

	if result.Left != expected.Left || result.Top != expected.Top ||
		result.Right != expected.Right || result.Bottom != expected.Bottom {
		t.Errorf("pushTop() = %v, want %v", result, expected)
	}
}

func TestPushBottom(t *testing.T) {
	disp := w32.RECT{Left: 0, Top: 0, Right: 1920, Bottom: 1080}
	cur := w32.RECT{Left: 500, Top: 200, Right: 1000, Bottom: 700} // Width: 500, Height: 500

	expected := w32.RECT{
		Left:   500,
		Top:    1080 - 500,
		Right:  1000,
		Bottom: 1080,
	}

	result := pushBottom(disp, cur)

	if result.Left != expected.Left || result.Top != expected.Top ||
		result.Right != expected.Right || result.Bottom != expected.Bottom {
		t.Errorf("pushBottom() = %v, want %v", result, expected)
	}
}
