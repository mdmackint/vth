package main

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

func TestAssetLoading(t *testing.T) {
	var err error
	_, _, err = ebitenutil.NewImageFromFileSystem(fs,"data/actor.png")
	_ = loadImage("data/actor.png")
	_ = loadMultiple("data/actor.png","data/gravreset.png")

	if err != nil {
		t.Error("failed testing: assets did not load correctly\n",err)
	}
}