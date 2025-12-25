// Package stealth - Bézier curve mouse movement implementation
// This is MANDATORY stealth technique #1
package stealth

import (
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
	"github.com/rs/zerolog"

	"linkedin-automation/internal/config"
)

// Point represents a 2D coordinate
type Point struct {
	X, Y float64
}

// MouseController handles human-like mouse movements using Bézier curves
type MouseController struct {
	config *config.StealthConfig
	logger zerolog.Logger
	lastX  float64
	lastY  float64
}

// NewMouseController creates a new mouse controller
func NewMouseController(cfg *config.StealthConfig, logger zerolog.Logger) *MouseController {
	return &MouseController{
		config: cfg,
		logger: logger.With().Str("module", "mouse").Logger(),
		lastX:  0,
		lastY:  0,
	}
}

// MoveTo moves the mouse to target coordinates using a Bézier curve
func (m *MouseController) MoveTo(page *rod.Page, targetX, targetY float64) error {
	m.logger.Debug().
		Float64("fromX", m.lastX).Float64("fromY", m.lastY).
		Float64("toX", targetX).Float64("toY", targetY).
		Msg("Moving mouse with Bézier curve")

	// Generate Bézier curve points
	points := m.generateBezierPath(m.lastX, m.lastY, targetX, targetY)

	mouse := page.Mouse

	// Move along the curve with variable timing
	for i, point := range points {
		// Calculate speed factor (slower at start and end, faster in middle)
		progress := float64(i) / float64(len(points))
		speedFactor := m.calculateSpeedFactor(progress)

		// Base delay between movements (5-15ms)
		baseDelay := 5 + rand.Intn(10)
		delay := time.Duration(float64(baseDelay)/speedFactor) * time.Millisecond

		// Move to point using Rod's MustMoveTo
		mouse.MustMoveTo(point.X, point.Y)

		time.Sleep(delay)
	}

	// Apply overshoot if enabled
	if m.config.EnableOvershoot && rand.Float64() < 0.3 {
		m.applyOvershoot(mouse, targetX, targetY)
	}

	// Update last position
	m.lastX = targetX
	m.lastY = targetY

	return nil
}

// MoveToElement moves mouse to the center of an element with human-like motion
func (m *MouseController) MoveToElement(page *rod.Page, element *rod.Element) error {
	// Get element position and size
	box, err := element.Shape()
	if err != nil {
		return err
	}

	if len(box.Quads) == 0 {
		return nil
	}

	// Calculate center with slight randomization
	quad := box.Quads[0]
	centerX := (quad[0] + quad[2] + quad[4] + quad[6]) / 4
	centerY := (quad[1] + quad[3] + quad[5] + quad[7]) / 4

	// Add small random offset (±5 pixels) to avoid exact center
	offsetX := (rand.Float64() - 0.5) * 10
	offsetY := (rand.Float64() - 0.5) * 10

	return m.MoveTo(page, centerX+offsetX, centerY+offsetY)
}

// Click performs a human-like click at the current position
func (m *MouseController) Click(page *rod.Page) error {
	mouse := page.Mouse

	// Small pre-click delay
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	// Mouse down
	if err := mouse.Down(proto.InputMouseButtonLeft, 1); err != nil {
		return err
	}

	// Hold duration (50-150ms like a real click)
	time.Sleep(time.Duration(50+rand.Intn(100)) * time.Millisecond)

	// Mouse up
	if err := mouse.Up(proto.InputMouseButtonLeft, 1); err != nil {
		return err
	}

	// Small post-click delay
	time.Sleep(time.Duration(100+rand.Intn(200)) * time.Millisecond)

	return nil
}

// ClickElement moves to an element and clicks it
func (m *MouseController) ClickElement(page *rod.Page, element *rod.Element) error {
	// Move to element first
	if err := m.MoveToElement(page, element); err != nil {
		return err
	}

	// Then click
	return m.Click(page)
}

