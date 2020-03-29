package img

import (
	"context"
	"github.com/pkg/errors"

	"gopkg.in/gographics/imagick.v3/imagick"
)

func BlurPolygon(ctx context.Context, srcPath, destPath string, polygons [][][]float64) error {
	imagick.Initialize()
	defer imagick.Terminate()

	baseMask := imagick.NewMagickWand()
	defer baseMask.Destroy()
	if err := baseMask.ReadImage(srcPath); err != nil {
		return errors.Wrapf(err, "failed to open src image %s", srcPath)
	}
	if err := baseMask.BlurImage(0, 30); err != nil {
		return errors.Wrapf(err, "failed to blur src image %s", srcPath)
	}
	width, height := baseMask.GetImageWidth(), baseMask.GetImageHeight()

	//copy jpg to png to have alpha chanel
	pixel := imagick.NewPixelWand()
	defer pixel.Destroy()
	pixel.SetColor("rgba(0,0,0,0)")

	baseMaskTransparent := imagick.NewMagickWand()
	defer baseMaskTransparent.Destroy()
	if err := baseMaskTransparent.NewImage(width, height, pixel); err != nil {
		return errors.Wrap(err, "failed to create base mask transparent image")
	}
	if err := baseMaskTransparent.SetFormat("png"); err != nil {
		return errors.Wrap(err, "failed to set base mask transparent fmt to png")
	}
	if err := baseMaskTransparent.CompositeImage(baseMask, imagick.COMPOSITE_OP_OVER, true, 0, 0); err != nil {
		return errors.Wrap(err, "failed to composite base mask transparent to base mask")
	}

	//create the mask from polygons
	mask, err := maskFromPolygons(width, height, polygons)
	if err != nil {
		return errors.WithMessage(err, "failed to get polygon mask")
	}
	defer mask.Destroy()

	//apply polygon mask onto png
	if err := baseMaskTransparent.CompositeImage(mask, imagick.COMPOSITE_OP_DST_IN, true, 0, 0); err != nil {
		return errors.Wrap(err, "failed to apply mask onto png")
	}

	//compose blurred and original images
	base := imagick.NewMagickWand()
	defer base.Destroy()
	if err := base.ReadImage(srcPath); err != nil {
		return errors.Wrap(err, "failed to open src image")
	}
	if err := base.CompositeImage(baseMaskTransparent, imagick.COMPOSITE_OP_DISSOLVE, true, 0, 0); err != nil {
		return errors.Wrap(err, "failed to apply trans mask on base img")
	}
	if err := base.WriteImage(destPath); err != nil {
		return errors.Wrap(err, "failed to save dest img to disk")
	}

	return nil
}


func maskFromPolygons(width, height uint, polygons [][][]float64) (mask *imagick.MagickWand, err error) {
	pixel1 := imagick.NewPixelWand()
	defer pixel1.Destroy()
	pixel1.SetColor("rgb(0, 0, 0)")

	drawer := imagick.NewDrawingWand()
	defer drawer.Destroy()
	drawer.SetFillColor(pixel1)
	blurMask := 0
	for _, polygon := range polygons {
		if len(polygon) > 2 {
			blurMask++
			var points []imagick.PointInfo
			for _, coordinate := range polygon {
				points = append(points, imagick.PointInfo{X: coordinate[0], Y: coordinate[1]})
			}
			drawer.Polygon(points)
		}
	}

	pixel2 := imagick.NewPixelWand()
	defer pixel2.Destroy()
	pixel2.SetColor("rgba(0, 0, 0, 0)")

	mask = imagick.NewMagickWand()
	if err := mask.NewImage(width, height, pixel2); err != nil {
		return nil, errors.Wrap(err, "failed to create polygon mask")
	}
	if err := mask.SetFormat("png"); err != nil {
		return nil, errors.Wrap(err, "failed to set polygon mask fmt")
	}
	if blurMask > 0 {
		if err := mask.DrawImage(drawer); err != nil {
			return nil, errors.Wrap(err, "failed to draw polygon on mask")
		}
	}

	return mask, nil
}
