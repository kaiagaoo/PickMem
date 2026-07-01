package picker

// EstimateTokens returns a crude token count for a set of note bodies:
// ceil(total_chars / 4). Good enough for a running "~T tokens" footer
// hint; not accurate enough to make decisions with. The spec says
// ceil(chars/4), and consistency with the extension matters more than
// tokenizer fidelity — both sides show the same number.
func EstimateTokens(bodies []string) int {
	total := 0
	for _, b := range bodies {
		total += len(b)
	}
	if total == 0 {
		return 0
	}
	// Ceiling division by 4.
	return (total + 3) / 4
}