// generateBezierPath generates points along a cubic Bézier curve
func (m *MouseController) generateBezierPath(startX, startY, endX, endY float64) []Point {
	// Calculate distance to determine number of points
	distance := math.Sqrt(math.Pow(endX-startX, 2) + math.Pow(endY-startY, 2))
	numPoints := int(math.Max(20, distance/10)) // At least 20 points, more for longer distances

	// Generate control points with randomization
	// Control points create the curve shape
	ctrl1, ctrl2 := m.generateControlPoints(startX, startY, endX, endY, distance)

	points := make([]Point, numPoints)

	for i := 0; i < numPoints; i++ {
		t := float64(i) / float64(numPoints-1)

		// Cubic Bézier formula: B(t) = (1-t)³P0 + 3(1-t)²tP1 + 3(1-t)t²P2 + t³P3
		x := m.cubicBezier(t, startX, ctrl1.X, ctrl2.X, endX)
		y := m.cubicBezier(t, startY, ctrl1.Y, ctrl2.Y, endY)

		// Add micro-jitter for more natural movement
		if i > 0 && i < numPoints-1 {
			x += (rand.Float64() - 0.5) * 2
			y += (rand.Float64() - 0.5) * 2
		}

		points[i] = Point{X: x, Y: y}
	}

	return points
}

// generateControlPoints creates control points for the Bézier curve
func (m *MouseController) generateControlPoints(startX, startY, endX, endY, distance float64) (Point, Point) {
	// Perpendicular offset for curve (creates arc)
	// Randomize the curvature amount
	curvature := distance * (0.1 + rand.Float64()*0.3)
	if rand.Float64() < 0.5 {
		curvature = -curvature
	}

	// Calculate perpendicular direction
	dx := endX - startX
	dy := endY - startY
	length := math.Sqrt(dx*dx + dy*dy)
	if length == 0 {
		length = 1
	}

	// Perpendicular vector (rotated 90 degrees)
	perpX := -dy / length
	perpY := dx / length

	// Control point 1: closer to start with some perpendicular offset
	ctrl1 := Point{
		X: startX + dx*0.25 + perpX*curvature*(0.5+rand.Float64()*0.5),
		Y: startY + dy*0.25 + perpY*curvature*(0.5+rand.Float64()*0.5),
	}

	// Control point 2: closer to end with perpendicular offset
	ctrl2 := Point{
		X: startX + dx*0.75 + perpX*curvature*(0.5+rand.Float64()*0.5),
		Y: startY + dy*0.75 + perpY*curvature*(0.5+rand.Float64()*0.5),
	}

	return ctrl1, ctrl2
}

// cubicBezier calculates a point on a cubic Bézier curve
func (m *MouseController) cubicBezier(t, p0, p1, p2, p3 float64) float64 {
	mt := 1 - t
	return mt*mt*mt*p0 + 3*mt*mt*t*p1 + 3*mt*t*t*p2 + t*t*t*p3
}

// calculateSpeedFactor returns a speed multiplier based on position in path
// Creates ease-in-ease-out effect
func (m *MouseController) calculateSpeedFactor(progress float64) float64 {
	// Use sine curve for natural acceleration/deceleration
	// sin(π * progress) gives 0 at start, 1 at middle, 0 at end
	// We invert and scale it for speed factor
	baseFactor := math.Sin(math.Pi * progress)

	// Scale between min and max speed from config
	minSpeed := m.config.MouseSpeedMin
	maxSpeed := m.config.MouseSpeedMax

	return minSpeed + baseFactor*(maxSpeed-minSpeed)
}

// applyOvershoot simulates overshooting the target and correcting
func (m *MouseController) applyOvershoot(mouse *rod.Mouse, targetX, targetY float64) {
	// Overshoot by 5-15 pixels in a random direction
	overshootDist := 5 + rand.Float64()*10
	angle := rand.Float64() * 2 * math.Pi

	overshootX := targetX + math.Cos(angle)*overshootDist
	overshootY := targetY + math.Sin(angle)*overshootDist

	// Move to overshoot position
	mouse.MustMoveTo(overshootX, overshootY)
	time.Sleep(time.Duration(30+rand.Intn(50)) * time.Millisecond)

	// Correct back to target
	mouse.MustMoveTo(targetX, targetY)
	time.Sleep(time.Duration(20+rand.Intn(30)) * time.Millisecond)

	m.logger.Debug().Msg("Applied mouse overshoot correction")
}

// SetPosition updates the tracked mouse position without moving
func (m *MouseController) SetPosition(x, y float64) {
	m.lastX = x
	m.lastY = y
}

// GetPosition returns the current tracked mouse position
func (m *MouseController) GetPosition() (float64, float64) {
	return m.lastX, m.lastY
}
