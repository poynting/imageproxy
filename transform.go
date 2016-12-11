// Copyright 2013 Google Inc. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package imageproxy

import (
	"bytes"
	"image"
	_ "image/gif" // register gif format
	"image/jpeg"
	"image/png"

	"github.com/disintegration/imaging"
	"github.com/rwcarlsen/goexif/exif"
	"willnorris.com/go/gifresize"
)

// default compression quality of resized jpegs
const defaultQuality = 95

// resample filter used when resizing images
var resampleFilter = imaging.Lanczos

// Transform the provided image.  img should contain the raw bytes of an
// encoded image in one of the supported formats (gif, jpeg, or png).  The
// bytes of a similarly encoded image is returned.
func Transform(img []byte, opt Options) ([]byte, error) {
	if !opt.transform() {
		// bail if no transformation was requested
		return img, nil
	}

	// decode image
	m, format, err := image.Decode(bytes.NewReader(img))
	if err != nil {
		return nil, err
	}

	// Auto set rotation/fh/fv options based on exif
	opt, err = autoRotate(img, opt)
	if err != nil {
		return nil, err
	}

	// transform and encode image
	buf := new(bytes.Buffer)
	switch format {
	case "gif":
		fn := func(img image.Image) image.Image {
			return transformImage(img, opt)
		}
		err = gifresize.Process(buf, bytes.NewReader(img), fn)
		if err != nil {
			return nil, err
		}
	case "jpeg":
		quality := opt.Quality
		if quality == 0 {
			quality = defaultQuality
		}

		m = transformImage(m, opt)
		err = jpeg.Encode(buf, m, &jpeg.Options{Quality: quality})
		if err != nil {
			return nil, err
		}
	case "png":
		m = transformImage(m, opt)
		err = png.Encode(buf, m)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

// resizeParams determines if the image needs to be resized, and if so, the
// dimensions to resize to.
func resizeParams(m image.Image, opt Options) (w, h int, resize bool) {
	// convert percentage width and height values to absolute values
	imgW := m.Bounds().Max.X - m.Bounds().Min.X
	imgH := m.Bounds().Max.Y - m.Bounds().Min.Y
	if 0 < opt.Width && opt.Width < 1 {
		w = int(float64(imgW) * opt.Width)
	} else if opt.Width < 0 {
		w = 0
	} else {
		w = int(opt.Width)
	}
	if 0 < opt.Height && opt.Height < 1 {
		h = int(float64(imgH) * opt.Height)
	} else if opt.Height < 0 {
		h = 0
	} else {
		h = int(opt.Height)
	}

	// never resize larger than the original image unless specifically allowed
	if !opt.ScaleUp {
		if w > imgW {
			w = imgW
		}
		if h > imgH {
			h = imgH
		}
	}

	// if requested width and height match the original, skip resizing
	if (w == imgW || w == 0) && (h == imgH || h == 0) {
		return 0, 0, false
	}

	return w, h, true
}

// transformImage modifies the image m based on the transformations specified
// in opt.
func transformImage(m image.Image, opt Options) image.Image {
	// resize if needed
	if w, h, resize := resizeParams(m, opt); resize {
		if opt.Fit {
			m = imaging.Fit(m, w, h, resampleFilter)
		} else {
			if w == 0 || h == 0 {
				m = imaging.Resize(m, w, h, resampleFilter)
			} else {
				m = imaging.Thumbnail(m, w, h, resampleFilter)
			}
		}
	}

	// flip
	if opt.FlipVertical {
		m = imaging.FlipV(m)
	}
	if opt.FlipHorizontal {
		m = imaging.FlipH(m)
	}

	// rotate
	switch opt.Rotate {
	case 90:
		m = imaging.Rotate90(m)
	case 180:
		m = imaging.Rotate180(m)
	case 270:
		m = imaging.Rotate270(m)
	}

	return m
}

func autoRotate(img []byte, opt Options) (Options, error) {
	if opt.Rotate != 360 {
		return opt, nil
	}
	// decode exif
	x, err := exif.Decode(bytes.NewReader(img))
	if err != nil {
		return opt, err
	}
	orientation, err := x.Get(exif.Orientation)
	if err != nil {
		return opt, err
	}
	intOrientation, err := orientation.Int(0)
	if err != nil {
		return opt, err
	}

	switch intOrientation {
	case 1:
		opt.Rotate = 0
		opt.FlipHorizontal = false
		opt.FlipVertical = false
	case 2:
		opt.Rotate = 0
		opt.FlipHorizontal = true
		opt.FlipVertical = false
	case 3:
		opt.Rotate = 180
		opt.FlipHorizontal = false
		opt.FlipVertical = false
	case 4:
		opt.Rotate = 0
		opt.FlipHorizontal = false
		opt.FlipVertical = true
	case 5:
		opt.Rotate = 270
		opt.FlipHorizontal = false
		opt.FlipVertical = true
	case 6:
		opt.Rotate = 270
		opt.FlipHorizontal = false
		opt.FlipVertical = false
	case 7:
		opt.Rotate = 90
		opt.FlipHorizontal = false
		opt.FlipVertical = true
	case 8:
		opt.Rotate = 90
		opt.FlipHorizontal = false
		opt.FlipVertical = false
	}

	return opt, nil
}
