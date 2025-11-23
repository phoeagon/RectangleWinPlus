package main

import (
	"reflect"
	"testing"

	"github.com/gonutz/w32/v2"
)

// helper to create a RECT
func rect(l, t, r, b int32) w32.RECT {
	return w32.RECT{Left: l, Top: t, Right: r, Bottom: b}
}

func TestToLeft(t *testing.T) {
	disp := rect(0, 0, 100, 100)
	got := toLeft(disp, 1, 2)
	want := rect(0, 0, 50, 100)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toLeft = %+v, want %+v", got, want)
	}
}

func TestToRight(t *testing.T) {
	disp := rect(0, 0, 100, 100)
	got := toRight(disp, 1, 2)
	want := rect(50, 0, 100, 100)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toRight = %+v, want %+v", got, want)
	}
}

func TestToTop(t *testing.T) {
	disp := rect(0, 0, 100, 100)
	got := toTop(disp, 1, 2)
	want := rect(0, 0, 100, 50)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toTop = %+v, want %+v", got, want)
	}
}

func TestToBottom(t *testing.T) {
	disp := rect(0, 0, 100, 100)
	got := toBottom(disp, 1, 2)
	want := rect(0, 50, 100, 100)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("toBottom = %+v, want %+v", got, want)
	}
}

func TestConvenienceWrappers(t *testing.T) {
	disp := rect(0, 0, 120, 120)
	tests := []struct {
		name string
		fn   func(w32.RECT, w32.RECT) w32.RECT
		want w32.RECT
	}{
		{"leftHalf", leftHalf, rect(0, 0, 60, 120)},
		{"leftOneThirds", leftOneThirds, rect(0, 0, 40, 120)},
		{"leftTwoThirds", leftTwoThirds, rect(0, 0, 80, 120)},
		{"topHalf", topHalf, rect(0, 0, 120, 60)},
		{"topOneThirds", topOneThirds, rect(0, 0, 120, 40)},
		{"topTwoThirds", topTwoThirds, rect(0, 0, 120, 80)},
		{"rightHalf", rightHalf, rect(60, 0, 120, 120)},
		{"rightOneThirds", rightOneThirds, rect(80, 0, 120, 120)},
		{"rightTwoThirds", rightTwoThirds, rect(40, 0, 120, 120)},
		{"bottomHalf", bottomHalf, rect(0, 60, 120, 120)},
		{"bottomOneThirds", bottomOneThirds, rect(0, 80, 120, 120)},
		{"bottomTwoThirds", bottomTwoThirds, rect(0, 40, 120, 120)},
	}
	for _, tt := range tests {
		got := tt.fn(disp, w32.RECT{})
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestMerge(t *testing.T) {
	left := rect(0, 0, 50, 100)
	top := rect(0, 0, 100, 50)
	got := merge(left, top)
	want := rect(0, 0, 50, 50)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("merge = %+v, want %+v", got, want)
	}
}

func TestResizeByPercent(t *testing.T) {
	disp := rect(0, 0, 100, 100)
	cur := rect(0, 0, 100, 100)
	// make smaller (sign = -1)
	got := resizeByPercent(disp, cur, -1)
	want := rect(5, 5, 95, 95)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("resizeByPercent (shrink) = %+v, want %+v", got, want)
	}
	// make larger (sign = 1) should stay within bounds
	got2 := resizeByPercent(disp, cur, 1)
	if !reflect.DeepEqual(got2, cur) {
		t.Errorf("resizeByPercent (grow) = %+v, want %+v", got2, cur)
	}
}

func TestMakeLargerSmaller(t *testing.T) {
	disp := rect(0, 0, 100, 100)
	cur := rect(0, 0, 100, 100)
	if got := makeLarger(disp, cur); !reflect.DeepEqual(got, cur) {
		t.Errorf("makeLarger should not exceed bounds, got %+v", got)
	}
	if got := makeSmaller(disp, cur); !reflect.DeepEqual(got, rect(5, 5, 95, 95)) {
		t.Errorf("makeSmaller = %+v, want %+v", got, rect(5, 5, 95, 95))
	}
}
