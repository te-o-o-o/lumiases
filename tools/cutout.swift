// Cut the subject out of a photo with macOS Vision (the engine behind
// "Remove Background"), crop to it, optionally keep a part of it, and
// write a transparent PNG.
// macOS only (Vision, macOS 14+); a helper to prepare heads, not part of lumiases.
// Usage: swift tools/cutout.swift in.jpg heads/out.png [x0 y0 x1 y1]  (fractions of the cut-out, to keep the head)
import CoreImage
import Foundation
import Vision

let args = CommandLine.arguments
guard args.count == 3 || args.count == 7 else {
    print("usage: cutout in.jpg out.png [x0 y0 x1 y1]")
    exit(2)
}
let handler = VNImageRequestHandler(url: URL(fileURLWithPath: args[1]))
let request = VNGenerateForegroundInstanceMaskRequest()
try handler.perform([request])
guard let result = request.results?.first else {
    print("no subject found in \(args[1])")
    exit(1)
}
let buffer = try result.generateMaskedImage(ofInstances: result.allInstances, from: handler, croppedToInstancesExtent: true)
var image = CIImage(cvPixelBuffer: buffer)
if args.count == 7 {
    let f = args[3...6].map { CGFloat(Double($0)!) }
    let e = image.extent
    // CIImage's origin is bottom-left; the fractions are given top-down.
    image = image.cropped(to: CGRect(x: e.minX + f[0] * e.width, y: e.minY + (1 - f[3]) * e.height,
                                     width: (f[2] - f[0]) * e.width, height: (f[3] - f[1]) * e.height))
}
try CIContext().writePNGRepresentation(of: image, to: URL(fileURLWithPath: args[2]), format: .RGBA8,
                                       colorSpace: CGColorSpace(name: CGColorSpace.sRGB)!)
print("wrote \(args[2]) \(Int(image.extent.width))x\(Int(image.extent.height))")
