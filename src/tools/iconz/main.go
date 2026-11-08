// Command iconz rasterizes ../../assets/vigil.svg into PNG brand icons and
// generates the Windows resource objects (.syso) that embed the icon + version
// info into the vigil.exe binary.
//
//	cd tools/iconz && go run .
package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

const svgPath = "../../assets/vigil.svg"

var iconSizes = []int{16, 32, 48, 64, 128, 256, 512}

func main() {
	icon, err := oksvg.ReadIcon(svgPath, oksvg.WarnErrorMode)
	if err != nil {
		fail(err)
	}

	outDir := "../../assets/icons"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fail(err)
	}

	var winImgs []image.Image // Windows .ico images must fit in 256x256.
	for _, size := range iconSizes {
		img := render(icon, size)
		if size <= 256 {
			winImgs = append(winImgs, img)
		}
		out := filepath.Join(outDir, fmt.Sprintf("vigil_%d.png", size))
		if err := writePNG(out, img); err != nil {
			fail(err)
		}
		fmt.Println("wrote", out)
	}

	if err := writeWindowsResources(winImgs); err != nil {
		fail(err)
	}
}

func render(icon *oksvg.SvgIcon, size int) *image.RGBA {
	icon.SetTarget(0, 0, float64(size), float64(size))
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	raster := rasterx.NewDasher(size, size, scanner)
	icon.Draw(raster, 1.0)
	return img
}

// writeWindowsResources embeds the icon + version info into rsrc_windows_*.syso
// files in cmd/vigil so `go build` links them into vigil.exe automatically.
func writeWindowsResources(imgs []image.Image) error {
	winIcon, err := winres.NewIconFromImages(imgs)
	if err != nil {
		return err
	}
	rs := &winres.ResourceSet{}
	if err := rs.SetIcon(winres.ID(1), winIcon); err != nil {
		return err
	}

	vi := version.Info{}
	vi.SetFileVersion("0.5.0.0")
	vi.SetProductVersion("0.5.0.0")
	const enUS = 0x0409
	vi.Set(enUS, version.ProductName, "Vigil")
	vi.Set(enUS, version.FileDescription, "Keep Microsoft Teams active on a schedule.")
	vi.Set(enUS, version.CompanyName, "Simtabi LLC")
	vi.Set(enUS, version.LegalCopyright, "Copyright (c) 2026 Simtabi LLC")
	vi.Set(enUS, version.OriginalFilename, "vigil.exe")
	rs.SetVersionInfo(vi)

	targets := map[string]winres.Arch{
		"amd64": winres.ArchAMD64,
		"386":   winres.ArchI386,
		"arm64": winres.ArchARM64,
	}
	for name, arch := range targets {
		path := filepath.Join("..", "..", "cmd", "vigil", "rsrc_windows_"+name+".syso")
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := rs.WriteObject(f, arch); err != nil {
			_ = f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		fmt.Println("wrote", path)
	}
	return nil
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "iconz:", err)
	os.Exit(1)
}
