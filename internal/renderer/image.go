package renderer

import (
	"context"
	"fmt"
	"os/exec"
)

func GenerateImages(dotFilename, svgFilename, pngFilename string) error {
	ctx := context.Background()

	if _, err := exec.LookPath("dot"); err != nil {
		return fmt.Errorf("graphviz (dot) not found")
	}

	if err := exec.CommandContext(ctx, "dot", "-Tsvg", dotFilename, "-o", svgFilename).Run(); err != nil {
		return fmt.Errorf("failed to generate SVG: %w", err)
	}

	if err := exec.CommandContext(ctx, "dot", "-Tpng", dotFilename, "-o", pngFilename).Run(); err != nil {
		return fmt.Errorf("failed to generate PNG: %w", err)
	}

	return nil
}
