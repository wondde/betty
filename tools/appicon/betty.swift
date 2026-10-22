// Draws every Betty icon asset from one definition: Betty's face as a slightly
// flattened blueberry with a side-swept fringe and a small fruit stem on top.
//
//   swift tools/appicon/betty.swift
//
// writes
//   build/appicon.png                     1024 px app icon (white squircle on the macOS grid)
//   build/appicon.icon/Assets/glyph.png   glyph layer for Icon Composer (body fills the canvas)
//   assets/tray-template.png              64 px menu bar template (black + alpha)
//   frontend/public/betty-icon.png        256 px for the sign-in and About screens
// then run `wails3 task common:generate:icons` (or the generate icons command) for .icns/.ico/Assets.car.
import AppKit

let ink = CGColor(red: 0.09, green: 0.09, blue: 0.10, alpha: 1)
let white = CGColor(gray: 1, alpha: 1)

// The glyph in 1024-canvas coordinates (y down), body area 100…924.
// knockout: clear the face instead of filling it white (template images use alpha only).
func drawGlyph(_ g: CGContext, knockout: Bool = false) {
  let squash: CGFloat = 1.12, base: CGFloat = 246
  let c = CGPoint(x: 512, y: 572), rx = base * squash, ry = base / squash
  let face = CGPath(ellipseIn: CGRect(x: c.x - rx, y: c.y - ry, width: 2 * rx, height: 2 * ry), transform: nil)
  let top = c.y - ry
  g.setStrokeColor(ink); g.setFillColor(ink); g.setLineCap(.round); g.setLineJoin(.round)
  // stem
  g.setLineWidth(26)
  let stem = CGMutablePath(); stem.move(to: CGPoint(x: 512, y: top + 6)); stem.addQuadCurve(to: CGPoint(x: 540, y: top - 56), control: CGPoint(x: 512, y: top - 34))
  g.addPath(stem); g.strokePath()
  // face (white so the stem's root is covered)
  if knockout { g.setBlendMode(.clear) }
  g.setFillColor(white); g.addPath(face); g.fillPath()
  g.setBlendMode(.normal)
  // fringe, clipped to the face so it meets the outline cleanly
  g.saveGState(); g.addPath(face); g.clip()
  g.setFillColor(ink)
  let hair = CGMutablePath()
  hair.move(to: CGPoint(x: 200, y: top + 146)); hair.addLine(to: CGPoint(x: 200, y: 200)); hair.addLine(to: CGPoint(x: 824, y: 200)); hair.addLine(to: CGPoint(x: 824, y: top + 128))
  hair.addQuadCurve(to: CGPoint(x: 476, y: top + 80), control: CGPoint(x: 640, y: top + 128))
  hair.addQuadCurve(to: CGPoint(x: 200, y: top + 146), control: CGPoint(x: 330, y: top + 60))
  hair.closeSubpath(); g.addPath(hair); g.fillPath()
  g.restoreGState()
  // outline
  g.setLineWidth(32); g.addPath(face); g.strokePath()
}

func render(_ path: String, size: Int, _ draw: (CGContext) -> Void) {
  let rep = NSBitmapImageRep(bitmapDataPlanes: nil, pixelsWide: size, pixelsHigh: size, bitsPerSample: 8, samplesPerPixel: 4,
                             hasAlpha: true, isPlanar: false, colorSpaceName: .deviceRGB, bytesPerRow: 0, bitsPerPixel: 0)!
  NSGraphicsContext.saveGraphicsState()
  let ctx = NSGraphicsContext(bitmapImageRep: rep)!
  NSGraphicsContext.current = ctx
  let g = ctx.cgContext
  g.scaleBy(x: CGFloat(size) / 1024, y: CGFloat(size) / 1024)
  g.translateBy(x: 0, y: 1024); g.scaleBy(x: 1, y: -1)
  draw(g)
  NSGraphicsContext.restoreGraphicsState()
  try! rep.representation(using: .png, properties: [:])!.write(to: URL(fileURLWithPath: path))
}

// Full app icon: white squircle with a soft shadow, glyph on top.
func appIcon(_ g: CGContext) {
  g.saveGState()
  g.setShadow(offset: CGSize(width: 0, height: 10), blur: 24, color: CGColor(gray: 0, alpha: 0.18))
  g.addPath(CGPath(roundedRect: CGRect(x: 100, y: 100, width: 824, height: 824), cornerWidth: 185, cornerHeight: 185, transform: nil))
  g.setFillColor(white); g.fillPath()
  g.restoreGState()
  drawGlyph(g)
}

render("build/appicon.png", size: 1024, appIcon)
render("frontend/public/betty-icon.png", size: 256, appIcon)

// Icon Composer layer: the system draws the squircle, so map the 824 body to the whole canvas.
render("build/appicon.icon/Assets/glyph.png", size: 1024) { g in
  let k: CGFloat = 1024 / 824
  g.scaleBy(x: k, y: k); g.translateBy(x: -100, y: -100)
  drawGlyph(g)
}

// Menu bar template: silhouette only (macOS tints it), cropped close to the glyph.
render("assets/tray-template.png", size: 64) { g in
  let k: CGFloat = 1024 / 640
  g.scaleBy(x: k, y: k); g.translateBy(x: -192, y: -270)
  drawGlyph(g, knockout: true)
}
