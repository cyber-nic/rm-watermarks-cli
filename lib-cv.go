package main

import (
	"image"

	"gocv.io/x/gocv"
)

// RemoveWatermark removes a watermark from an image using inpainting
func RemoveWatermark(src, mask gocv.Mat) gocv.Mat {
	inpaintedImage := gocv.NewMat()

	var inpaintRadius float32 = 3
	telea := gocv.InpaintMethods(gocv.Telea)
	// ns := gocv.InpaintMethods(gocv.NS)

	gocv.Inpaint(src, mask, &inpaintedImage, inpaintRadius, telea)
	return inpaintedImage.Clone()
}

// ComputeWatermarkMask computes a mask for the watermark in the input image.
// This excludes the foreground text from the watermark mask.
// Return the binary and foreground text images for debugging purposes.
func ComputeWatermarkMask(img, maskTpl gocv.Mat, gravity string, thresh float32, excludeForeground bool) (gocv.Mat, gocv.Mat, gocv.Mat, gocv.Mat) {
	// Crop the watermark mask template to match src image size
	crop := CropWithGravity(maskTpl, img.Cols(), img.Rows(), gravity)
	defer crop.Close()

	// Compute binary image using mean threshold to extract the foreground text with the watermark
	bin := ConvertToBinaryUsingMeanThreshold(img, thresh)
	defer bin.Close()

	// Extract foreground text from binary image
	fg := ExtractForegroundText(bin)
	defer fg.Close()

	// Subtract the text area from the watermark mask
	mask := gocv.NewMat()
	defer mask.Close()

	if excludeForeground {
		gocv.BitwiseAnd(crop, fg, &mask)
	} else {
		mask = crop.Clone()
	}

	return crop.Clone(), bin.Clone(), fg.Clone(), mask.Clone()
}

// ComputeImageChannelMetrics calculates key statistical measures, including mean and standard deviation,
// across color channels of an image to quantify its brightness, color balance, and contrast. Where:
// b captures the overall average brightness of the image
// m represents the average of the channel-wise means, indicating the image's overall color balance
// s measures the average spread of pixel values across channels, reflecting the image's overall contrast or detail level
func ComputeImageChannelMetrics(img gocv.Mat) (float32, float32, float32) {
	// Create Mats to store mean and standard deviation
	mean := gocv.NewMat()
	stdDev := gocv.NewMat()

	// Calculate the mean color across all channels
	gocv.MeanStdDev(img, &mean, &stdDev)

	// Represents the overall mean pixel value of the original image img
	// Calculated using ComputeMatMean(img), directly averaging all pixel values across all channels.
	b := ComputeMatMean(img)

	// Represents the mean of the mean values across all channels.
	// Calculated using ComputeMatMean(mean), averaging the mean values stored in the mean Mat.
	// Provides a single-value summary of the overall central tendency of the image's colors.
	m := ComputeMatMean(mean)

	// Represents the mean of the standard deviation values across all channels.
	// Calculated using ComputeMatMean(stdDev), averaging the standard deviation values stored in the stdDev Mat.
	// Provides a single-value summary of the overall spread or variability of the image's pixel values.
	s := ComputeMatMean(stdDev)

	return b, m, s
}

func ConvertToBinaryUsingMeanThreshold(img gocv.Mat, t float32) gocv.Mat {
	// Convert to grayscale if it's a color image
	if img.Channels() > 1 {
		gocv.CvtColor(img, &img, gocv.ColorBGRToGray)
	}

	// Apply thresholding using the mean value as the threshold
	bin := gocv.NewMat()
	gocv.Threshold(img, &bin, t, 255, gocv.ThresholdBinary)

	// Convert back to BGR (3 channels) while keeping it grayscale
	bgr := gocv.NewMat()
	gocv.CvtColor(bin, &bgr, gocv.ColorGrayToBGR)

	return bgr.Clone()
}

func ExtractForegroundText(img gocv.Mat) gocv.Mat {
	// Convert to grayscale
	gray := gocv.NewMat()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// current extraction strategy is to dilate the image to enhance the watermark features
	return DilateImageToExtractForegroundText(gray)
}

// DilateImageToExtractForegroundText dilates the input image to enhance the watermark features.
func DilateImageToExtractForegroundText(gray gocv.Mat) gocv.Mat {
	// Apply thresholding to highlight watermark
	// Assuming the watermark is lighter than the background
	thresholdImage := gocv.NewMat()
	defer thresholdImage.Close()

	// ThresholdBinary  0
	// ThresholdBinaryInv  1
	// ThresholdTrunc  2
	// ThresholdToZero  3
	// ThresholdToZeroInv  4
	// ThresholdMask  7
	// ThresholdOtsu  8
	// ThresholdTriangle  16

	// var t gocv.ThresholdType = gocv.ThresholdBinaryInv + gocv.ThresholdOtsu
	// gocv.Threshold(gray, &thresholdImage, thresh, maxvalue, t)
	var thresh float32 // this value is ignored when using the Otsu algorithm

	var t gocv.ThresholdType = gocv.ThresholdBinaryInv + gocv.ThresholdOtsu

	gocv.Threshold(gray, &thresholdImage, thresh, 255, t)
	// gocv.NewWindow("Thresh").IMShow(thresholdImage)

	// Use morphology to enhance the watermark features
	// Create a kernel for morphological operations
	ksize := image.Point{X: 3, Y: 3} // image.Point{X: 3, Y: 3}
	// gocv.MorphRect
	// gocv.MorphEllipse
	// gocv.MorphCross
	kernel := gocv.GetStructuringElement(gocv.MorphRect, ksize)
	// kernel := gocv.GetStructuringElement(gocv.MorphCross, image.Point{X: 5, Y: 1}) // Example for horizontal lines

	defer kernel.Close()

	// Dilate to enhance the features of the watermark
	dilatedImage := gocv.NewMat()
	defer dilatedImage.Close()
	gocv.Dilate(thresholdImage, &dilatedImage, kernel)
	// gocv.NewWindow("Dilated").IMShow(dilatedImage)

	// Invert the colors to get the watermark in black on a white background
	invertedImage := gocv.NewMat()
	defer invertedImage.Close()
	gocv.BitwiseNot(dilatedImage, &invertedImage)

	// Clone and return the image with the watermark
	return invertedImage.Clone()
}

