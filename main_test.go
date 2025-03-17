package main

import (
	"embed"
	"testing"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

//go:embed data
var testFs embed.FS

func TestAssetLoading(t *testing.T) {
	var err error
	_, _, err = ebitenutil.NewImageFromFileSystem(testFs,"data/actor.png")
	if err != nil {
		t.Error("failed testing: assets did not load correctly\n",err)
	}
}