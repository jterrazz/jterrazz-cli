package theme

// =============================================================================
// Unicode Icons & Symbols
// =============================================================================

const (
	// Navigation
	IconSelected = "›"

	// Status
	IconCheck   = "✓"
	IconCross   = "✗"
	IconWarning = "!"
	IconBullet  = "•"

	// Service status
	IconServiceOn  = "●"
	IconServiceOff = "○"

	// Progress
	IconProgressFull  = "█"
	IconProgressEmpty = "░"

	// Loading
	IconLoading = "◌"
)

// =============================================================================
// Spinner Frames
// =============================================================================

// BrailleSpinner is the braille dots spinner animation frames
var BrailleSpinner = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// =============================================================================
// Box Drawing Characters
// =============================================================================

// Rounded box (for subsections)
const (
	BoxRoundedTopLeft     = "╭"
	BoxRoundedTopRight    = "╮"
	BoxRoundedBottomLeft  = "╰"
	BoxRoundedBottomRight = "╯"
	BoxRoundedHorizontal  = "─"
	BoxRoundedVertical    = "│"
	BoxRoundedTeeLeft     = "├"
	BoxRoundedTeeRight    = "┤"
)

// Thick box (for section headers)
const (
	BoxThickTopLeft     = "┏"
	BoxThickTopRight    = "┓"
	BoxThickBottomLeft  = "┗"
	BoxThickBottomRight = "┛"
	BoxThickHorizontal  = "━"
	BoxThickVertical    = "┃"
)