// CropGravitySouthWest crops the input image to match the specified width and height,
// starting from the bottom left corner of the image.
func CropGravitySouthWest(img gocv.Mat, width, height int) gocv.Mat {
	imgSize := img.Size()

	// Calculate the starting point (bottom left) for the crop
	startX := 0
	startY := imgSize[0] - height
	if startY < 0 {
		startY = 0
	}

	// Ensure the width and height do not exceed the image's dimensions
	if startX+width > imgSize[1] {
		width = imgSize[1] - startX
	}
	if startY+height > imgSize[0] {
		height = imgSize[0] - startY
	}

	// Define the region of interest (ROI) and crop
	rect := image.Rect(startX, startY, startX+width, startY+height)
	cropped := img.Region(rect)

	return cropped
}

func CropWithGravity(img gocv.Mat, width, height int, gravity string) gocv.Mat {
	imgSize := img.Size()

	// Calculate starting coordinates based on gravity
	startX, startY := 0, 0
	switch gravity {
	case "north":
			startY = 0
	case "north-west":
			startY = 0
			startX = 0
	case "north-east":
		startY = 0
		startX = imgSize[1] - width
			if startX < 0 {
					startX = 0
					width = imgSize[1] // Adjust width to fit
			}
	case "west":
			startX = 0
	case "east":
			startX = imgSize[1] - width
			if startX < 0 {
					startX = 0
					width = imgSize[1] // Adjust width to fit
			}
	case "south":
		startY = imgSize[0] - height
		if startY < 0 {
				startY = 0
				height = imgSize[0] // Adjust height to fit
		}
	case "south-west":
		startX = 0
		startY = imgSize[0] - height
		if startY < 0 {
				startY = 0
				height = imgSize[0] // Adjust height to fit
		}
	case "south-east":
		startY = imgSize[0] - height
		if startY < 0 {
				startY = 0
				height = imgSize[0] // Adjust height to fit
		}
		startX = imgSize[1] - width
			if startX < 0 {
					startX = 0
					width = imgSize[1] // Adjust width to fit
			}
	default:
			// Handle invalid gravity (optional: return an error or log a warning)
			panic("invalid gravity")
	}

	// Ensure width and height do not exceed image dimensions
	if startX+width > imgSize[1] {
			width = imgSize[1] - startX
	}
	if startY+height > imgSize[0] {
			height = imgSize[0] - startY
	}

	// Define the region of interest (ROI) and crop
	rect := image.Rect(startX, startY, startX+width, startY+height)
	cropped := img.Region(rect)

	return cropped
}

// InvertColors inverts the colors of the input image.
func InvertColors(img gocv.Mat) gocv.Mat {
	invertedImg := gocv.NewMat()
	gocv.BitwiseNot(img, &invertedImg)

	return invertedImg.Clone()
}

// ComputeMatMean calculates the mean (average) pixel value of an image represented as a gocv.Mat object.
// This function can be useful for determining if an image is dark (eg. carbon copy)
func ComputeMatMean(img gocv.Mat) float32 {
	// compute total pixels
	totalPixels := img.Rows() * img.Cols()
	sum := float32(0.0)

	for y := 0; y < img.Rows(); y++ {
		for x := 0; x < img.Cols(); x++ {
			pixel := img.GetUCharAt(y, x)
			sum += float32(pixel)
		}
	}

	return sum / float32(totalPixels)
}

// RemoveColors converts the input image to grayscale, then converts it back to BGR (3 channels).
func RemoveColors(img gocv.Mat) gocv.Mat {
	// Convert to grayscale
	gray := gocv.NewMat()
	gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

	// Convert back to BGR (3 channels) while keeping it grayscale
	bgr := gocv.NewMat()
	gocv.CvtColor(gray, &bgr, gocv.ColorGrayToBGR)

	return bgr.Clone() // Return the 3-channel grayscale image
}

// IsColor checks if the input image contains color pixels above a certain threshold.
func IsColor(img gocv.Mat) bool {
	// Convert to HSV color space (preferred for color detection)
	hsv := gocv.NewMat()
	gocv.CvtColor(img, &hsv, gocv.ColorBGRToHSV)
	defer hsv.Close()

	// red, green, blue, alpha
	minRange := gocv.Scalar{Val1: 32, Val2: 32, Val3: 32, Val4: 255}
	maxRange := gocv.Scalar{Val1: 255, Val2: 255, Val3: 255, Val4: 255}

	// Create a mask for the color
	mask := gocv.NewMat()
	gocv.InRangeWithScalar(hsv, minRange, maxRange, &mask)
	defer mask.Close()

	// Check if any pixels match the color mask
	return gocv.CountNonZero(mask) > 0
}
