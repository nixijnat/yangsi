package img

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"yangsi/log"

	rz "github.com/nfnt/resize"
)

const (
	fmtPNG  = "png"
	fmtJPG  = "jpg"
	fmtJPEG = "jpeg"
)

type Image struct {
	Dir      string
	Basename string
	Format   string
	ModTime  string
	Raw      image.Image
	rawBytes []byte
}

func NewImage(dir string, filename string) (*Image, error) {
	index := strings.LastIndexByte(filename, '.')
	if index <= 0 || index >= len(filename) {
		return nil, log.NewError("invalid image file: %s, %s", dir, filename)
	}
	format := strings.ToLower(string(filename[index+1:]))
	if err := varifyFormat(format); err != nil {
		return nil, err
	}

	stat, err := os.Stat(fmt.Sprintf("%s/%s", dir, filename))
	if err != nil {
		return nil, log.NewWarn("stat file failed: %s, %s", filename, err.Error())
	}
	return &Image{
		Dir:      dir,
		Basename: string(filename[:index]),
		Format:   format,
		ModTime:  stat.ModTime().Format("2006-01-02 15:04:05"),
	}, nil
}

func (i *Image) Path() string {
	return fmt.Sprintf("%s/%s.%s", i.Dir, i.Basename, i.Format)
}

func (i *Image) Load() (err error) {
	i.Raw, err = load(i.Path(), i.Format)
	return
}

func (i *Image) Store() (string, error) {
	var err error
	if len(i.rawBytes) == 0 {
		i.rawBytes, err = i.Compress()
		if err != nil {
			return "", err
		}
	}
	var path = fmt.Sprintf("%s/%s_o.%s", nowOutDir, i.Basename, i.Format)
	err = ioutil.WriteFile(path, i.rawBytes, os.ModePerm)
	if err != nil {
		return "", err
	}
	return path, nil
}

func (i *Image) Compress() ([]byte, error) {
	var err error
	i.rawBytes, err = compress(i.Raw, &jpeg.Options{
		Quality: localConf.OutImg.Quality,
	})
	return i.rawBytes, err
}

func (i *Image) Resize() {
	i.Raw = resize(i.Raw, rzOption{
		MaxPixel: localConf.OutImg.MaxPixel,
	})
}

func (i *Image) Smaller() ([]byte, error) {
	i.Resize()
	return i.Compress()
}

func varifyFormat(format string) error {
	switch format {
	case fmtJPEG, fmtJPG, fmtPNG:
		return nil
	}
	return log.NewError("invalid image format: %s", format)
}

func load(path, format string) (image.Image, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var img image.Image
	switch format {
	case fmtJPEG, fmtJPG:
		img, err = jpeg.Decode(file)
	case fmtPNG:
		img, err = png.Decode(file)
	}
	if err != nil {
		return nil, err
	}
	return img, nil
}

type rzOption struct {
	Func     rz.InterpolationFunction
	MaxPixel uint
}

func calSize(oWidth, oHeight, maxPixel uint) (uint, uint, bool) {
	if maxPixel <= 0 || (oWidth <= maxPixel && oHeight <= maxPixel) {
		return oWidth, oHeight, false
	}
	if oWidth > oHeight {
		return maxPixel, oHeight * maxPixel / oWidth, true
	} else {
		return oWidth * maxPixel / oHeight, maxPixel, true
	}
}

func resize(img image.Image, o rzOption) image.Image {
	rect := img.Bounds()
	width, height := uint(rect.Dx()), uint(rect.Dy())
	width, height, doResize := calSize(width, height, o.MaxPixel)
	if !doResize {
		return img
	}
	var fn rz.InterpolationFunction = o.Func
	if fn < rz.NearestNeighbor || fn > rz.Lanczos3 {
		fn = rz.NearestNeighbor
	}
	return rz.Resize(width, height, img, fn)
}

func compress(img image.Image, option *jpeg.Options) ([]byte, error) {
	var buf = new(bytes.Buffer)
	err := jpeg.Encode(buf, img, option)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

///

type config struct {
	OutDir string `json:"out_dir"`
	OutImg struct {
		MaxPixel uint `json:"max_pixel"`
		Quality  int  `json:"quality"`
	} `json:"out_img"`
}

func (c *config) check() error {
	if c.OutDir == "" || c.OutImg.MaxPixel == 0 ||
		c.OutImg.Quality == 0 || c.OutImg.Quality > 100 {
		return log.NewError("invalid img config: %+v", *c)
	}
	return nil

}

var (
	localConf config
	nowOutDir string
)

func Init(str json.RawMessage) error {
	err := json.Unmarshal(str, &localConf)
	if err != nil {
		return log.NewError("unmarshal img config failed: %s, %s", err.Error(), string(str))
	}
	err = localConf.check()
	if err != nil {
		return err
	}
	nowOutDir = fmt.Sprintf("%s/%s", localConf.OutDir, time.Now().Format("20060102"))
	_, err = os.Stat(nowOutDir)
	if err != nil {
		if !os.IsNotExist(err) {
			return log.NewError("out dir failed: %s, %s", nowOutDir, err.Error())
		}
		err = os.MkdirAll(nowOutDir, os.ModePerm)
		if err != nil {
			return log.NewError("mkdir failed: %s, %s", nowOutDir, err.Error())
		}
	}
	return nil
}
