package render

// Ramps are character sequences ordered from "lightest" (least ink) to
// "darkest" (most ink). At render time a pixel's brightness picks an index.
//
// Named ramps make the CLI surface friendlier: a user can pick a vibe
// (--ramp dense) without thinking about which exact characters look good.
var Ramps = map[string]string{
	"standard": " .:-=+*#%@",
	"dense":    " .'`,^:;Il!i><~+_-?][}{1)(|/tfjrxnuvczXYUJCLQ0OZmwqpdbkhao*#MW&8%B@$",
	"blocks":   " ░▒▓█",
	"minimal":  " .oO@",
	"binary":   " █",
}

// DefaultRamp is the ramp name used when no --ramp flag is given.
const DefaultRamp = "standard"
