// Package logo renders a 4RGED wordmark in a stylized way.
package logo

import (
	"fmt"
	"image/color"
	"math/rand/v2"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/neelworx-cpu/F4RGE-CLI/internal/ui/styles"
)

// letterform represents a letterform. It can be stretched horizontally by
// a given amount via the boolean argument.
type letterform func(bool) string

const diag = `╱`

// Opts are the options for rendering the 4RGED title art.
type Opts struct {
	FieldColor   color.Color // diagonal lines
	TitleColorA  color.Color // left gradient ramp point
	TitleColorB  color.Color // right gradient ramp point
	BrandColor   color.Color // 4RGED meta text color
	VersionColor color.Color // version text color
	Width        int         // width of the rendered logo, used for truncation
	Hyper        bool        // whether it is 4RGED or Hyper 4RGED

	// When true, stretch a random letterform on each render. Has no effect in
	// compact mode. Mainly for testing. In production you will want to cache
	// the stretched letterform to keep the logo from jittering on resize.
	Unstable bool
}

// Render renders the 4RGED logo. Set the argument to true to render the narrow
// version, intended for use in a sidebar.
//
// The compact argument determines whether it renders compact for the sidebar
// or wider for the main pane.
func Render(base lipgloss.Style, version string, compact bool, o Opts) string {
	brand := "4RGED"
	if !o.Hyper {
		brand = " " + brand
	}

	fg := func(c color.Color, s string) string {
		return lipgloss.NewStyle().Foreground(c).Render(s)
	}

	// Title.
	const spacing = 1
	var hyperLetterforms []letterform
	if o.Hyper {
		hyperLetterforms = []letterform{
			LetterH,
			LetterYAlt,
			LetterP,
			LetterE,
			LetterR,
		}
	}
	f4rgedLetterforms := []letterform{
		Letter4,
		LetterR,
		LetterG,
		LetterE,
		LetterD,
	}
	if o.Hyper && !compact {
		f4rgedLetterforms = append(hyperLetterforms, f4rgedLetterforms...)
	}

	stretchIndex := -1 // -1 means no stretching.
	if !compact && o.Unstable {
		// Stretch a random letterform on every render.
		stretchIndex = normalizeStretchIndex(rand.IntN(len(f4rgedLetterforms)), o.Hyper)
	}
	f4rged := renderWord(spacing, stretchIndex, f4rgedLetterforms...)
	if o.Hyper && compact {
		f4rged = renderWord(spacing, stretchIndex, hyperLetterforms...) + "\n" + f4rged
	}
	f4rgedWidth := lipgloss.Width(f4rged)
	b := new(strings.Builder)
	for r := range strings.SplitSeq(f4rged, "\n") {
		fmt.Fprintln(b, styles.ApplyForegroundGrad(base, r, o.TitleColorA, o.TitleColorB))
	}
	f4rged = b.String()

	// Brand and version.
	metaRowGap := 1
	maxVersionWidth := f4rgedWidth - lipgloss.Width(brand) - metaRowGap
	version = ansi.Truncate(version, maxVersionWidth, "…") // truncate version if too long.
	if o.Hyper && compact {
		version += " "
	}
	gap := max(0, f4rgedWidth-lipgloss.Width(brand)-lipgloss.Width(version))
	metaRow := fg(o.BrandColor, brand) + strings.Repeat(" ", gap) + fg(o.VersionColor, version)

	// Join the meta row and big 4RGED title.
	f4rged = strings.TrimSpace(metaRow + "\n" + f4rged)

	// Narrow version. If this is Hyper 4RGED, this is also a stacked version.
	if compact {
		field := fg(o.FieldColor, strings.Repeat(diag, f4rgedWidth))
		return strings.Join([]string{field, field, f4rged, field, ""}, "\n")
	}

	fieldHeight := lipgloss.Height(f4rged)

	// Left field.
	const leftWidth = 6
	leftFieldRow := fg(o.FieldColor, strings.Repeat(diag, leftWidth))
	leftField := new(strings.Builder)
	for range fieldHeight {
		fmt.Fprintln(leftField, leftFieldRow)
	}

	// Right field.
	rightWidth := max(15, o.Width-f4rgedWidth-leftWidth-2) // 2 for the gap.
	const stepDownAt = 0
	rightField := new(strings.Builder)
	for i := range fieldHeight {
		width := rightWidth
		if i >= stepDownAt {
			width = rightWidth - (i - stepDownAt)
		}
		fmt.Fprint(rightField, fg(o.FieldColor, strings.Repeat(diag, width)), "\n")
	}

	// Return the wide version.
	const hGap = " "
	logo := lipgloss.JoinHorizontal(lipgloss.Top, leftField.String(), hGap, f4rged, hGap, rightField.String())
	if o.Width > 0 {
		// Truncate the logo to the specified width.
		lines := strings.Split(logo, "\n")
		for i, line := range lines {
			lines[i] = ansi.Truncate(line, o.Width, "")
		}
		logo = strings.Join(lines, "\n")
	}
	return logo
}

// SmallRender renders a smaller version of the 4RGED logo, suitable for
// smaller windows or sidebar usage.
func SmallRender(t *styles.Styles, width int, o Opts) string {
	name := "4RGED"
	if o.Hyper {
		name = "HYPER 4RGED"
	}
	title := t.Logo.SmallBrand.Render(name)
	remainingWidth := width - lipgloss.Width(title) - 1 // 1 for the space after the name
	if remainingWidth > 0 {
		lines := strings.Repeat("╱", remainingWidth)
		title = fmt.Sprintf("%s %s", title, t.Logo.SmallDiagonals.Render(lines))
	}
	return title
}

func normalizeStretchIndex(index int, hyper bool) int {
	if index < 0 {
		return index
	}
	if !hyper {
		// 4RGED: do not stretch 4, E, or D.
		if index == 0 || index == 3 || index == 4 {
			return 1
		}
		return index
	}
	// HYPER4RGED: do not stretch either E, the 4, or D.
	if index == 3 || index == 5 || index == 8 || index == 9 {
		return 6
	}
	return index
}
