package main_test

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"

	"github.com/fogleman/primitive/primitive"
)

func TestOwl(t *testing.T) {
	input, err := primitive.LoadImage("./examples/owl.png")
	if err != nil {
		t.Fatal(err)
	}

	rand.Seed(42) // Ensure results are reproducible

	alpha := 128
	scale := 1
	mode := 1
	runs := 3

	expectedPath := fmt.Sprintf("./_testdata/owl_%d_%d_%d_%d.png", alpha, scale, mode, runs)

	expected, err := primitive.LoadImage(expectedPath)
	if err != nil {
		t.Fatal(err)
	}

	model := primitive.NewModel(input, alpha, scale, primitive.Mode(mode))
	output := model.Run(runs)

	//primitive.SavePNG(expectedPath, output) // uncomment to overwrite the control file (in case the algorithm really changed)

	if !reflect.DeepEqual(expected, output) {
		t.Fatal("Output does not match expectations")
	}
}

func BenchmarkOwl(b *testing.B) {
	input, err := primitive.LoadImage("./examples/owl.png")
	if err != nil {
		b.Fatal(err)
	}

	rand.Seed(42) // Ensure results are reproducible

	alpha := 128
	scale := 1
	mode := 1
	runs := 3

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model := primitive.NewModel(input, alpha, scale, primitive.Mode(mode))
		model.Run(runs)
	}
}
