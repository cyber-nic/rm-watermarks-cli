# Fedora Installation

```
dnf install opencv opencv-devel gcc-c++

go mod tidy

make
```

# OpenCV Image Types

CV_8UC3 is an 8-bit unsigned integer matrix/image with 3 channels

```
CV_<bit-depth>{U|S|F}C(<number_of_channels>)
```

# gocv docs

https://pkg.go.dev/gocv.io/x/gocv#BitwiseAndWithMask

# Gimp

image -> mode -> grayscale
color -> threshold
https://docs.gimp.org/en/gimp-image-convert-grayscale.html

# links

https://stackoverflow.com/questions/27183946/what-does-cv-8uc3-and-the-other-types-stand-for-in-opencv
https://github.com/hybridgroup/gocv/issues/152

# Background Subtraction + Inpainting

Idea: Identify the watermark area based on repeating pattern and inpaint it with surrounding texture.
Steps:
Convert image to grayscale.
Apply morphological operations (e.g., erosion) to highlight the repeating pattern.
Threshold the image to isolate the watermark area.
Dilate the mask slightly to compensate for misalignment.
Use cv2.inpaint with the mask and Telea algorithm to fill the watermark area with surrounding texture.

# Frequency Domain Filtering

Idea: Watermark typically has high-frequency components, filter them out while preserving text.
Steps:
Convert image to frequency domain using cv2.dft.
Apply a high-pass filter to remove high-frequency components (watermark).
Invert the frequency domain image and convert back to spatial domain using cv2.idft.
Sharpen the image slightly to restore text details.

# Suggestions for improving the mask and process

## Mask Creation

- Edge Refinement: Use edge detection algorithms (Canny or Sobel) to refine mask boundaries, minimizing impact on handwritten content.
- Variable Opacity: Create a grayscale mask with varying opacity to better blend with image regions, reducing visual artifacts.
- Gradient Masks: Experiment with soft-edged masks that fade out around watermark edges to minimize abrupt transitions.
- Adaptive Thresholding: Explore adaptive thresholding techniques (e.g., Otsu's method) for more precise watermark isolation based on local image properties.

## Mask Application

- Frequency Domain: Explore working in the frequency domain (using Fourier Transform) for potentially more accurate watermark isolation and removal.
- Inpainting: Consider using inpainting techniques to fill in areas covered by watermark after removal, preserving visual consistency.
- Content-Aware Algorithms: Explore content-aware algorithms that adapt to image features and minimize impact on handwritten content.
- Machine Learning: Consider using machine learning models (e.g., convolutional neural networks) for more sophisticated watermark detection and removal.

## Specific Recommendations for Existing Code

- Kernel Size: Experiment with different kernel sizes for morphological operations to find the optimal balance between watermark removal and preservation of handwritten content.
- Thresholding Methods: Explore alternative thresholding techniques (e.g., adaptive thresholding) to better isolate the watermark.
- Foreground Text Extraction: Evaluate the effectiveness of the dilateImageToExtractForegroundText function and consider refinements to improve its accuracy.
- Mask Refinement: Implement suggested mask creation techniques (edge refinement, variable opacity, gradient masks) to enhance mask quality.

## Image Evaluation

- Visual Inspection: Carefully examine original and masked images in detail to assess visual artifacts and preservation of handwritten content.
- Objective Metrics: Use image quality metrics (e.g., PSNR, SSIM) to quantitatively measure the impact of watermark removal on image quality.

## Additional Considerations

- Watermark Characteristics: Consider the watermark's font, size, color, and transparency when refining mask creation and application methods.
- Image Quality: Higher-quality images often yield better results in watermark removal processes.
  Processing Time: Balance accuracy and efficiency based on the specific application requirements.
