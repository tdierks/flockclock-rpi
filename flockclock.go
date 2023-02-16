package main

/*
#include <sys/ioctl.h>
#include <linux/fb.h>
struct fb_fix_screeninfo getFixScreenInfo(int fd) {
	struct fb_fix_screeninfo info;
	ioctl(fd, FBIOGET_FSCREENINFO, &info);
	return info;
}
struct fb_var_screeninfo getVarScreenInfo(int fd) {
	struct fb_var_screeninfo info;
	ioctl(fd, FBIOGET_VSCREENINFO, &info);
	return info;
}
*/
import "C"
import (
//   "errors"
  "fmt"
 	"image"
 	"image/color"
 	"image/draw"
	"os"
	"syscall"
	"time"
)

func main() {
  fb, err := fbImage("/dev/fb0")
  if err != nil {
    panic(err)
  }
  defer fb.Close()
  
	start := time.Now()
	for n:=0; n < 256; n++ {
	  c := image.NewUniform(color.RGBA{uint8(n), 0, uint8(255-n), 255})
  	draw.Draw(fb, fb.Bounds(), c, image.ZP, draw.Src)
  }
  duration := time.Since(start)
  fmt.Println(duration)
}

func fbImage(devName string) (*FrameBuffer, error) {
  file, err := os.OpenFile(devName, os.O_RDWR, os.ModeDevice)
  if err != nil {
    return nil, err
  }

  fixInfo := C.getFixScreenInfo(C.int(file.Fd()))
  varInfo := C.getVarScreenInfo(C.int(file.Fd()))
  
  pixels, err := syscall.Mmap(
                    int(file.Fd()),
                    0,
                    int(varInfo.xres * varInfo.yres * varInfo.bits_per_pixel/8),
                    syscall.PROT_READ | syscall.PROT_WRITE,
                    syscall.MAP_SHARED,
                    )
  if err != nil {
    file.Close()
    return nil, err
  }
  
  fmt.Printf("Pixels at %p %d x %d, %d bpp, %d stride \n", pixels, varInfo.xres, varInfo.yres, varInfo.bits_per_pixel, int(fixInfo.line_length))

  fmt.Printf("varInfo: %+v\n", varInfo)

  for y:=0; y < int(varInfo.yres); y++ {
    for x:= 0; x < int(varInfo.xres); x++ {
      n := y * int(fixInfo.line_length) + x * int(varInfo.bits_per_pixel)/8
      pixels[n] = 0
      pixels[n+1] = 0
      pixels[n+2] = 0
      pixels[n+3] = 0
      pixels[n] = byte(x & 0xFF);
      pixels[n+1] = byte(((x + y) / 4) & 0xFF)
      pixels[n+2] = byte(y & 0xFF)
    }
  }

  return  &FrameBuffer {
            file,
            pixels,
            int(fixInfo.line_length),
            image.Rect(0, 0, int(varInfo.xres), int(varInfo.yres)),
            color.RGBAModel,
          },
          nil
}

type FrameBuffer struct {
  file *os.File
  pixels []byte
  pitch int
  bounds image.Rectangle
  colorModel color.Model
}

func (fb *FrameBuffer) Close() {
  syscall.Munmap(fb.pixels)
  fb.file.Close()
}

func (fb *FrameBuffer) Bounds() image.Rectangle {
  return fb.bounds
}

func (fb *FrameBuffer) ColorModel() color.Model {
  return fb.colorModel
}

func (fb *FrameBuffer) At(x, y int) color.Color {
  if x < fb.bounds.Min.X || x > fb.bounds.Max.X || 
    y < fb.bounds.Min.Y || y >= fb.bounds.Max.Y {
    return color.Black
  }
  i := y * fb.pitch + 4 * x
  return color.RGBA{fb.pixels[i+2], fb.pixels[i+1], fb.pixels[i], 0xFF}
}

func (fb *FrameBuffer) Set(x,y int, c color.Color) {
  if x >= 0 && x < fb.bounds.Max.X &&
		y >= 0 && y < fb.bounds.Max.Y {
		r, g, b, a := c.RGBA()
		if a > 0 {
      i := y * fb.pitch + 4 * x
      fb.pixels[i] = byte(b)
      fb.pixels[i+1] = byte(g)
      fb.pixels[i+2] = byte(r)
		}
  }
}