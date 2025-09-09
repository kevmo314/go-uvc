package transfers

import (
	"fmt"
	"sort"
)

// FormatPreference defines user preferences for audio format selection
type FormatPreference struct {
	PreferredSampleRate    uint32
	PreferredChannels      int
	PreferredBitsPerSample int
	PreferCompressed       bool   // Prefer compressed formats if available
	MaxBitRate             uint16 // For compressed formats
}

// DefaultFormatPreference returns sensible defaults
func DefaultFormatPreference() FormatPreference {
	return FormatPreference{
		PreferredSampleRate:    48000,
		PreferredChannels:      2,
		PreferredBitsPerSample: 16,
		PreferCompressed:       false,
	}
}

// FormatScore represents how well a format matches preferences
type FormatScore struct {
	Interface *AudioStreamingInterface
	Score     int
	Reason    string
}

// NegotiateBestFormat selects the best audio interface based on preferences
func NegotiateBestFormat(interfaces []*AudioStreamingInterface, pref FormatPreference) (*AudioStreamingInterface, error) {
	if len(interfaces) == 0 {
		return nil, fmt.Errorf("no audio interfaces available")
	}

	scores := make([]FormatScore, 0, len(interfaces))

	for _, iface := range interfaces {
		score := scoreInterface(iface, pref)
		scores = append(scores, score)
	}

	// Sort by score (higher is better)
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].Score > scores[j].Score
	})

	// Return the best match
	best := scores[0]
	if best.Score <= 0 {
		return nil, fmt.Errorf("no suitable audio format found")
	}

	fmt.Printf("Selected format: %s (score: %d)\n", best.Reason, best.Score)
	return best.Interface, nil
}

func scoreInterface(iface *AudioStreamingInterface, pref FormatPreference) FormatScore {
	score := 0
	reasons := []string{}

	// Score format type
	if pref.PreferCompressed {
		if iface.FormatType == 0x02 {
			score += 50
			reasons = append(reasons, "compressed format")
		} else if iface.FormatType == 0x01 && iface.FormatTag != 0x0001 {
			score += 30 // Compressed Type I format
			reasons = append(reasons, "compressed PCM")
		}
	} else {
		if iface.FormatType == 0x01 && iface.FormatTag == 0x0001 {
			score += 50
			reasons = append(reasons, "uncompressed PCM")
		}
	}

	// Score sample rate match
	sampleRateScore := scoreSampleRate(iface.SamplingFreqs, pref.PreferredSampleRate)
	score += sampleRateScore
	if sampleRateScore > 0 {
		reasons = append(reasons, fmt.Sprintf("%dHz", findClosestRate(iface.SamplingFreqs, pref.PreferredSampleRate)))
	}

	// Score channel match
	if int(iface.NrChannels) == pref.PreferredChannels {
		score += 30
		reasons = append(reasons, fmt.Sprintf("%d channels", iface.NrChannels))
	} else if iface.NrChannels > 0 {
		// Partial score for different channel count
		score += 10
		reasons = append(reasons, fmt.Sprintf("%d channels (wanted %d)", iface.NrChannels, pref.PreferredChannels))
	}

	// Score bit depth match
	if int(iface.BitResolution) == pref.PreferredBitsPerSample {
		score += 20
		reasons = append(reasons, fmt.Sprintf("%d-bit", iface.BitResolution))
	} else if iface.BitResolution >= 16 {
		score += 10
		reasons = append(reasons, fmt.Sprintf("%d-bit", iface.BitResolution))
	}

	// For Type II formats, check bitrate
	if iface.FormatType == 0x02 && pref.MaxBitRate > 0 {
		if iface.MaxBitRate <= pref.MaxBitRate {
			score += 10
			reasons = append(reasons, fmt.Sprintf("%d kbps", iface.MaxBitRate))
		}
	}

	// Bonus for having an endpoint
	if iface.EndpointAddress != 0 {
		score += 5
	}

	return FormatScore{
		Interface: iface,
		Score:     score,
		Reason:    fmt.Sprintf("%v", reasons),
	}
}

func scoreSampleRate(available []uint32, preferred uint32) int {
	if len(available) == 0 {
		return 0
	}

	// Check for exact match
	for _, rate := range available {
		if rate == preferred {
			return 50 // Perfect match
		}
	}

	// Find closest rate
	closest := findClosestRate(available, preferred)
	diff := float64(closest) - float64(preferred)
	if preferred > closest {
		diff = float64(preferred) - float64(closest)
	}

	// Score based on how close it is
	percentDiff := (diff / float64(preferred)) * 100
	if percentDiff < 5 {
		return 40 // Very close
	} else if percentDiff < 10 {
		return 30 // Close
	} else if percentDiff < 25 {
		return 20 // Acceptable
	} else if percentDiff < 50 {
		return 10 // Far but usable
	}

	return 5 // Very different
}

func findClosestRate(available []uint32, preferred uint32) uint32 {
	if len(available) == 0 {
		return 0
	}

	closest := available[0]
	minDiff := uint32(^uint32(0))

	for _, rate := range available {
		diff := rate - preferred
		if preferred > rate {
			diff = preferred - rate
		}
		if diff < minDiff {
			minDiff = diff
			closest = rate
		}
	}

	return closest
}

// GetFormatCapabilities returns a summary of all format capabilities
func GetFormatCapabilities(interfaces []*AudioStreamingInterface) string {
	if len(interfaces) == 0 {
		return "No audio formats available"
	}

	// Collect unique capabilities
	sampleRates := make(map[uint32]bool)
	channels := make(map[uint8]bool)
	bitDepths := make(map[uint8]bool)
	formatTypes := make(map[string]bool)

	for _, iface := range interfaces {
		for _, rate := range iface.SamplingFreqs {
			sampleRates[rate] = true
		}
		channels[iface.NrChannels] = true
		bitDepths[iface.BitResolution] = true

		// Format type
		switch iface.FormatType {
		case 0x01:
			if iface.FormatTag == 0x0001 {
				formatTypes["PCM"] = true
			} else {
				formatTypes["Compressed Type I"] = true
			}
		case 0x02:
			formatTypes["Type II (MPEG/AC3)"] = true
		case 0x03:
			formatTypes["Type III (Format Specific)"] = true
		}
	}

	// Build summary
	summary := "Audio Format Capabilities:\n"

	summary += "  Formats: "
	for fmt := range formatTypes {
		summary += fmt + " "
	}
	summary += "\n"

	summary += "  Sample Rates: "
	rates := make([]int, 0, len(sampleRates))
	for rate := range sampleRates {
		rates = append(rates, int(rate))
	}
	sort.Ints(rates)
	for _, rate := range rates {
		summary += fmt.Sprintf("%dHz ", rate)
	}
	summary += "\n"

	summary += "  Channels: "
	for ch := range channels {
		summary += fmt.Sprintf("%d ", ch)
	}
	summary += "\n"

	summary += "  Bit Depths: "
	for bits := range bitDepths {
		summary += fmt.Sprintf("%d-bit ", bits)
	}

	return summary
}
